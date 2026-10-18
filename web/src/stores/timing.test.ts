import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { timingRepository } from '@/api/repositories/timing';
import { useTimingStore } from '@/stores/timing';

const getStatus = vi.hoisted(() => vi.fn<typeof timingRepository.getStatus>());

vi.mock('@/api/repositories/timing', () => ({
  timingRepository: {
    getStatus,
  },
}));

const connectedStatus = {
  connected: true,
  port: '/dev/ttyUSB0',
  configuredPort: '',
  simulateEvent: 0,
  replayFile: '',
};

const disconnectedStatus = {
  ...connectedStatus,
  connected: false,
  port: '',
};

describe('useTimingStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    getStatus.mockReset();
    vi.useFakeTimers();
  });

  afterEach(() => {
    useTimingStore().stop();
    vi.useRealTimers();
  });

  describe('connected', () => {
    it('is optimistic before the first poll', () => {
      const store = useTimingStore();

      const connected = store.connected;

      expect(connected).toBe(true);
      expect(store.status).toBeNull();
    });
  });

  describe('start', () => {
    it('stores the status of the timing device', async () => {
      getStatus.mockResolvedValue(connectedStatus);
      const store = useTimingStore();

      store.start();
      await vi.advanceTimersByTimeAsync(0);

      expect(getStatus).toHaveBeenCalledOnce();
      expect(store.status).toEqual(connectedStatus);
    });

    it('polls the status at a fixed interval', async () => {
      getStatus.mockResolvedValueOnce(connectedStatus).mockResolvedValueOnce(disconnectedStatus);
      const store = useTimingStore();

      store.start();
      await vi.advanceTimersByTimeAsync(5_000);

      expect(getStatus).toHaveBeenCalledTimes(2);
      expect(store.connected).toBe(false);
    });

    it('keeps a running poll when started again', async () => {
      getStatus.mockResolvedValue(connectedStatus);
      const store = useTimingStore();
      store.start();

      store.start();
      await vi.advanceTimersByTimeAsync(0);

      expect(getStatus).toHaveBeenCalledOnce();
    });

    it('keeps the last status when a poll fails', async () => {
      getStatus.mockResolvedValueOnce(connectedStatus).mockRejectedValueOnce(new Error('offline'));
      const store = useTimingStore();

      store.start();
      await vi.advanceTimersByTimeAsync(5_000);

      expect(getStatus).toHaveBeenCalledTimes(2);
      expect(store.status).toEqual(connectedStatus);
    });
  });

  describe('stop', () => {
    it('ends the polling', async () => {
      getStatus.mockResolvedValue(connectedStatus);
      const store = useTimingStore();
      store.start();
      await vi.advanceTimersByTimeAsync(5_000);

      store.stop();
      await vi.advanceTimersByTimeAsync(20_000);

      expect(getStatus).toHaveBeenCalledTimes(2);
    });

    it('keeps a stopped poll stopped when stopped again', async () => {
      getStatus.mockResolvedValue(connectedStatus);
      const store = useTimingStore();
      store.start();
      await vi.advanceTimersByTimeAsync(0);
      store.stop();

      store.stop();
      await vi.advanceTimersByTimeAsync(20_000);

      expect(getStatus).toHaveBeenCalledOnce();
    });
  });
});
