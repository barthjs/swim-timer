<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { webhookPreviewRepository } from '@/api/repositories/webhookPreview';
import type { WebhookEvent, WebhookResponse } from '@/api/types/webhook';
import type { WebhookPreviewResponse } from '@/api/types/webhookPreview';
import { useApiError } from '@/composables/form/useApiError';
import { useCopy } from '@/composables/ui/useCopy';
import { formatMessageBody, type WebhookEventOption } from '@/lib/webhook/webhook';

// Shows a message to a webhook with the current data of an event as it would be sent now, signed with its secret,
// so the receiver can be built without reading the documentation. Nothing is sent.
const props = defineProps<{
  webhook: WebhookResponse | null;
  events: WebhookEventOption[];
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const { t } = useI18n();
const { messageFor } = useApiError();
const { copy } = useCopy();

const event = ref<WebhookEvent | undefined>(undefined);
const preview = ref<WebhookPreviewResponse | null>(null);
const loading = ref(false);
const loadError = ref<unknown>(null);

const eventItems = computed(() =>
  props.events.map((option) => ({
    label: option.label,
    value: option.value,
  })),
);

const body = computed(() => (preview.value ? formatMessageBody(preview.value.body) : ''));

// Counts the loads, so the answer of an earlier event never replaces the one of the event chosen since
let latestLoad = 0;

/** Builds the message of the chosen event. */
async function load(): Promise<void> {
  if (!props.webhook || !event.value) return;
  const current = ++latestLoad;
  loading.value = true;
  loadError.value = null;
  try {
    const result = await webhookPreviewRepository.get(props.webhook.id, event.value);
    if (current === latestLoad) preview.value = result;
  } catch (err) {
    if (current === latestLoad) loadError.value = err;
  } finally {
    if (current === latestLoad) loading.value = false;
  }
}

// Each opening starts with the first event the webhook receives and without the message of the last opening
watch(open, (isOpen) => {
  if (!isOpen) return;
  preview.value = null;
  event.value = props.webhook?.events[0] ?? props.events[0]?.value;
});

// One load per opening or change of the event, also when the opening changes the event
watch([open, event], ([isOpen]) => {
  if (isOpen) void load();
});

/** Copies the body of the message to the clipboard. */
async function copyBody(): Promise<void> {
  await copy(body.value, t('webhook.preview.copied'));
}
</script>

<template>
  <UModal
    v-model:open="open"
    :title="t('webhook.preview.title', { name: webhook?.name ?? '' })"
    :description="t('webhook.preview.description')"
    :ui="{ content: 'sm:max-w-4xl' }"
  >
    <template #body>
      <div class="flex flex-col gap-4">
        <UFormField :label="t('webhook.preview.event')">
          <USelect v-model="event" :items="eventItems" class="w-full sm:w-64" />
        </UFormField>

        <UAlert
          v-if="loadError"
          :title="t('webhook.preview.loadError')"
          :description="messageFor(loadError)"
          icon="i-lucide-circle-alert"
          color="error"
        />
        <USkeleton v-else-if="loading && !preview" class="h-64 w-full" />
        <template v-else-if="preview">
          <div class="flex flex-col gap-1">
            <h3 class="text-highlighted heading-card">
              {{ t('webhook.preview.request') }}
            </h3>
            <code class="text-sm break-all">{{ preview.method }} {{ preview.url }}</code>
          </div>

          <div class="flex flex-col gap-1">
            <h3 class="text-highlighted heading-card">
              {{ t('webhook.preview.headers') }}
            </h3>
            <dl class="sm:grid-cols-label grid grid-cols-1 gap-x-4 gap-y-1 text-sm">
              <template v-for="header in preview.headers" :key="header.name">
                <dt class="text-muted font-mono">{{ header.name }}</dt>
                <dd class="font-mono break-all">{{ header.value }}</dd>
              </template>
            </dl>
          </div>

          <div class="flex flex-col gap-1">
            <div class="flex items-center justify-between gap-2">
              <h3 class="text-highlighted heading-card">
                {{ t('webhook.preview.body') }}
              </h3>
              <UButton
                :label="t('webhook.preview.copy')"
                icon="i-lucide-copy"
                color="neutral"
                variant="outline"
                size="sm"
                @click="copyBody"
              />
            </div>
            <pre class="bg-elevated max-h-96 overflow-auto rounded-md p-3 font-mono text-xs">{{
              body
            }}</pre>
          </div>
        </template>
      </div>
    </template>
    <template #footer>
      <div class="flex w-full justify-end">
        <UButton :label="t('common.ok')" @click="open = false" />
      </div>
    </template>
  </UModal>
</template>
