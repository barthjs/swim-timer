import { defineStore } from 'pinia';
import { ref } from 'vue';

import { type RealtimeMessage, useRealtimeStore } from '@/stores/realtime';

/** Carries the state of every dependency of the server, on connect and whenever one of them changes. */
export const HEALTH_MESSAGE = 'health';

/** Names the state of every dependency, ok for one that is reachable. */
export type HealthMessage = RealtimeMessage & {
  data: Record<string, string>;
};

const CHECK_OK = 'ok';

/**
 * Holds the state of the dependencies of the server, which it sends on every connect
 * and whenever one of them changes. A dependency counts as ready until the server reports otherwise,
 * so a page does not flash a missing one on load.
 */
export const useHealthStore = defineStore('health', () => {
  const realtime = useRealtimeStore();

  const checks = ref<Record<string, string>>({});

  realtime.onState<HealthMessage>(HEALTH_MESSAGE, (message) => {
    checks.value = message.data;
  });

  /** Reports whether a dependency of the server is reachable. */
  function isReady(name: string): boolean {
    return (checks.value[name] ?? CHECK_OK) === CHECK_OK;
  }

  return {
    checks,
    isReady,
  };
});
