<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type { TableColumn } from '@nuxt/ui';

import { webhookDeliveryRepository } from '@/api/repositories/webhookDelivery';
import type { WebhookResponse } from '@/api/types/webhook';
import type { WebhookDeliveryResponse } from '@/api/types/webhookDelivery';
import DataTable from '@/components/table/DataTable.vue';
import TableColumnMenu from '@/components/table/TableColumnMenu.vue';
import TableEmpty from '@/components/table/TableEmpty.vue';
import TableToolbar from '@/components/table/TableToolbar.vue';
import { useTableColumnState } from '@/composables/table/useTableColumnState';
import { useTableView } from '@/composables/table/useTableView';
import { useApiError } from '@/composables/form/useApiError';
import { useWebhookTexts } from '@/composables/webhook/useWebhookTexts';
import { useDates } from '@/composables/ui/useDates';
import { matchesDeliverySearch, type WebhookEventOption } from '@/lib/webhook/webhook';

// The last deliveries to a webhook since the start. The log holds one page, so the table searches, sorts,
// and pages on the client, and keeps its view in memory, since the table of the webhooks below keeps its own in the URL.
const props = defineProps<{
  webhook: WebhookResponse | null;
  events: WebhookEventOption[];
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

// The columns the deliveries can be sorted by
const SORTABLE = ['attemptedAt', 'event', 'outcome', 'statusCode', 'durationMs'] as const;

const OUTCOME_COLORS = {
  delivered: 'success',
  retry: 'warning',
  deferred: 'info',
  gone: 'error',
  rejected: 'error',
} as const;

const { t } = useI18n();
const { formatDateTimeNumeric } = useDates();
const { messageFor } = useApiError();
const { eventLabel, problem } = useWebhookTexts(() => props.events);

const columnState = useTableColumnState('webhook-deliveries');

/**
 * Describes what came back: the reason for a delivery without a response,
 * or the start of the response body.
 */
function answer(delivery: WebhookDeliveryResponse): string {
  if (delivery.errorCode) return problem(delivery.errorCode, delivery.statusCode);
  return delivery.responseBody;
}

const deliveries = ref<WebhookDeliveryResponse[]>([]);
const loading = ref(false);
const loadError = ref<unknown>(null);
const search = ref('');

/** Reads the deliveries of the webhook. */
async function load(): Promise<void> {
  if (!props.webhook) return;
  loading.value = true;
  loadError.value = null;
  try {
    deliveries.value = await webhookDeliveryRepository.list(props.webhook.id);
  } catch (err) {
    loadError.value = err;
  } finally {
    loading.value = false;
  }
}

// Each opening starts without the rows of the last opened webhook
watch(open, (isOpen) => {
  if (!isOpen) return;
  search.value = '';
  deliveries.value = [];
  void load();
});

const filtered = computed(() =>
  deliveries.value.filter((delivery) => matchesDeliverySearch(delivery, search.value)),
);

const view = useTableView(filtered, {
  sortable: SORTABLE,
  value: (delivery, id) => delivery[id],
  resetOn: search,
  inUrl: false,
});

const columns = computed<TableColumn<WebhookDeliveryResponse>[]>(() => [
  {
    accessorKey: 'event',
    header: t('webhook.deliveries.event'),
    enableSorting: true,
  },
  {
    accessorKey: 'outcome',
    header: t('webhook.deliveries.outcome'),
    enableSorting: true,
  },
  {
    accessorKey: 'statusCode',
    header: t('webhook.deliveries.status'),
    enableSorting: true,
  },
  {
    accessorKey: 'durationMs',
    meta: {
      numeric: true,
    },
    header: t('webhook.deliveries.duration'),
    enableSorting: true,
  },
  {
    id: 'response',
    meta: {
      grow: true,
    },
    header: t('webhook.deliveries.response'),
  },
  {
    accessorKey: 'attemptedAt',
    header: t('webhook.deliveries.time'),
    enableSorting: true,
  },
]);
</script>

<template>
  <UModal
    v-model:open="open"
    :title="t('webhook.deliveries.title', { name: webhook?.name ?? '' })"
    :description="t('webhook.deliveries.description')"
    :ui="{ content: 'h-dialog sm:max-w-5xl', body: 'flex min-h-0 flex-col' }"
  >
    <!--
      The dialog has a fixed height and the table takes what the toolbar leaves, so its rows scroll inside it
      while the header and the pagination stay in view.
    -->
    <template #body>
      <div class="flex min-h-0 flex-1 flex-col gap-3">
        <TableToolbar
          v-model:search="search"
          :placeholder="t('webhook.deliveries.search')"
          :can-reset="!!search"
          :loading="loading"
          :on-reload="load"
          @reset="search = ''"
        >
          <template #actions>
            <TableColumnMenu v-model:column-state="columnState" :columns="columns" />
          </template>
        </TableToolbar>

        <UAlert
          v-if="loadError"
          :title="t('webhook.deliveries.loadError')"
          :description="messageFor(loadError)"
          icon="i-lucide-circle-alert"
          color="error"
        />
        <DataTable
          v-else
          v-model:column-state="columnState"
          v-model:sorting="view.sorting.value"
          v-model:page="view.page.value"
          v-model:limit="view.pageSize.value"
          class="min-h-0"
          :data="view.visible.value"
          :total="view.total.value"
          :columns="columns"
          :loading="loading"
          :get-row-id="
            (delivery: WebhookDeliveryResponse) => delivery.messageId + delivery.attemptedAt
          "
        >
          <template #attemptedAt-cell="{ row }">
            {{ formatDateTimeNumeric(row.original.attemptedAt) }}
          </template>

          <template #event-cell="{ row }">
            {{ eventLabel(row.original.event) }}
          </template>

          <template #outcome-cell="{ row }">
            <UBadge
              :label="t(`webhook.outcomes.${row.original.outcome}`)"
              :color="OUTCOME_COLORS[row.original.outcome]"
              variant="subtle"
            />
          </template>

          <template #statusCode-cell="{ row }">
            {{ row.original.statusCode || '' }}
          </template>

          <template #durationMs-cell="{ row }">
            {{ t('webhook.deliveries.milliseconds', { value: row.original.durationMs }) }}
          </template>

          <template #response-cell="{ row }">
            <span
              class="block truncate text-xs"
              :class="{ 'font-mono': !row.original.errorCode }"
              :title="row.original.error || row.original.responseBody"
            >
              {{ answer(row.original) }}
            </span>
          </template>

          <template #empty>
            <TableEmpty
              icon="i-lucide-history"
              :text="t('webhook.deliveries.empty')"
              :filtered="!!search"
              @reset="search = ''"
            />
          </template>
        </DataTable>
      </div>
    </template>
    <template #footer>
      <div class="flex w-full justify-end">
        <UButton :label="t('common.ok')" @click="open = false" />
      </div>
    </template>
  </UModal>
</template>
