<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import type { DropdownMenuItem, TableColumn } from '@nuxt/ui';
import { useToast } from '@nuxt/ui/composables';

import { webhookRepository } from '@/api/repositories/webhook';
import type { WebhookResponse } from '@/api/types/webhook';
import type { WebhookDeliveryResponse } from '@/api/types/webhookDelivery';
import DataTable from '@/components/table/DataTable.vue';
import TableColumnMenu from '@/components/table/TableColumnMenu.vue';
import TableEmpty from '@/components/table/TableEmpty.vue';
import TableToolbar from '@/components/table/TableToolbar.vue';
import TableRowAction from '@/components/table/TableRowAction.vue';
import TableRowActions from '@/components/table/TableRowActions.vue';
import WebhookDeliveriesModal from '@/components/webhook/WebhookDeliveriesModal.vue';
import WebhookFormModal from '@/components/webhook/WebhookFormModal.vue';
import WebhookPreviewModal from '@/components/webhook/WebhookPreviewModal.vue';
import WebhookSecretModal from '@/components/webhook/WebhookSecretModal.vue';
import { useMutation } from '@/composables/form/useMutation';
import { useTableColumnState } from '@/composables/table/useTableColumnState';
import { useTableView } from '@/composables/table/useTableView';
import { useApiError } from '@/composables/form/useApiError';
import { useWebhookTexts } from '@/composables/webhook/useWebhookTexts';
import { useDeleteConfirm } from '@/composables/ui/useDeleteConfirm';
import { useDates } from '@/composables/ui/useDates';
import { TABLE_ACTIONS_COLUMN } from '@/lib/table/columns';
import {
  matchesWebhookSearch,
  WEBHOOK_HEALTH_ORDER,
  webhookHealth,
  type WebhookEventOption,
  type WebhookHealth,
} from '@/lib/webhook/webhook';

const props = defineProps<{
  events: WebhookEventOption[];
}>();

// The columns the webhooks can be sorted by
const SORTABLE = ['name', 'url', 'status'] as const;

const HEALTH_COLORS: Record<WebhookHealth, 'neutral' | 'info' | 'success' | 'error'> = {
  disabled: 'neutral',
  waiting: 'info',
  ok: 'success',
  failing: 'error',
};

// The webhooks of the application, which the page passes with the events they can receive.
// There are a handful of them, so the table searches, sorts, and pages on the client.
// Each webhook is saved, tested, and deleted on its own, right away, independent of the configuration around it.
const { t } = useI18n();
const { formatDateTimeNumeric } = useDates();
const { messageFor } = useApiError();
const { eventLabel, problem } = useWebhookTexts(() => props.events);
const { confirmDelete } = useDeleteConfirm();
const toast = useToast();

const columnState = useTableColumnState('webhooks');

const webhooks = ref<WebhookResponse[]>([]);
const loading = ref(true);
const loadError = ref<unknown>(null);
const search = ref('');

/** Reads the webhooks with the state of their deliveries. */
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
    id === 'status' ? WEBHOOK_HEALTH_ORDER[webhookHealth(webhook)] : webhook[id],
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

/** Shows the dialog for the given webhook, or for a new one. */
function openForm(webhook: WebhookResponse | null): void {
  editedWebhook.value = webhook;
  formOpen.value = true;
}

/** Reloads the table and shows the secret of a new webhook, so it can be handed to the receiver. */
async function onSaved(webhook: WebhookResponse, created: boolean): Promise<void> {
  if (created) {
    createdWebhook.value = webhook;
    secretOpen.value = true;
  }
  await load();
}

/** Shows the last deliveries to a webhook. */
function openDeliveries(webhook: WebhookResponse): void {
  deliveriesWebhook.value = webhook;
  deliveriesOpen.value = true;
}

/** Shows a message to a webhook as it would be sent now. */
function openPreview(webhook: WebhookResponse): void {
  previewWebhook.value = webhook;
  previewOpen.value = true;
}

const { mutate: sendTest } = useMutation(
  async (webhook: WebhookResponse) => webhookRepository.test(webhook.id),
  {
    key: (webhook) => webhook.id,
    onSuccess: async (delivery) => {
      showTestResult(delivery);
      await load();
    },
  },
);

/** Reports how the test delivery ended. */
function showTestResult(delivery: WebhookDeliveryResponse): void {
  if (delivery.outcome === 'delivered') {
    toast.add({
      title: t('webhook.testDelivered'),
      icon: 'i-lucide-check-circle',
      color: 'success',
    });
    return;
  }
  toast.add({
    title: t('webhook.testFailed'),
    description: problem(delivery.errorCode, delivery.statusCode),
    icon: 'i-lucide-circle-alert',
    color: 'error',
  });
}

