<script setup lang="ts">
import { useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';

import { webhookRepository } from '@/api/repositories/webhook';
import type { WebhookCreateRequest, WebhookResponse } from '@/api/types/webhook';
import FormModal from '@/components/form/FormModal.vue';
import HelpHint from '@/components/form/HelpHint.vue';
import WebhookSecretField from '@/components/webhook/WebhookSecretField.vue';
import { useEditDialog } from '@/composables/form/useEditDialog';
import { useFormValidation, type FormErrorTarget } from '@/composables/form/useFormValidation';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';
import type { WebhookEventOption } from '@/lib/webhook/webhook';

// Creates a webhook, or edits the given one, in a dialog. Every webhook is saved on its own, right away.
const props = defineProps<{
  // The webhook to edit, null for a new one
  webhook: WebhookResponse | null;
  events: WebhookEventOption[];
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const emit = defineEmits<{
  // created tells a new webhook from an edited one
  saved: [webhook: WebhookResponse, created: boolean];
}>();

// The maximum lengths the server accepts for the text fields of a webhook
const NAME_MAX = 100;
const URL_MAX = 2048;
const SECRET_MAX = 255;

const { t } = useI18n();
const form = useTemplateRef<FormErrorTarget>('form');

// Each opening starts from the edited webhook, or from the defaults of a new one.
// The secret of a saved webhook is never shown again, so its field starts empty
// and keeps the stored secret unless a new one is entered.
const {
  open: openProxy,
  state,
  attemptClose,
  save,
  saving,
} = useEditDialog<WebhookResponse, WebhookCreateRequest, WebhookResponse>({
  open,
  row: () => props.webhook,
  initial: (webhook) => ({
    name: webhook?.name ?? '',
    url: webhook?.url ?? '',
    events: webhook ? [...webhook.events] : props.events.slice(0, 1).map((event) => event.value),
    enabled: webhook?.enabled ?? true,
    secret: '',
    skipTlsVerify: webhook?.skipTlsVerify ?? false,
  }),
  create: async (current) => webhookRepository.create(current),
  update: async (webhook, current) => webhookRepository.update(webhook.id, current),
  success: () => t('webhook.saved'),
  onSaved: (webhook, isEdit) => emit('saved', webhook, !isEdit),
  form,
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
</script>

<template>
  <FormModal
    v-model:open="openProxy"
    :title="webhook ? t('webhook.edit') : t('webhook.new')"
    form-id="webhook-form"
    :submit-label="t('common.save')"
    :loading="saving"
    :ui="{ content: 'sm:max-w-2xl' }"
    @cancel="attemptClose"
  >
    <UForm
      id="webhook-form"
      ref="form"
      :validate-on="FORM_VALIDATE_ON"
      :state="state"
      :validate="validate"
      class="flex flex-col gap-4"
      @submit="void save()"
    >
      <UFormField :label="t('fields.name')" name="name" required>
        <UInput v-model="state.name" :maxlength="NAME_MAX" class="w-full" />
      </UFormField>
      <UFormField :label="t('fields.url')" name="url" required>
        <template #hint>
          <HelpHint :text="t('webhook.urlHint')" />
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
        <UCheckboxGroup v-model="state.events" :items="events" />
      </UFormField>
      <UFormField :label="t('fields.secret')" name="secret">
        <template #hint>
          <HelpHint :text="webhook ? t('webhook.secretKeepHint') : t('webhook.secretHint')" />
        </template>
        <WebhookSecretField
          v-model="state.secret"
          :maxlength="SECRET_MAX"
          :placeholder="
            webhook ? t('webhook.secretKeepPlaceholder') : t('webhook.secretPlaceholder')
          "
        />
      </UFormField>
      <UFormField :label="t('fields.skipTlsVerify')" name="skipTlsVerify">
        <template #hint>
          <HelpHint :text="t('webhook.skipTlsVerifyHint')" />
        </template>
        <USwitch v-model="state.skipTlsVerify" />
      </UFormField>
      <UFormField :label="t('fields.enabled')" name="enabled">
        <USwitch v-model="state.enabled" />
      </UFormField>
    </UForm>
  </FormModal>
</template>
