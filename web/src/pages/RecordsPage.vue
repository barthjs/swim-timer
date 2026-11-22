<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { recordRepository } from '@/api/repositories/record';
import { recordBreakRepository } from '@/api/repositories/recordBreak';
import { recordTableRepository } from '@/api/repositories/recordTable';
import type { RecordResponse } from '@/api/types/record';
import type { RecordBreakResponse } from '@/api/types/recordBreak';
import type { RecordTableResponse } from '@/api/types/recordTable';
import { ApiError } from '@/api/client/errors';
import RecordBreaksTable from '@/components/record/RecordBreaksTable.vue';
import RecordsTable from '@/components/record/RecordsTable.vue';
import RecordTablePicker from '@/components/record/RecordTablePicker.vue';
import DocsLink from '@/components/system/DocsLink.vue';
import SectionNav from '@/components/ui/SectionNav.vue';
import { useMutation } from '@/composables/form/useMutation';
import { useApiError } from '@/composables/form/useApiError';
import { useUrlSection } from '@/composables/ui/useUrlSection';
import { useDates } from '@/composables/ui/useDates';
import { recordRows } from '@/lib/record/record';
import { chosenRecordTableId } from '@/lib/record/recordTable';
import { replaceRouteQuery } from '@/lib/ui/routeQuery';
import { useToast } from '@nuxt/ui/composables';

// The record tables, their records, and the records set at the meet. The tables are checked in their order
// once a heat is official, see the documentation of the records.
const { t } = useI18n();
const { formatDateTime } = useDates();
const route = useRoute();
const router = useRouter();
const { messageFor } = useApiError();
const toast = useToast();

const tables = ref<RecordTableResponse[]>([]);
const records = ref<RecordResponse[]>([]);
const breaks = ref<RecordBreakResponse[]>([]);
const loadedAt = ref<string | null>(null);
const loading = ref(true);
const loadError = ref<unknown>(null);
// Counts the loads, so only the answer of the latest one is shown
let loads = 0;

/**
 * Reads the tables, their records, and the records set at the meet.
 * The answer of a load that a later one overtook is dropped.
 */
async function load(): Promise<void> {
  loads += 1;
  const current = loads;
  loading.value = true;
  loadError.value = null;
  try {
    const [tableList, recordList, breakList] = await Promise.all([
      recordTableRepository.list(),
      recordRepository.list(),
      recordBreakRepository.list(),
    ]);
    if (current !== loads) return;
    tables.value = tableList.tables;
    // A link to a table that no longer exists shows every table and drops the table from the URL
    if (route.query.table !== undefined && !tableId.value) tableId.value = '';
    records.value = recordList.records;
    breaks.value = breakList;
    loadedAt.value = tableList.loadedAt ?? null;
  } catch (err) {
    if (current !== loads) return;
    loadError.value = err;
  } finally {
    if (current === loads) loading.value = false;
  }
}

onMounted(load);

// The chosen record table, kept in the URL so a view of one table can be shared, empty for every table or an unknown one
const tableId = computed<string>({
  get: () =>
    chosenRecordTableId(
      typeof route.query.table === 'string' ? route.query.table : '',
      tables.value,
    ),
  set: (value) => {
    void replaceRouteQuery(route, router, (query) => ({
      ...query,
      table: value || undefined,
      page: undefined,
    }));
  },
});

const rows = computed(() =>
  recordRows(records.value, tables.value).filter(
    (row) => !tableId.value || row.record.tableId === tableId.value,
  ),
);

// Computed, since the language and the counts change after the page is set up
const sections = computed(() => [
  {
    value: 'records' as const,
    label: t('pages.records.sections.records'),
    icon: 'i-lucide-trophy',
    count: rows.value.length,
  },
  {
    value: 'breaks' as const,
    label: t('pages.records.sections.breaks'),
    icon: 'i-lucide-medal',
    count: breaks.value.length,
  },
]);

// Each section has its own columns, so switching sections starts on the first page without sorting or chosen table
const section = useUrlSection(() => sections.value.map((item) => item.value), 'records');

const loadedAtLabel = computed(() => (loadedAt.value ? formatDateTime(loadedAt.value) : null));

const { mutate: reload, pending: reloading } = useMutation(
  async () => recordTableRepository.reload(),
  {
    success: () => t('pages.records.reloadSuccess'),
    onSuccess: load,
    onError: (err) => {
      // A file that cannot be read is reported with its position, since the admin has to fix it
      toast.add({
        title: t('pages.records.reloadError'),
        description:
          err instanceof ApiError && err.code === 'bad_request' ? err.message : messageFor(err),
        icon: 'i-lucide-circle-alert',
        color: 'error',
        duration: 0,
      });
      return true;
    },
  },
);
</script>

<template>
  <div class="flex flex-col gap-4 lg:min-h-0 lg:flex-1">
    <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
      <div class="flex items-center gap-1">
        <h1 class="text-highlighted heading-page">{{ t('pages.records.title') }}</h1>
        <DocsLink page="records" />
      </div>
      <span
        v-if="loadedAtLabel"
        class="text-muted order-last flex w-full items-center gap-1 text-sm sm:order-0 sm:ml-auto sm:w-auto"
      >
        {{ t('pages.records.loadedAt', { time: loadedAtLabel }) }}
      </span>
      <div class="ml-auto flex items-center gap-1.5" :class="{ 'sm:ml-0': loadedAtLabel }">
        <UTooltip :text="t('pages.records.reloadHint')">
          <UButton
            :label="t('pages.records.reload')"
            icon="i-lucide-refresh-ccw"
            :loading="reloading"
            @click="reload()"
          />
        </UTooltip>
      </div>
    </div>

    <UAlert
      v-if="loadError"
      :title="t('pages.records.loadError')"
      :description="messageFor(loadError)"
      icon="i-lucide-circle-alert"
      color="error"
    />

    <template v-else>
      <SectionNav v-model="section" :sections="sections" variant="pill" />

      <RecordsTable v-if="section === 'records'" :rows="rows" :loading="loading">
        <template #leading>
          <RecordTablePicker v-model="tableId" :tables="tables" @changed="load" />
        </template>
      </RecordsTable>
      <RecordBreaksTable v-else :breaks="breaks" :loading="loading" @changed="load" />
    </template>
  </div>
</template>
