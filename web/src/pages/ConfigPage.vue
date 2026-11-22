<script setup lang="ts">
import { computed, nextTick, onMounted, ref, useTemplateRef, watch, type Component } from 'vue';
import { useI18n } from 'vue-i18n';
import { onBeforeRouteLeave } from 'vue-router';
import type { Form, FormErrorEvent, FormSchema } from '@nuxt/ui';
import { useToast } from '@nuxt/ui/composables';

import { ApiError } from '@/api/client/errors';
import { configRepository } from '@/api/repositories/config';
import type { ConfigSettings } from '@/api/types/config';
import ConfigAppSection from '@/components/config/ConfigAppSection.vue';
import ConfigFileInfo from '@/components/config/ConfigFileInfo.vue';
import ConfigResetButton from '@/components/config/ConfigResetButton.vue';
import ConfigServerSection from '@/components/config/ConfigServerSection.vue';
import ConfigStorageSection from '@/components/config/ConfigStorageSection.vue';
import ConfigThemeSection from '@/components/config/ConfigThemeSection.vue';
import ConfigTimingSection from '@/components/config/ConfigTimingSection.vue';
import ConfigWebhookSection from '@/components/config/ConfigWebhookSection.vue';
import DocsLink from '@/components/system/DocsLink.vue';
import SectionNav from '@/components/ui/SectionNav.vue';
import { useApiError } from '@/composables/form/useApiError';
import { provideConfigForm } from '@/composables/config/useConfigForm';
import { useConfirm } from '@/composables/ui/useConfirm';
import { SCROLL_SPY_ATTRIBUTE, useScrollSpy } from '@/composables/ui/useScrollSpy';
import { useUrlSection } from '@/composables/ui/useUrlSection';
import { useDiscardGuard } from '@/composables/form/useDiscardConfirm';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import { useMutation } from '@/composables/form/useMutation';
import { useSaveShortcut, type SubmitTarget } from '@/composables/form/useSaveShortcut';
import { useUnloadGuard } from '@/composables/ui/useUnloadGuard';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';

const { t } = useI18n();
const { messageFor } = useApiError();
const { confirm } = useConfirm();
const toast = useToast();

const config = provideConfigForm();
const form = useTemplateRef<FormErrorTarget & SubmitTarget & Pick<Form<FormSchema>, 'getErrors'>>(
  'form',
);

const loading = ref(true);
const loadFailed = ref(false);
const loadError = ref<unknown>(null);

/** Reads the configuration from the server. */
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

/** Drops every change and the errors shown for them. */
function resetChanges(): void {
  config.resetChanges();
  form.value?.clear();
}

useUnloadGuard(() => config.isDirty.value);

const restartNoticeOpen = ref(false);

