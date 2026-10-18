import { defineStore } from 'pinia';
import { computed, ref, shallowRef, watch } from 'vue';

import { meetRepository } from '@/api/repositories/meet';
import type { MeetResponse } from '@/api/types';
import { buildMeetIndex } from '@/lib/meet';
import { useLiveStore } from '@/stores/live';

/**
 * useMeetStore holds the loaded meet. It loads again whenever the websocket announces a reload of the meet data
 * or reconnects, since a reload may have been missed while the connection was lost.
 */
export const useMeetStore = defineStore('meet', () => {
  const liveStore = useLiveStore();

  // A shallow ref, since the meet is only ever replaced as a whole and holds thousands of entries
  const meet = shallowRef<MeetResponse | null>(null);
  const loading = ref(false);
  const error = ref<unknown>(null);
  // Numbers the loads, so only the answer of the latest one is kept when reloads overlap
  let latestLoad = 0;

  const index = computed(() => (meet.value ? buildMeetIndex(meet.value) : null));

  /** load reads the meet from the server. An answer that arrives after a later load started is dropped. */
  async function load(): Promise<void> {
    const current = ++latestLoad;
    loading.value = true;
    try {
      const result = await meetRepository.get();
      if (current !== latestLoad) return;
      meet.value = result;
      error.value = null;
    } catch (err) {
      if (current !== latestLoad) return;
      error.value = err;
    } finally {
      if (current === latestLoad) loading.value = false;
    }
  }

  // Registered in the setup scope of the store, so it lives as long as the store and not only as long
  // as the component that called start first.
  watch(
    () => [liveStore.meetRevision, liveStore.reconnects],
    () => void load(),
  );

  /** start loads the meet and keeps it current. It also opens the websocket for the live heat. */
  function start(): void {
    liveStore.connect();
    if (!meet.value) void load();
  }

  return {
    meet,
    index,
    loading,
    error,
    load,
    start,
  };
});
