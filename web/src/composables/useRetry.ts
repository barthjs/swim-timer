import { getCurrentScope, onScopeDispose, ref } from 'vue';

/** RETRY_BASE_DELAY_MS is the wait after the first failure, doubled after each further one. */
const RETRY_BASE_DELAY_MS = 1_000;

/** RETRY_MAX_DELAY_MS is the longest wait between two tries. */
const RETRY_MAX_DELAY_MS = 30_000;

/** retryDelay returns how long to wait before the next try after the given number of failures. */
export function retryDelay(failures: number): number {
  const exponent = Math.max(0, failures - 1);
  return Math.min(RETRY_MAX_DELAY_MS, RETRY_BASE_DELAY_MS * 2 ** exponent);
}

/**
 * useRetry runs a task until it succeeds, waiting longer after each failure, and hands its result to onSuccess.
 * It serves pages that run unattended, such as the overlays.
 * Starting again drops the result of a run still in progress.
 * Inside a component or another effect scope, the retries stop with it.
 */
export function useRetry<T>(task: () => Promise<T>, onSuccess: (result: T) => void) {
  // Whether the last run failed and a retry is scheduled
  const failed = ref(false);

  let failures = 0;
  let run = 0;
  let timer: ReturnType<typeof setTimeout> | null = null;

  /** clearTimer cancels a scheduled retry. */
  function clearTimer(): void {
    if (timer === null) return;
    clearTimeout(timer);
    timer = null;
  }

  /** attempt runs the task once and schedules the next try if it fails. */
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

  /** start runs the task from the beginning, dropping a run or retry still pending. */
  function start(): void {
    stop();
    failures = 0;
    void attempt(run);
  }

  /** retryNow runs a scheduled retry at once, for example after the connection came back. */
  function retryNow(): void {
    if (!failed.value || timer === null) return;
    clearTimer();
    void attempt(run);
  }

  /** stop cancels the scheduled retry and drops the result of a run in progress. */
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