const { mutate: save, pending: saving } = useMutation(
  async () => configRepository.save(config.saveRequest()),
  {
    // The configuration only takes effect after a restart, so saving asks first
    confirm: async () =>
      confirm({
        title: t('pages.config.saveTitle'),
        description: saveDescription(),
        confirmLabel: t('common.save'),
        icon: 'i-lucide-save',
      }),
    form,
    values: config.state,
    onSuccess: (response) => {
      config.applyResponse(response);
      restartNoticeOpen.value = true;
    },
    onError: (err) => {
      // A configuration the server rejects as a whole, such as an invalid list of origins,
      // is reported with the reason of the server, since no single field shows it.
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
 * Explains what saving does and names the changed settings that a flag or an environment variable overrides,
 * since saving them to the config file has no effect.
 */
function saveDescription(): string {
  const description = t('pages.config.saveDescription');
  const overridden = config.overriddenChanges();
  if (overridden.length === 0) return description;
  const names = overridden.map((source) => source.name).join(', ');
  return `${description} ${t('pages.config.saveOverridden', { names })}`;
}

const { kbds: saveKbds } = useSaveShortcut(form, () => config.isDirty.value && !saving.value);

const { confirmDiscard } = useDiscardGuard(config.isDirty);

onBeforeRouteLeave(async () => confirmDiscard());

/** A section of the configuration page, the log settings are part of the app section. */
type ConfigSection = 'app' | 'server' | 'storage' | 'theme' | 'timing' | 'webhook';

/** The groups of the navigation in their order, the meet before the general settings. */
const SECTION_GROUPS = ['meet', 'general'] as const;

// Computed, since the language can change after the page is set up. The sections are sorted by their key,
// the navigation orders them by their group.
const sections = computed(() => [
  {
    value: 'app' as const,
    label: t('pages.config.sections.app'),
    icon: 'i-lucide-info',
    group: 'general' as const,
    docsPage: 'configApp' as const,
  },
  {
    value: 'server' as const,
    label: t('pages.config.sections.server'),
    icon: 'i-lucide-network',
    group: 'general' as const,
    docsPage: 'configServer' as const,
  },
  {
    value: 'storage' as const,
    label: t('pages.config.sections.storage'),
    icon: 'i-lucide-hard-drive',
    group: 'general' as const,
    docsPage: 'configStorage' as const,
  },
  {
    value: 'theme' as const,
    label: t('pages.config.sections.theme'),
    icon: 'i-lucide-palette',
    group: 'general' as const,
    docsPage: 'configTheme' as const,
  },
  {
    value: 'timing' as const,
    label: t('pages.config.sections.timing'),
    icon: 'i-lucide-timer',
    group: 'meet' as const,
    docsPage: 'configTiming' as const,
  },
  {
    value: 'webhook' as const,
    label: t('pages.config.sections.webhook'),
    icon: 'i-lucide-webhook',
    group: 'meet' as const,
    docsPage: 'configWebhook' as const,
  },
]);

const sectionValues = computed<ConfigSection[]>(() =>
  sections.value.map((section) => section.value),
);
// The section in view, which the URL keeps, so a link opens the page at a section
const activeSection = useUrlSection(sectionValues, 'timing');

const sectionsContainer = useTemplateRef<HTMLElement>('sections');
const { scrollTo } = useScrollSpy(
  sectionsContainer,
  () => navSections.value.map((section) => section.value),
  (section) => {
    activeSection.value = section;
  },
);

// A choice in the navigation scrolls to the section, and the scroll spy then marks it
const navSection = computed<ConfigSection>({
  get: () => activeSection.value,
  set: (section) => {
    activeSection.value = section;
    scrollTo(section);
  },
});

// The page opens at the section the URL names, once the form is rendered
watch(sectionsContainer, async (container) => {
  if (!container || activeSection.value === sectionValues.value[0]) return;
  await nextTick();
  scrollTo(activeSection.value, false);
});

/** The fields of each section. */
const SECTION_COMPONENTS: Record<ConfigSection, Component> = {
  app: ConfigAppSection,
  server: ConfigServerSection,
  storage: ConfigStorageSection,
  theme: ConfigThemeSection,
  timing: ConfigTimingSection,
  webhook: ConfigWebhookSection,
};

/**
 * Opens the section of the first field that failed validation,
 * since only the fields of the open section are rendered.
 */
function showFirstError(event: FormErrorEvent): void {
  const section = sectionOf(event.errors[0]?.name);
  if (section) navSection.value = section;
}

// The errors of the form per section, so the navigation shows where a field failed
const sectionErrors = computed(() => {
  const counts: Partial<Record<ConfigSection, number>> = {};
  for (const error of form.value?.getErrors() ?? []) {
    const section = sectionOf(error.name);
    if (section) counts[section] = (counts[section] ?? 0) + 1;
  }
  return counts;
});

const navSections = computed(() =>
  SECTION_GROUPS.flatMap((group) =>
    sections.value
      .filter((section) => section.group === group)
      .map((section) => ({
        ...section,
        changed: config.isSectionDirty(sectionKeys(section.value)),
        errors: sectionErrors.value[section.value],
      })),
  ),
);

/** Returns the top-level keys of the configuration a section edits, the log settings belong to the app. */
function sectionKeys(section: ConfigSection): (keyof ConfigSettings)[] {
  return section === 'app' ? ['app', 'log'] : [section];
}

/** Returns the section of a field by the top-level key of its name. */
function sectionOf(name: string | undefined): ConfigSection | undefined {
  const key = name?.split('.')[0];
  const section = key === 'log' ? 'app' : key;
  return sectionValues.value.find((value) => value === section);
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <!--
      On small screens the badge of unsaved changes moves into a row of its own below the title
      and the reset button only shows its icon, so the title and the buttons keep one row.
      Below md, the select of the sections takes the last row, so it stays at hand while the page scrolls.
      The header sticks above the padding of the panel, so no section shows through above it.
    -->
    <div
      class="bg-default sticky -top-3 z-10 -mx-3 -mt-3 flex flex-wrap items-center gap-2 px-3 pt-3 pb-2 sm:-top-4 sm:-mx-4 sm:-mt-4 sm:px-4 sm:pt-4"
    >
      <div class="mr-auto flex items-center gap-1">
        <h1 class="text-highlighted heading-page">{{ t('pages.config.title') }}</h1>
        <DocsLink page="configApp" />
        <ConfigFileInfo v-if="!loading && !loadFailed" class="ms-3 hidden md:flex" />
      </div>

      <template v-if="!loading && !loadFailed">
        <div v-if="config.isDirty.value" class="order-last basis-full sm:order-0 sm:basis-auto">
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
          <ConfigResetButton />
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

        <SectionNav
          v-model="navSection"
          :sections="navSections"
          class="order-last basis-full md:hidden"
        />
      </template>
    </div>

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
      v-else
      id="config-form"
      ref="form"
      :validate-on="FORM_VALIDATE_ON"
      :state="config.state"
      :validate="config.validate"
      class="flex flex-col gap-3 md:flex-row md:items-start md:gap-4"
      @submit="void save()"
      @error="showFirstError"
    >
      <div class="hidden shrink-0 flex-col gap-3 md:sticky md:top-14 md:flex md:w-48">
        <SectionNav v-model="navSection" :sections="navSections" orientation="vertical" />
      </div>

      <!-- Every section on one page, which scrolls below the sticky header to the section the navigation names -->
      <div ref="sections" class="flex min-w-0 flex-1 flex-col gap-8">
        <section
          v-for="section in navSections"
          :key="section.value"
          v-bind="{ [SCROLL_SPY_ATTRIBUTE]: section.value }"
          :aria-labelledby="`config-${section.value}`"
          class="flex scroll-mt-32 flex-col gap-3 md:scroll-mt-20"
        >
          <div class="flex items-center gap-1">
            <!-- The heading scrolls its section to the top like the navigation -->
            <h2 :id="`config-${section.value}`" class="text-highlighted heading-section">
              <button
                type="button"
                class="hover:text-primary flex items-center gap-2 transition-colors"
                @click="navSection = section.value"
              >
                <UIcon :name="section.icon" class="text-muted size-5" />
                {{ section.label }}
              </button>
            </h2>
            <DocsLink :page="section.docsPage" />
            <ConfigResetButton
              :sections="sectionKeys(section.value)"
              :label="section.label"
              class="ms-auto"
            />
          </div>
          <component :is="SECTION_COMPONENTS[section.value]" />
        </section>
      </div>
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
