import { defineStore } from 'pinia';
import { computed, ref } from 'vue';

import { type LiveHeat, type LiveMessage, LiveMessageType } from '@/api/types';
import { createClockSync } from '@/lib/clock';
import { formatSwimtime } from '@/lib/time';
import { RESYNC_MESSAGE, useRealtimeStore } from '@/stores/realtime';

/** The time shown while no lane has a time yet. */
const EMPTY_TIME = '00:00,00';

/**
 * useLiveStore holds the live heat the realtime connection reports. Every message carries the full live heat,
 * so each one replaces the current state and the last known state stays visible while reconnecting.
 */
export const useLiveStore = defineStore('live', () => {
  const realtime = useRealtimeStore();

  const state = ref<LiveHeat | null>(null);
  const connected = computed(() => realtime.isConnected);
  // Counts the announced reloads of the meet data, so pages showing the meet can watch it
  const meetRevision = ref(0);
  // Counts the connections opened after a lost or failed one.
  // Pages watch it to fetch again what they might have missed meanwhile.
  const reconnects = computed(() => realtime.reconnects);

  const clock = createClockSync();
  let unsubscribe: Array<() => void> = [];

  /** connect subscribes to the messages of the live heat and opens the connection unless it is already open. */
  function connect(): void {
    if (unsubscribe.length === 0) {
      unsubscribe = [
        // The server may have restarted with another clock
        realtime.on(RESYNC_MESSAGE, () => clock.reset()),
        realtime.on<LiveMessage>(LiveMessageType.Init, applyHeat),
        realtime.on<LiveMessage>(LiveMessageType.Update, applyHeat),
        realtime.on(LiveMessageType.Meet, () => meetRevision.value++),
      ];
    }
    realtime.connect();
  }

  /** disconnect closes the connection and stops reconnecting. The last known state stays. */
  function disconnect(): void {
    for (const off of unsubscribe) off();
    unsubscribe = [];
    realtime.disconnect();
  }

  /**
   * applyHeat replaces the state with the heat of a message. A server that restarted sends an init
   * without a heat until the timing device reports again, so that init keeps a known heat, see keepsHeat.
   */
  function applyHeat(message: LiveMessage): void {
    if (message.type !== LiveMessageType.Init && message.type !== LiveMessageType.Update) return;
    clock.sample(Date.parse(message.serverTime));
    if (message.type === LiveMessageType.Init && keepsHeat(state.value, message.data)) return;
    state.value = message.data;
  }

  // The highest lane time, which is the heat time of the last lane to finish,
  // shown by the timers while the heat is not running.
  const displayTime = computed(() => {
    if (!state.value) return EMPTY_TIME;

    let maxTime = 0;
    for (const lane of Object.values(state.value.lanes)) {
      maxTime = Math.max(maxTime, lane.time);
    }

    return maxTime > 0 ? formatSwimtime(maxTime) : EMPTY_TIME;
  });

  /** startTimeLocal returns the heat start on the performance.now() clock, or null without a start. */
  function startTimeLocal(): number | null {
    if (!state.value?.startTime) return null;
    return clock.toLocal(Date.parse(state.value.startTime));
  }

  return {
    state,
    connected,
    meetRevision,
    reconnects,
    displayTime,
    connect,
    disconnect,
    startTimeLocal,
  };
});

/**
 * keepsHeat reports whether an init snapshot without a heat leaves the known heat in place.
 * A heat is known once the timing device reported it, before that its status is empty.
 */
function keepsHeat(current: LiveHeat | null, init: LiveHeat): boolean {
  return init.status === '' && current !== null && current.status !== '';
}
