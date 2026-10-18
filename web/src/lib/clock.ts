/** How many recent messages the clock offset is estimated from. */
const DEFAULT_SAMPLE_COUNT = 32;

export type ClockSync = {
  /** Records the server time a message was sent at, on its arrival. */
  sample: (serverTimeMS: number) => void;
  /** Converts a server timestamp to the local performance.now() clock, or null before the first sample. */
  toLocal: (serverTimeMS: number) => number | null;
  /** Discards all samples, for example after a reconnect to a possibly restarted server. */
  reset: () => void;
};

/**
 * createClockSync creates an estimate of the offset between the server clock and the local monotonic clock.
 *
 * Every sample is the server time minus the local arrival time, which is the clock offset minus the delay
 * the message took. A message can only arrive late, never early, so the largest offset of the recent samples
 * is the one closest to the real offset. Keeping only recent samples lets the estimate follow a slow drift
 * or a corrected server clock. performance.now() is used instead of Date.now(), so a change of the local
 * wall clock does not move a running timer.
 */
export function createClockSync(sampleCount = DEFAULT_SAMPLE_COUNT): ClockSync {
  let samples: number[] = [];
  let offset: number | null = null;

  /** sample adds the server time of a message and updates the offset estimate. */
  function sample(serverTimeMS: number): void {
    if (!Number.isFinite(serverTimeMS)) return;

    samples.push(serverTimeMS - performance.now());
    if (samples.length > sampleCount) {
      samples.shift();
    }
    offset = Math.max(...samples);
  }

  /**
   * toLocal converts a server time to the performance.now() clock, or returns null without an estimate.
   */
  function toLocal(serverTimeMS: number): number | null {
    if (offset === null || !Number.isFinite(serverTimeMS)) return null;
    return serverTimeMS - offset;
  }

  /** reset drops the estimate, for example after reconnecting to a restarted server. */
  function reset(): void {
    samples = [];
    offset = null;
  }

  return {
    sample,
    toLocal,
    reset,
  };
}
