/** How many recent messages the clock offset is estimated from. */
const DEFAULT_SAMPLE_COUNT = 32;

export type ClockSync = {
  /** Records the server time a message was sent at, on its arrival. */
  sample: (serverTimeMS: number) => void;
  /** Converts a server timestamp to the local performance.now() clock, or null before the first sample. */
  toLocal: (serverTimeMS: number) => number | null;
  /** Discards all samples, after a reconnect to a server that may have restarted. */
  reset: () => void;
};

/** Creates an estimate of the offset between the server clock and the local monotonic clock. */
export function createClockSync(sampleCount = DEFAULT_SAMPLE_COUNT): ClockSync {
  // Every sample is the server time minus the local arrival time, the clock offset minus the delay of the message.
  // A message can only arrive late, so the largest offset of the recent samples is the one closest to the real offset,
  // and keeping only recent samples follows a slow drift or a corrected server clock.
  // performance.now() keeps a change of the local wall clock from moving a running timer.
  let samples: number[] = [];
  let offset: number | null = null;

  /** Adds the server time of a message and updates the offset estimate. */
  function sample(serverTimeMS: number): void {
    if (!Number.isFinite(serverTimeMS)) return;

    samples.push(serverTimeMS - performance.now());
    if (samples.length > sampleCount) {
      samples.shift();
    }
    offset = Math.max(...samples);
  }

  /**
   * Converts a server time to the performance.now() clock, or returns null without an estimate.
   */
  function toLocal(serverTimeMS: number): number | null {
    if (offset === null || !Number.isFinite(serverTimeMS)) return null;
    return serverTimeMS - offset;
  }

  /** Drops the estimate, after reconnecting to a server that may have restarted. */
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
