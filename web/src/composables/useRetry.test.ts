import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { retryDelay, useRetry } from '@/composables/useRetry';

/** settle lets pending promise callbacks run. */
async function settle(): Promise<void> {
  await Promise.resolve();
  await Promise.resolve();
}

describe('retryDelay', () => {
  it.each([
    {
      failures: 1,
      expected: 1_000,
    },
    {
      failures: 2,
      expected: 2_000,
    },
    {
      failures: 3,
      expected: 4_000,
    },
  ])('doubles the delay to $expected ms after $failures failures', ({ failures, expected }) => {
    const delay = retryDelay(failures);

    expect(delay).toBe(expected);
  });

  it('keeps the delay at the maximum', () => {
    const delay = retryDelay(10);

    expect(delay).toBe(30_000);
  });
});

describe('useRetry', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  /** failTwice returns a task that fails twice and then resolves. */
  function failTwice() {
    return vi
      .fn<() => Promise<string>>()
      .mockRejectedValueOnce(new Error('offline'))
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValue('done');
  }

  describe('start', () => {
    it('passes the result of the task to the callback', async () => {
      const task = vi.fn<() => Promise<string>>().mockResolvedValue('done');
      const onSuccess = vi.fn<(result: string) => void>();
      const retry = useRetry(task, onSuccess);

      retry.start();
      await settle();

      expect(task).toHaveBeenCalledOnce();
      expect(onSuccess).toHaveBeenCalledWith('done');
      expect(retry.failed.value).toBe(false);
    });

    it('marks a failed run', async () => {
      const retry = useRetry(failTwice(), () => {});

      retry.start();
      await settle();

      expect(retry.failed.value).toBe(true);
    });

    it('retries after the first delay', async () => {
      const task = failTwice();
      const retry = useRetry(task, () => {});

      retry.start();
      await vi.advanceTimersByTimeAsync(1_000);

      expect(task).toHaveBeenCalledTimes(2);
    });

    it('waits longer before the next retry', async () => {
      const task = failTwice();
      const retry = useRetry(task, () => {});
      retry.start();
      await vi.advanceTimersByTimeAsync(1_000);

      await vi.advanceTimersByTimeAsync(1_999);

      expect(task).toHaveBeenCalledTimes(2);
    });

    it('retries with growing delays until the task succeeds', async () => {
      const task = failTwice();
      const onSuccess = vi.fn<(result: string) => void>();
      const retry = useRetry(task, onSuccess);
      retry.start();
      await vi.advanceTimersByTimeAsync(1_000);

      await vi.advanceTimersByTimeAsync(2_000);

      expect(task).toHaveBeenCalledTimes(3);
      expect(onSuccess).toHaveBeenCalledWith('done');
      expect(retry.failed.value).toBe(false);
    });

    it('drops the result of a run that was started again', async () => {
      let resolveFirst: (value: string) => void = () => {};
      const task = vi
        .fn<() => Promise<string>>()
        .mockImplementationOnce(
          () =>
            new Promise<string>((resolve) => {
              resolveFirst = resolve;
            }),
        )
        .mockResolvedValue('second');
      const onSuccess = vi.fn<(result: string) => void>();
      const retry = useRetry(task, onSuccess);
      retry.start();

      retry.start();
      await settle();
      resolveFirst('first');
      await settle();

      expect(onSuccess).toHaveBeenCalledExactlyOnceWith('second');
    });
  });

  describe('retryNow', () => {
    it('retries at once while a retry is scheduled', async () => {
      const task = vi
        .fn<() => Promise<number>>()
        .mockRejectedValueOnce(new Error('offline'))
        .mockResolvedValue(1);
      const onSuccess = vi.fn<(result: number) => void>();
      const retry = useRetry(task, onSuccess);
      retry.start();
      await settle();

      retry.retryNow();
      await settle();

      expect(task).toHaveBeenCalledTimes(2);
      expect(onSuccess).toHaveBeenCalledWith(1);
    });

    it('keeps a run that succeeded', async () => {
      const task = vi.fn<() => Promise<number>>().mockResolvedValue(1);
      const retry = useRetry(task, () => {});
      retry.start();
      await settle();

      retry.retryNow();
      await settle();

      expect(task).toHaveBeenCalledOnce();
    });
  });

  describe('stop', () => {
    it('ends the retries', async () => {
      const task = vi.fn<() => Promise<void>>().mockRejectedValue(new Error('offline'));
      const retry = useRetry(task, () => {});
      retry.start();
      await settle();

      retry.stop();
      await vi.advanceTimersByTimeAsync(60_000);

      expect(task).toHaveBeenCalledOnce();
    });
  });
});
