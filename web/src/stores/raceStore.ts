import { defineStore } from 'pinia';
import { ref } from 'vue';

import { formatSwimtime } from '@/utils/time';

import type { RaceState, WSMessage } from '@/types';
import { RaceStatus, WSMessageType } from '@/types';

export const useRaceStore = defineStore('race', () => {
  const state = ref<RaceState | null>(null);

  let ws: WebSocket | null = null;
  let raceStartTime: number | null = null;
  let animationFrameId: number | null = null;

  const connect = () => {
    if (ws) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    ws = new WebSocket(wsUrl);

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data);
        if (msg.type === WSMessageType.Init || msg.type === WSMessageType.Update) {
          state.value = msg.data;
          handleStateUpdate();
        }
      } catch (e) {
        console.error('Failed to parse WS message', e);
      }
    };

    ws.onclose = () => {
      ws = null;
      setTimeout(() => connect(), 500);
    };

    ws.onerror = (e) => {
      console.error('WS error', e);
    };
  };

  // Processes incoming state changes and manages the timer loop
  const handleStateUpdate = () => {
    if (!state.value) return;

    const isRunning = state.value.status === RaceStatus.Start;

    if (isRunning && state.value.start_time) {
      syncRaceStartTime();
    } else {
      raceStartTime = null;
    }

    const shouldRunTimer = isRunning && raceStartTime !== null;

    if (shouldRunTimer && animationFrameId === null) {
      startTimerLoop();
    }

    if (!shouldRunTimer && animationFrameId !== null) {
      cancelAnimationFrame(animationFrameId);
      animationFrameId = null;
    }
  };

  // Calculates and anchors the exact start time of the race
  const syncRaceStartTime = () => {
    if (!state.value || !state.value.start_time) {
      raceStartTime = null;
      return;
    }

    const backendStart = new Date(state.value.start_time).getTime();
    const now = Date.now();
    const perfNow = performance.now();

    // Convert backend absolute start time to performance.now() relative time
    raceStartTime = perfNow - (now - backendStart);
  };

  // Starts the timer loop that updates the visual timer every frame
  const startTimerLoop = () => {
    const loop = () => {
      updateTimer();
      animationFrameId = requestAnimationFrame(loop);
    };
    animationFrameId = requestAnimationFrame(loop);
  };

  // Continuously updates the visual timer
  const updateTimer = () => {
    if (raceStartTime === null || !state.value || state.value.status === RaceStatus.Official)
      return;

    const timer = document.getElementById('timer');
    if (!timer) return;

    const elapsedMs = Math.max(0, performance.now() - raceStartTime);
    timer.innerText = formatSwimtime(elapsedMs);
  };

  // Returns the formatted display time for the current race state, using the maximum valid time from all lanes
  const getDisplayTime = (): string => {
    if (!state.value) return '00:00,00';

    const lanes = Object.values(state.value.lanes);
    const maxTime = lanes.reduce((max, lane) => Math.max(max, lane.time), 0);
    return formatSwimtime(maxTime) || '00:00,00';
  };

  return {
    state,
    connect,
    getDisplayTime,
  };
});
