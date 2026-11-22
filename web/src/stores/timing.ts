import { defineStore } from 'pinia';
import { computed, ref } from 'vue';

import { timingRepository } from '@/api/repositories/timing';
import type { TimingStatusResponse } from '@/api/types/timing';
import { useRealtimeStore } from '@/stores/realtime';

/** Holds the connection status of the timing device and fetches it again whenever it changes on the server. */
export const useTimingStore = defineStore('timing', () => {
  const realtime = useRealtimeStore();

  // The last status the backend reported, or null before the first fetch resolves
  const status = ref<TimingStatusResponse | null>(null);

  // Optimistic until the first fetch resolves, so the dashboard never flashes a false "disconnected" state on load
  const connected = computed(() => status.value?.connected ?? true);

  let unsubscribe: (() => void) | null = null;

  /** Reads the status of the timing device. */
  async function load(): Promise<void> {
    try {
      status.value = await timingRepository.getStatus();
    } catch {
      // A failed request does not prove that the device is disconnected, so the last status stays
    }
  }

  /**
   * Fetches the status and follows the announced changes until stop.
   * A connection that opens fetches it through the change it reports, so only an open one needs a fetch here.
   */
  function start(): void {
    if (unsubscribe) return;

    unsubscribe = realtime.onChange('timing', () => void load());
    if (realtime.isConnected) void load();
    realtime.connect();
  }

  /** Ends following the changes. */
  function stop(): void {
    unsubscribe?.();
    unsubscribe = null;
  }

  return {
    status,
    connected,
    start,
    stop,
  };
});
