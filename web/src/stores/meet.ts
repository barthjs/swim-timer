import { defineStore } from 'pinia';
import { computed, ref, shallowRef } from 'vue';

import { meetRepository } from '@/api/repositories/meet';
import type { MeetResponse } from '@/api/types/meet';
import { buildMeetIndex } from '@/lib/meet/meet';
import { useLiveStore } from '@/stores/live';
import { useRealtimeStore } from '@/stores/realtime';

/** Holds the loaded meet and loads it again whenever it changes on the server. */
export const useMeetStore = defineStore('meet', () => {
  const liveStore = useLiveStore();
  const realtime = useRealtimeStore();

  // A shallow ref, since the meet is only ever replaced as a whole and holds thousands of entries
  const meet = shallowRef<MeetResponse | null>(null);
  const loading = ref(false);
  const error = ref<unknown>(null);
  // Numbers the loads, so only the answer of the latest one is kept when reloads overlap
  let latestLoad = 0;

  const index = computed(() => (meet.value ? buildMeetIndex(meet.value) : null));

  /** Reads the meet from the server. An answer that arrives after a later load started is dropped. */
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
  realtime.onChange('meet', () => void load());

  /** Loads the meet, keeps it current, and opens the websocket for the live heat. */
  function start(): void {
    // A connection that opens loads the meet through the change it reports
    if (!meet.value && realtime.isConnected) void load();
    liveStore.connect();
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
