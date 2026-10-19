<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import type { DropdownMenuItem, TableColumn } from '@nuxt/ui';
import { useToast } from '@nuxt/ui/composables';

import { webhookRepository } from '@/api/repositories/webhook';
import type { WebhookDeliveryResponse, WebhookResponse } from '@/api/types';
import DataTable from '@/components/table/DataTable.vue';
import TableColumnMenu from '@/components/table/TableColumnMenu.vue';
import TableEmpty from '@/components/table/TableEmpty.vue';
import TableToolbar from '@/components/table/TableToolbar.vue';
import WebhookDeliveriesModal from '@/components/webhook/WebhookDeliveriesModal.vue';
import WebhookFormModal from '@/components/webhook/WebhookFormModal.vue';
import WebhookPreviewModal from '@/components/webhook/WebhookPreviewModal.vue';
import WebhookSecretModal from '@/components/webhook/WebhookSecretModal.vue';
import { useMutation } from '@/composables/form/useMutation';
import { useTableColumnState } from '@/composables/table/useTableColumnState';
import { useTableView } from '@/composables/table/useTableView';
import { useApiError } from '@/composables/useApiError';
import { useWebhookTexts } from '@/composables/useWebhookTexts';
import { useConfirm } from '@/composables/ui/useConfirm';
import { TABLE_ACTIONS_COLUMN } from '@/lib/table/columns';
import {
  formatWebhookTime,
  matchesWebhookSearch,
  WEBHOOK_HEALTH_ORDER,
  webhookHealth,
  type WebhookHealth,
} from '@/lib/webhook';

const SORTABLE = ['name', 'url', 'status'] as const;

// The colors of the states of a webhook.
const HEALTH_COLORS: Record<WebhookHealth, 'neutral' | 'info' | 'success' | 'error'> = {
  disabled: 'neutral',
  waiting: 'info',
  ok: 'success',
  failing: 'error',
};

// The webhooks the live data is delivered to. A pool has a handful of them, so the table searches, sorts,
// and pages on the client. Each webhook is saved, tested, and deleted on its own, right away,
// independent of the configuration around it.
const { t, locale } = useI18n();
const { messageFor } = useApiError();
const { eventLabel, problem } = useWebhookTexts();
const { confirm } = useConfirm();
const toast = useToast();

const columnState = useTableColumnState('webhooks');

const webhooks = ref<WebhookResponse[]>([]);
const loading = ref(true);
const loadError = ref<unknown>(null);
const search = ref('');

/** load reads the webhooks with the state of their deliveries. */
async function load(): Promise<void> {
  loading.value = true;
  loadError.value = null;
  try {
    webhooks.value = await webhookRepository.list();
  } catch (err) {
    loadError.value = err;
  } finally {
    loading.value = false;
  }
}

onMounted(load);

const filtered = computed(() =>
  webhooks.value.filter((webhook) => matchesWebhookSearch(webhook, search.value)),
);

const view = useTableView(filtered, {
  sortable: SORTABLE,
  value: (webhook, id) =>
    id === 'status' ? WEBHOOK_HEALTH_ORDER[webhookHealth(webhook)] : webhook[id as 'name' | 'url'],
  resetOn: search,
});

const formOpen = ref(false);
const editedWebhook = ref<WebhookResponse | null>(null);
const createdWebhook = ref<WebhookResponse | null>(null);
const secretOpen = ref(false);
const deliveriesOpen = ref(false);
const deliveriesWebhook = ref<WebhookResponse | null>(null);
const previewOpen = ref(false);
const previewWebhook = ref<WebhookResponse | null>(null);

/** openForm shows the dialog for the given webhook, or for a new one. */
function openForm(webhook: WebhookResponse | null): void {
  editedWebhook.value = webhook;
  formOpen.value = true;
}

/** onSaved reloads the table and shows the secret of a new webhook, so it can be handed to the receiver. */
async function onSaved(webhook: WebhookResponse, created: boolean): Promise<void> {
  if (created) {
    createdWebhook.value = webhook;
    secretOpen.value = true;
  }
  await load();
}

/** openDeliveries shows the last deliveries to a webhook. */
function openDeliveries(webhook: WebhookResponse): void {
  deliveriesWebhook.value = webhook;
  deliveriesOpen.value = true;
}

/** openPreview shows a message to a webhook as it would be sent now. */
function openPreview(webhook: WebhookResponse): void {
  previewWebhook.value = webhook;
  previewOpen.value = true;
}

const { mutate: sendTest } = useMutation(
  (webhook: WebhookResponse) => webhookRepository.test(webhook.id),
  {
    onSuccess: async (delivery) => {
      showTestResult(delivery);
      await load();
    },
  },
);

/** showTestResult reports how the test delivery ended. */
function showTestResult(delivery: WebhookDeliveryResponse): void {
  if (delivery.outcome === 'delivered') {
    toast.add({
      title: t('pages.config.webhook.testDelivered'),
      icon: 'i-lucide-check-circle',
      color: 'success',
    });
    return;
  }
  toast.add({
    title: t('pages.config.webhook.testFailed'),
    description: problem(delivery.errorCode, delivery.statusCode),
    icon: 'i-lucide-circle-alert',
    color: 'error',
  });
}

const { mutate: deleteWebhook } = useMutation(
  (webhook: WebhookResponse) => webhookRepository.delete(webhook.id),
  {
    success: (_, webhook) =>
      t('pages.config.webhook.deleted', {
        name: webhook.name,
      }),
    onSuccess: load,
  },
);

