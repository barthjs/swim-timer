import { defineStore } from 'pinia';
import { computed, ref } from 'vue';

import { timingRepository } from '@/api/repositories/timing';
import type { TimingStatusResponse } from '@/api/types';

/** POLL_INTERVAL_MS is how often the connection to the timing device is checked. */
const POLL_INTERVAL_MS = 5_000;

/** useTimingStore polls and holds the connection status of the timing device. */
export const useTimingStore = defineStore('timing', () => {
  // The last status the backend reported, or null before the first poll resolves
  const status = ref<TimingStatusResponse | null>(null);

  // Optimistic until the first poll resolves, so the dashboard never flashes a false "disconnected" state on load
  const connected = computed(() => status.value?.connected ?? true);

  let timer: ReturnType<typeof setInterval> | null = null;

  /** poll reads the status of the timing device. */
  async function poll(): Promise<void> {
    try {
      status.value = await timingRepository.getStatus();
    } catch {
      // A failed request does not prove that the device is disconnected, so the last status stays
    }
  }

  /** start polls the status of the timing device at a fixed interval. */
  function start(): void {
    if (timer !== null) return;
    void poll();
    timer = setInterval(poll, POLL_INTERVAL_MS);
  }

  /** stop ends the polling. */
  function stop(): void {
    if (timer === null) return;
    clearInterval(timer);
    timer = null;
  }

  return {
    status,
    connected,
    start,
    stop,
  };
});
