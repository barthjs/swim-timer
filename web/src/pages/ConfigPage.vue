<script setup lang="ts">
import { computed, onMounted, ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router';
import type { FormErrorEvent } from '@nuxt/ui';
import { useToast } from '@nuxt/ui/composables';

import { ApiError } from '@/api/errors';
import { configRepository } from '@/api/repositories/config';
import ConfigAppSection from '@/components/config/ConfigAppSection.vue';
import ConfigFileInfo from '@/components/config/ConfigFileInfo.vue';
import ConfigResetMenu from '@/components/config/ConfigResetMenu.vue';
import ConfigServerSection from '@/components/config/ConfigServerSection.vue';
import ConfigStorageSection from '@/components/config/ConfigStorageSection.vue';
import ConfigThemeSection from '@/components/config/ConfigThemeSection.vue';
import ConfigTimingSection from '@/components/config/ConfigTimingSection.vue';
import ConfigWebhookSection from '@/components/config/ConfigWebhookSection.vue';
import DocsLink from '@/components/system/DocsLink.vue';
import { useApiError } from '@/composables/useApiError';
import { provideConfigForm } from '@/composables/useConfigForm';
import { useConfirm } from '@/composables/ui/useConfirm';
import { useDiscardGuard } from '@/composables/form/useDiscardConfirm';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import { useMutation } from '@/composables/form/useMutation';
import { useSaveShortcut, type SubmitTarget } from '@/composables/form/useSaveShortcut';
import { useUnloadGuard } from '@/composables/useUnloadGuard';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';

const { t } = useI18n();
const { messageFor } = useApiError();
const { confirm } = useConfirm();
const toast = useToast();

const config = provideConfigForm();
const form = useTemplateRef<FormErrorTarget & SubmitTarget>('form');

const loading = ref(true);
const loadFailed = ref(false);
const loadError = ref<unknown>(null);

/** load reads the configuration from the server. */
async function load(): Promise<void> {
  loading.value = true;
  loadFailed.value = false;
  loadError.value = null;
  try {
    config.applyResponse(await configRepository.get());
  } catch (err) {
    loadFailed.value = true;
    loadError.value = err;
  } finally {
    loading.value = false;
  }
}

onMounted(load);

/** resetChanges drops every change and the errors shown for them. */
function resetChanges(): void {
  config.resetChanges();
  form.value?.clear();
}

useUnloadGuard(() => config.isDirty.value);

const restartNoticeOpen = ref(false);

const { mutate: persist, pending: saving } = useMutation(
  () => configRepository.save(config.saveRequest()),
  {
    form,
    values: config.state,
    onSuccess: (response) => {
      config.applyResponse(response);
      restartNoticeOpen.value = true;
    },
    onError: (err) => {
      // A configuration the server rejects as a whole, such as an invalid list of origins, is reported
      // with the reason of the server, since no single field shows it.
      if (!(err instanceof ApiError) || err.code !== 'bad_request') return false;
      toast.add({
        title: t('pages.config.saveError'),
        description: err.message,
        icon: 'i-lucide-circle-alert',
        color: 'error',
        duration: 0,
      });
      return true;
    },
  },
);

/**
 * saveDescription explains what saving does and names the changed settings that a flag
 * or an environment variable overrides, since saving them to the config file has no effect.
 */
function saveDescription(): string {
  const description = t('pages.config.saveDescription');
  const overridden = config.overriddenChanges();
  if (overridden.length === 0) return description;
  const names = overridden.map((source) => source.name).join(', ');
  return `${description} ${t('pages.config.saveOverridden', { names })}`;
}

/** save writes the configuration after a confirmation, since it only takes effect after a restart. */
async function save(): Promise<void> {
  const confirmed = await confirm({
    title: t('pages.config.saveTitle'),
    description: saveDescription(),
    confirmLabel: t('common.save'),
    icon: 'i-lucide-save',
  });
  if (confirmed) void persist();
}

const { kbds: saveKbds } = useSaveShortcut(form, () => config.isDirty.value && !saving.value);

const { confirmDiscard } = useDiscardGuard(config.isDirty);

onBeforeRouteLeave(() => confirmDiscard());

// Computed, since the language can change after the page is set up
const tabs = computed(() => [
  {
    label: t('pages.config.sections.app'),
    icon: 'i-lucide-info',
    slot: 'app' as const,
    value: 'app',
    docsPage: 'configApp' as const,
  },
  {
    label: t('pages.config.sections.timing'),
    icon: 'i-lucide-timer',
    slot: 'timing' as const,
    value: 'timing',
    docsPage: 'configTiming' as const,
  },
  {
    label: t('pages.config.sections.server'),
    icon: 'i-lucide-network',
    slot: 'server' as const,
    value: 'server',
    docsPage: 'configServer' as const,
  },
  {
    label: t('pages.config.sections.webhook'),
    icon: 'i-lucide-webhook',
    slot: 'webhook' as const,
    value: 'webhook',
    docsPage: 'configWebhook' as const,
  },
  {
    label: t('pages.config.sections.storage'),
    icon: 'i-lucide-hard-drive',
    slot: 'storage' as const,
    value: 'storage',
    docsPage: 'configStorage' as const,
  },
  {
    label: t('pages.config.sections.theme'),
    icon: 'i-lucide-palette',
    slot: 'theme' as const,
    value: 'theme',
    docsPage: 'configTheme' as const,
  },
]);

const route = useRoute();
const router = useRouter();
const tabValues = tabs.value.map((tab) => tab.value);

/** tabFromQuery returns the tab named in the URL, or the first tab. */
function tabFromQuery(): string {
  const query = route.query.tab;
  return typeof query === 'string' && tabValues.includes(query)
    ? query
    : (tabs.value[0]?.value ?? 'app');
}

const activeTab = ref(tabFromQuery());

/** activeSections returns the configuration sections of the open tab. The log settings are part of the app tab. */
const activeSections = computed(() =>
  activeTab.value === 'app' ? ['app', 'log'] : [activeTab.value],
);
const activeLabel = computed(
  () => tabs.value.find((tab) => tab.value === activeTab.value)?.label ?? '',
);
const docsPage = computed(
  () => tabs.value.find((tab) => tab.value === activeTab.value)?.docsPage ?? 'configApp',
);

watch(activeTab, (tab) => {
  if (route.query.tab === tab) return;
  void router.push({
    query: {
      ...route.query,
      tab,
    },
  });
});

watch(
  () => route.query.tab,
  () => {
    activeTab.value = tabFromQuery();
  },
);

/**
 * showFirstError opens the tab of the first field that failed validation,
 * since the tabs only render the fields of the open one. The log settings are part of the app tab.
 */
function showFirstError(event: FormErrorEvent): void {
  const section = event.errors[0]?.name?.split('.')[0];
  if (!section) return;
  const tab = section === 'log' ? 'app' : section;
  if (tabValues.includes(tab)) activeTab.value = tab;
}
</script>

<template>
  <!-- From lg up the webhook tab fills the viewport, so its table reaches down to the footer. -->
  <div class="flex flex-col gap-4 lg:min-h-0 lg:flex-1">
    <!--
      On small screens the badge of unsaved changes moves into a row of its own below the title
      and the reset button only shows its icon, so the title and the buttons keep one row.
    -->
    <div class="flex flex-wrap items-center gap-2">
      <div class="mr-auto flex items-center gap-1">
        <h1 class="text-highlighted text-xl font-semibold">{{ t('pages.config.title') }}</h1>
        <DocsLink :page="docsPage" />
      </div>

      <template v-if="!loading && !loadFailed">
        <div v-if="config.isDirty.value" class="order-last basis-full sm:order-none sm:basis-auto">
          <UBadge
            icon="i-lucide-circle-alert"
            color="warning"
            variant="solid"
            class="flex h-8 w-fit items-center px-3"
          >
            {{ t('pages.config.unsavedChanges') }}
          </UBadge>
        </div>

        <div class="flex items-center gap-2">
          <UButton
            v-if="config.isDirty.value"
            :label="t('common.reset')"
            :aria-label="t('common.reset')"
            icon="i-lucide-rotate-ccw"
            color="neutral"
            variant="outline"
            :ui="{ label: 'hidden sm:inline' }"
            @click="resetChanges"
          />
          <ConfigResetMenu :sections="activeSections" :label="activeLabel" />
          <UTooltip :text="t('common.save')" :kbds="saveKbds">
            <UButton
              type="submit"
              form="config-form"
              :label="t('common.save')"
              icon="i-lucide-save"
              :disabled="!config.isDirty.value"
              :loading="saving"
            />
          </UTooltip>
        </div>
      </template>
    </div>

    <ConfigFileInfo v-if="!loading && !loadFailed" class="shrink-0" />

    <div v-if="loading" class="flex flex-col gap-4">
      <USkeleton class="h-10 w-full" />
      <USkeleton class="h-64 w-full" />
    </div>

    <UAlert
      v-else-if="loadFailed"
      :title="t('pages.config.loadError')"
      :description="messageFor(loadError)"
      icon="i-lucide-circle-alert"
      color="error"
    />

    <UForm
      :validate-on="FORM_VALIDATE_ON"
      v-else
      id="config-form"
      ref="form"
      :state="config.state"
      :validate="config.validate"
      class="space-y-4 lg:flex lg:min-h-0 lg:flex-1 lg:flex-col"
      @submit="void save()"
      @error="showFirstError"
    >
      <!-- Small screens choose the section from a select, so the tabs only show from sm up. -->
      <div class="mb-4 sm:hidden">
        <USelect v-model="activeTab" :items="tabs" class="w-full" />
      </div>

      <UTabs
        v-model="activeTab"
        :items="tabs"
        class="w-full lg:min-h-0 lg:flex-1"
        :ui="{ list: 'hidden sm:flex', content: 'lg:flex lg:min-h-0 lg:flex-1 lg:flex-col' }"
      >
        <template #app>
          <ConfigAppSection />
        </template>
        <template #timing>
          <ConfigTimingSection />
        </template>
        <template #server>
          <ConfigServerSection />
        </template>
        <template #webhook>
          <ConfigWebhookSection />
        </template>
        <template #storage>
          <ConfigStorageSection />
        </template>
        <template #theme>
          <ConfigThemeSection />
        </template>
      </UTabs>
    </UForm>

    <UModal
      v-model:open="restartNoticeOpen"
      :title="t('pages.config.restartTitle')"
      :description="t('pages.config.restartDescription')"
    >
      <template #footer>
        <div class="flex w-full justify-end">
          <UButton :label="t('common.ok')" @click="restartNoticeOpen = false" />
        </div>
      </template>
    </UModal>
  </div>
</template>
