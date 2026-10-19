<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type { TableColumn } from '@nuxt/ui';

import { webhookDeliveryRepository } from '@/api/repositories/webhookDelivery';
import type { WebhookDeliveryResponse, WebhookResponse } from '@/api/types';
import DataTable from '@/components/table/DataTable.vue';
import TableColumnMenu from '@/components/table/TableColumnMenu.vue';
import TableEmpty from '@/components/table/TableEmpty.vue';
import TableToolbar from '@/components/table/TableToolbar.vue';
import { useTableColumnState } from '@/composables/table/useTableColumnState';
import { useTableView } from '@/composables/table/useTableView';
import { useApiError } from '@/composables/useApiError';
import { useWebhookTexts } from '@/composables/useWebhookTexts';
import { formatWebhookTime, matchesDeliverySearch } from '@/lib/webhook';

const SORTABLE = ['attemptedAt', 'event', 'outcome', 'statusCode', 'durationMs'] as const;

// The colors of the outcomes of a delivery.
const OUTCOME_COLORS = {
  delivered: 'success',
  retry: 'warning',
  deferred: 'info',
  gone: 'error',
  rejected: 'error',
} as const;

// The last deliveries to a webhook since the start. The log holds one page, so the table searches, sorts,
// and pages on the client, and keeps its view in memory, since the table of the webhooks below keeps its in the URL.
const props = defineProps<{
  webhook: WebhookResponse | null;
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const { t, locale } = useI18n();
const { messageFor } = useApiError();
const { eventLabel, problem } = useWebhookTexts();

const columnState = useTableColumnState('webhook-deliveries');

/**
 * answer describes what came back: the reason for a delivery without a response,
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

/** load reads the deliveries of the webhook. */
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
  value: (delivery, id) => delivery[id as (typeof SORTABLE)[number]],
  resetOn: search,
  inUrl: false,
});

const columns = computed<TableColumn<WebhookDeliveryResponse>[]>(() => [
  {
    accessorKey: 'attemptedAt',
    header: t('pages.config.webhook.deliveries.time'),
    enableSorting: true,
    size: 190,
  },
  {
    accessorKey: 'event',
    header: t('pages.config.webhook.deliveries.event'),
    enableSorting: true,
    size: 150,
  },
  {
    accessorKey: 'outcome',
    header: t('pages.config.webhook.deliveries.outcome'),
    enableSorting: true,
    size: 150,
  },
  {
    accessorKey: 'statusCode',
    header: t('pages.config.webhook.deliveries.status'),
    enableSorting: true,
    size: 100,
  },
  {
    accessorKey: 'durationMs',
    header: t('pages.config.webhook.deliveries.duration'),
    enableSorting: true,
    size: 110,
  },
  {
    id: 'response',
    header: t('pages.config.webhook.deliveries.response'),
  },
]);
</script>

<template>
  <UModal
    v-model:open="open"
    :title="t('pages.config.webhook.deliveries.title', { name: webhook?.name ?? '' })"
    :description="t('pages.config.webhook.deliveries.description')"
    :ui="{ content: 'h-[85dvh] sm:max-w-5xl', body: 'flex min-h-0 flex-col' }"
  >
    <!--
      The dialog has a fixed height and the table takes what the toolbar leaves, so its rows scroll inside it
      while the header and the pagination stay in view.
    -->
    <template #body>
      <div class="flex min-h-0 flex-1 flex-col gap-3">
        <TableToolbar
          v-model:search="search"
          :placeholder="t('pages.config.webhook.deliveries.search')"
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
          :title="t('pages.config.webhook.deliveries.loadError')"
          :description="messageFor(loadError)"
          icon="i-lucide-circle-alert"
          color="error"
        />
        <DataTable
          v-else
          class="min-h-0"
          v-model:column-state="columnState"
          v-model:sorting="view.sorting.value"
          v-model:page="view.page.value"
          v-model:limit="view.pageSize.value"
          :data="view.visible.value"
          :total="view.total.value"
          :columns="columns"
          :loading="loading"
          :get-row-id="
            (delivery: WebhookDeliveryResponse) => delivery.messageId + delivery.attemptedAt
          "
        >
          <template #attemptedAt-cell="{ row }">
            {{ formatWebhookTime(row.original.attemptedAt, locale) }}
          </template>

          <template #event-cell="{ row }">
            {{ eventLabel(row.original.event) }}
          </template>

          <template #outcome-cell="{ row }">
            <UBadge
              :label="t(`pages.config.webhook.outcomes.${row.original.outcome}`)"
              :color="OUTCOME_COLORS[row.original.outcome]"
              variant="subtle"
            />
          </template>

          <template #statusCode-cell="{ row }">
            {{ row.original.statusCode || '' }}
          </template>

          <template #durationMs-cell="{ row }">
            {{
              t('pages.config.webhook.deliveries.milliseconds', { value: row.original.durationMs })
            }}
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
              :text="t('pages.config.webhook.deliveries.empty')"
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