/** confirmDelete deletes a webhook after a confirmation. */
async function confirmDelete(webhook: WebhookResponse): Promise<void> {
  const confirmed = await confirm({
    title: t('pages.config.webhook.deleteTitle'),
    description: t('pages.config.webhook.deleteDescription', {
      name: webhook.name,
    }),
    confirmLabel: t('common.delete'),
    color: 'error',
    icon: 'i-lucide-trash-2',
  });
  if (confirmed) void deleteWebhook(webhook);
}

/** rowMenuItems returns the further actions of a webhook. */
function rowMenuItems(webhook: WebhookResponse): DropdownMenuItem[] {
  return [
    {
      label: t('pages.config.webhook.testAction'),
      icon: 'i-lucide-send',
      onSelect: () => void sendTest(webhook),
    },
    {
      label: t('pages.config.webhook.previewAction'),
      icon: 'i-lucide-eye',
      onSelect: () => openPreview(webhook),
    },
    {
      label: t('pages.config.webhook.deliveriesAction'),
      icon: 'i-lucide-history',
      onSelect: () => openDeliveries(webhook),
    },
    {
      label: t('common.delete'),
      icon: 'i-lucide-trash-2',
      color: 'error',
      onSelect: () => void confirmDelete(webhook),
    },
  ];
}

/** healthTooltip explains the state of a webhook, with the last error of a failing one. */
function healthTooltip(webhook: WebhookResponse): string {
  const { status } = webhook;
  if (webhookHealth(webhook) === 'failing' && status.lastFailedAt) {
    return t('pages.config.webhook.failingSince', {
      error: problem(status.lastErrorCode, status.lastStatusCode),
      count: status.consecutiveFailures,
      time: formatWebhookTime(status.lastFailedAt, locale.value),
    });
  }
  if (status.lastDeliveredAt) {
    return t('pages.config.webhook.lastDelivered', {
      time: formatWebhookTime(status.lastDeliveredAt, locale.value),
    });
  }
  return t('pages.config.webhook.noDelivery');
}

const columns = computed<TableColumn<WebhookResponse>[]>(() => [
  {
    accessorKey: 'name',
    header: t('fields.name'),
    enableSorting: true,
    size: 180,
  },
  {
    accessorKey: 'url',
    header: t('fields.url'),
    enableSorting: true,
  },
  {
    accessorKey: 'events',
    header: t('fields.events'),
    size: 240,
  },
  {
    id: 'status',
    header: t('pages.config.webhook.status'),
    enableSorting: true,
    size: 140,
  },
  {
    id: TABLE_ACTIONS_COLUMN,
    size: 100,
  },
]);
</script>

<template>
  <div class="flex flex-col gap-3 lg:min-h-0 lg:flex-1">
    <TableToolbar
      v-model:search="search"
      :placeholder="t('pages.config.webhook.search')"
      :search-shortcut="['meta', 'f']"
      :can-reset="!!search"
      :loading="loading"
      :on-reload="load"
      @reset="search = ''"
    >
      <template #actions>
        <TableColumnMenu v-model:column-state="columnState" :columns="columns" />
        <UButton
          :label="t('pages.config.webhook.new')"
          icon="i-lucide-plus"
          @click="openForm(null)"
        />
      </template>
    </TableToolbar>

    <UAlert
      v-if="loadError"
      :title="t('pages.config.webhook.loadError')"
      :description="messageFor(loadError)"
      icon="i-lucide-circle-alert"
      color="error"
    />
    <DataTable
      v-else
      class="lg:min-h-80"
      v-model:column-state="columnState"
      v-model:sorting="view.sorting.value"
      v-model:page="view.page.value"
      v-model:limit="view.pageSize.value"
      :data="view.visible.value"
      :total="view.total.value"
      :columns="columns"
      :loading="loading"
      :get-row-id="(webhook: WebhookResponse) => webhook.id"
    >
      <template #events-cell="{ row }">
        <div class="flex flex-wrap gap-1">
          <UBadge
            v-for="event in row.original.events"
            :key="event"
            :label="eventLabel(event)"
            color="neutral"
            variant="outline"
          />
        </div>
      </template>

      <template #status-cell="{ row }">
        <UTooltip :text="healthTooltip(row.original)">
          <UBadge
            :label="t(`pages.config.webhook.health.${webhookHealth(row.original)}`)"
            :color="HEALTH_COLORS[webhookHealth(row.original)]"
            variant="subtle"
          />
        </UTooltip>
      </template>

      <template #actions-cell="{ row }">
        <div class="flex items-center justify-end gap-1">
          <UTooltip :text="t('pages.config.webhook.editAction')">
            <UButton
              icon="i-lucide-pencil"
              color="neutral"
              variant="ghost"
              size="sm"
              :aria-label="t('pages.config.webhook.editAction')"
              @click="openForm(row.original)"
            />
          </UTooltip>
          <UDropdownMenu :items="rowMenuItems(row.original)" :content="{ align: 'end' }">
            <UButton
              icon="i-lucide-ellipsis-vertical"
              color="neutral"
              variant="ghost"
              size="sm"
              :aria-label="t('table.rowActions')"
            />
          </UDropdownMenu>
        </div>
      </template>

      <template #empty>
        <TableEmpty
          icon="i-lucide-webhook"
          :text="t('pages.config.webhook.empty')"
          :filtered="!!search"
          @reset="search = ''"
        />
      </template>
    </DataTable>

    <WebhookFormModal v-model:open="formOpen" :webhook="editedWebhook" @saved="onSaved" />
    <WebhookSecretModal v-model:open="secretOpen" :webhook="createdWebhook" />
    <WebhookDeliveriesModal v-model:open="deliveriesOpen" :webhook="deliveriesWebhook" />
    <WebhookPreviewModal v-model:open="previewOpen" :webhook="previewWebhook" />
  </div>
</template>
