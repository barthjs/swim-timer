import { getCurrentScope, onScopeDispose, ref } from 'vue';

/** The wait after the first failure, doubled after each further one. */
const RETRY_BASE_DELAY_MS = 1_000;

/** The longest wait between two tries. */
const RETRY_MAX_DELAY_MS = 30_000;

/** Returns how long to wait before the next try after the given number of failures. */
export function retryDelay(failures: number): number {
  const exponent = Math.max(0, failures - 1);
  return Math.min(RETRY_MAX_DELAY_MS, RETRY_BASE_DELAY_MS * 2 ** exponent);
}

/**
 * Runs a task until it succeeds, waiting longer after each failure, and hands its result to onSuccess,
 * for pages that run unattended.
 */
export function useRetry<T>(task: () => Promise<T>, onSuccess: (result: T) => void) {
  // Whether the last run failed and a retry is scheduled
  const failed = ref(false);

  let failures = 0;
  let run = 0;
  let timer: ReturnType<typeof setTimeout> | null = null;

  /** Cancels a scheduled retry. */
  function clearTimer(): void {
    if (timer === null) return;
    clearTimeout(timer);
    timer = null;
  }

  /** Runs the task once and schedules the next try if it fails. */
  async function attempt(current: number): Promise<void> {
    timer = null;
    try {
      const result = await task();
      if (current !== run) return;
      failures = 0;
      failed.value = false;
      onSuccess(result);
    } catch (err) {
      if (current !== run) return;
      console.error(err);
      failures++;
      failed.value = true;
      timer = setTimeout(() => void attempt(current), retryDelay(failures));
    }
  }

  /** Runs the task from the beginning, dropping a run or retry still pending. */
  function start(): void {
    stop();
    failures = 0;
    void attempt(run);
  }

  /** Runs a scheduled retry at once, as after the connection came back. */
  function retryNow(): void {
    if (!failed.value || timer === null) return;
    clearTimer();
    void attempt(run);
  }

  /** Cancels the scheduled retry and drops the result of a run in progress. */
  function stop(): void {
    clearTimer();
    run++;
    failed.value = false;
  }

  if (getCurrentScope()) {
    onScopeDispose(stop);
  }

  return {
    failed,
    start,
    retryNow,
    stop,
  };
}
