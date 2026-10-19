<script setup lang="ts">
import { computed, reactive, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { webhookRepository } from '@/api/repositories/webhook';
import { WebhookEvent, type WebhookCreateRequest, type WebhookResponse } from '@/api/types';
import HelpHint from '@/components/form/HelpHint.vue';
import WebhookSecretField from '@/components/webhook/WebhookSecretField.vue';
import { useFormValidation, type FormErrorTarget } from '@/composables/form/useFormValidation';
import { useMutation } from '@/composables/form/useMutation';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';

// The maximum lengths the server accepts for the text fields of a webhook.
const NAME_MAX = 100;
const URL_MAX = 2048;
const SECRET_MAX = 255;

// Creates a webhook, or edits the given one, in a dialog. Every webhook is saved on its own, right away.
const props = defineProps<{
  // The webhook to edit, null for a new one
  webhook: WebhookResponse | null;
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const emit = defineEmits<{
  // created tells a new webhook from an edited one
  saved: [webhook: WebhookResponse, created: boolean];
}>();

const { t } = useI18n();
const form = useTemplateRef<FormErrorTarget>('form');

const state = reactive<WebhookCreateRequest>({
  name: '',
  url: '',
  events: [WebhookEvent.HeatUpdated],
  enabled: true,
  secret: '',
  skipTlsVerify: false,
});

const eventItems = computed(() => [
  {
    label: t('pages.config.webhook.events.heatUpdated'),
    description: t('pages.config.webhook.events.heatUpdatedHint'),
    value: WebhookEvent.HeatUpdated,
  },
  {
    label: t('pages.config.webhook.events.meetReloaded'),
    description: t('pages.config.webhook.events.meetReloadedHint'),
    value: WebhookEvent.MeetReloaded,
  },
]);

// Each opening starts from the edited webhook, or from the defaults of a new one
watch(open, (isOpen) => {
  if (!isOpen) return;
  state.name = props.webhook?.name ?? '';
  state.url = props.webhook?.url ?? '';
  state.events = props.webhook ? [...props.webhook.events] : [WebhookEvent.HeatUpdated];
  state.enabled = props.webhook?.enabled ?? true;
  state.secret = props.webhook?.secret ?? '';
  state.skipTlsVerify = props.webhook?.skipTlsVerify ?? false;
});

const { validate } = useFormValidation<WebhookCreateRequest>(() => ({
  name: {
    required: 'required',
    max: NAME_MAX,
  },
  url: {
    required: 'required',
    max: URL_MAX,
  },
  events: {
    required: 'required',
  },
  secret: {
    max: SECRET_MAX,
  },
}));

const { mutate: save, pending: saving } = useMutation(
  () =>
    props.webhook
      ? webhookRepository.update(props.webhook.id, state)
      : webhookRepository.create(state),
  {
    form,
    values: state,
    success: t('pages.config.webhook.saved'),
    onSuccess: (webhook) => {
      open.value = false;
      emit('saved', webhook, !props.webhook);
    },
  },
);
</script>

<template>
  <UModal
    v-model:open="open"
    :title="webhook ? t('pages.config.webhook.edit') : t('pages.config.webhook.new')"
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #body>
      <UForm
        :validate-on="FORM_VALIDATE_ON"
        id="webhook-form"
        ref="form"
        :state="state"
        :validate="validate"
        class="flex flex-col gap-4"
        @submit="void save()"
      >
        <UFormField :label="t('fields.name')" name="name" required>
          <UInput v-model="state.name" :maxlength="NAME_MAX" autofocus class="w-full" />
        </UFormField>
        <UFormField :label="t('fields.url')" name="url" required>
          <template #hint>
            <HelpHint :text="t('pages.config.webhook.urlHint')" />
          </template>
          <UInput
            v-model="state.url"
            :maxlength="URL_MAX"
            type="url"
            placeholder="https://example.com/webhook"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.events')" name="events" required>
          <UCheckboxGroup v-model="state.events" :items="eventItems" />
        </UFormField>
        <UFormField :label="t('fields.secret')" name="secret">
          <template #hint>
            <HelpHint :text="t('pages.config.webhook.secretHint')" />
          </template>
          <WebhookSecretField
            v-model="state.secret"
            :maxlength="SECRET_MAX"
            :placeholder="t('pages.config.webhook.secretPlaceholder')"
          />
        </UFormField>
        <UFormField :label="t('fields.skipTlsVerify')" name="skipTlsVerify">
          <template #hint>
            <HelpHint :text="t('pages.config.webhook.skipTlsVerifyHint')" />
          </template>
          <USwitch v-model="state.skipTlsVerify" />
        </UFormField>
        <UFormField :label="t('fields.enabled')" name="enabled">
          <USwitch v-model="state.enabled" />
        </UFormField>
      </UForm>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton
          :label="t('common.cancel')"
          color="neutral"
          variant="outline"
          @click="open = false"
        />
        <UButton
          type="submit"
          form="webhook-form"
          :label="t('common.save')"
          icon="i-lucide-save"
          :loading="saving"
        />
      </div>
    </template>
  </UModal>
</template>