const { mutate: deleteWebhook } = useMutation(
  async (webhook: WebhookResponse) => webhookRepository.delete(webhook.id),
  {
    key: (webhook) => webhook.id,
    confirm: async (webhook) =>
      confirmDelete({
        title: t('webhook.deleteTitle'),
        description: t('webhook.deleteDescription', {
          name: webhook.name,
        }),
      }),
    success: (_, webhook) =>
      t('webhook.deleted', {
        name: webhook.name,
      }),
    onSuccess: load,
  },
);

/** Returns the further actions of a webhook. */
function rowMenuItems(webhook: WebhookResponse): DropdownMenuItem[] {
  return [
    {
      label: t('webhook.testAction'),
      icon: 'i-lucide-send',
      onSelect: () => void sendTest(webhook),
    },
    {
      label: t('webhook.previewAction'),
      icon: 'i-lucide-eye',
      onSelect: () => openPreview(webhook),
    },
    {
      label: t('webhook.deliveriesAction'),
      icon: 'i-lucide-history',
      onSelect: () => openDeliveries(webhook),
    },
    {
      label: t('common.delete'),
      icon: 'i-lucide-trash-2',
      color: 'error',
      onSelect: () => void deleteWebhook(webhook),
    },
  ];
}

/** Explains the state of a webhook, with the last error of a failing one. */
function healthTooltip(webhook: WebhookResponse): string {
  const { status } = webhook;
  if (webhookHealth(webhook) === 'failing' && status.lastFailedAt) {
    return t('webhook.failingSince', {
      error: problem(status.lastErrorCode, status.lastStatusCode),
      count: status.consecutiveFailures,
      time: formatDateTimeNumeric(status.lastFailedAt),
    });
  }
  if (status.lastDeliveredAt) {
    return t('webhook.lastDelivered', {
      time: formatDateTimeNumeric(status.lastDeliveredAt),
    });
  }
  return t('webhook.noDelivery');
}

const columns = computed<TableColumn<WebhookResponse>[]>(() => [
  {
    accessorKey: 'name',
    header: t('fields.name'),
    enableSorting: true,
  },
  {
    accessorKey: 'url',
    meta: {
      grow: true,
    },
    header: t('fields.url'),
    enableSorting: true,
  },
  {
    id: 'status',
    header: t('webhook.status'),
    enableSorting: true,
  },
  {
    accessorKey: 'events',
    header: t('fields.events'),
  },
  {
    id: TABLE_ACTIONS_COLUMN,
  },
]);
</script>

<template>
  <div class="flex flex-col gap-3 lg:min-h-0 lg:flex-1">
    <TableToolbar
      v-model:search="search"
      :placeholder="t('webhook.search')"
      :can-reset="!!search"
      :loading="loading"
      :on-reload="load"
      @reset="search = ''"
    >
      <template #actions>
        <TableColumnMenu v-model:column-state="columnState" :columns="columns" />
        <UButton icon="i-lucide-plus" class="max-sm:px-1.5" @click="openForm(null)">
          <span class="max-sm:sr-only">{{ t('webhook.new') }}</span>
        </UButton>
      </template>
    </TableToolbar>

    <UAlert
      v-if="loadError"
      :title="t('webhook.loadError')"
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
      class="lg:min-h-80"
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
        <!-- The badge takes the focus, so its tooltip with the reason also opens from the keyboard -->
        <UTooltip :text="healthTooltip(row.original)">
          <UBadge
            tabindex="0"
            :label="t(`webhook.health.${webhookHealth(row.original)}`)"
            :color="HEALTH_COLORS[webhookHealth(row.original)]"
            variant="subtle"
          />
        </UTooltip>
      </template>

      <template #actions-cell="{ row }">
        <TableRowActions :items="rowMenuItems(row.original)">
          <TableRowAction
            :label="t('webhook.editAction')"
            icon="i-lucide-pencil"
            @click="openForm(row.original)"
          />
        </TableRowActions>
      </template>

      <template #empty>
        <TableEmpty
          icon="i-lucide-webhook"
          :text="t('webhook.empty')"
          :filtered="!!search"
          @reset="search = ''"
        />
      </template>
    </DataTable>

    <WebhookFormModal
      v-model:open="formOpen"
      :webhook="editedWebhook"
      :events="events"
      @saved="onSaved"
    />
    <WebhookSecretModal v-model:open="secretOpen" :webhook="createdWebhook" />
    <WebhookDeliveriesModal
      v-model:open="deliveriesOpen"
      :webhook="deliveriesWebhook"
      :events="events"
    />
    <WebhookPreviewModal v-model:open="previewOpen" :webhook="previewWebhook" :events="events" />
  </div>
</template>
