import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import type { timingRepository } from '@/api/repositories/timing';
import type { TimingStatusResponse } from '@/api/types/timing';
import { useTimingStore } from '@/stores/timing';

const { getStatus, handlers, connect, connection } = vi.hoisted(() => ({
  getStatus: vi.fn<typeof timingRepository.getStatus>(),
  handlers: new Map<string, Set<() => void>>(),
  connect: vi.fn<() => void>(),
  connection: {
    isConnected: true,
  },
}));

vi.mock('@/api/repositories/timing', () => ({
  timingRepository: {
    getStatus,
  },
}));

vi.mock('@/stores/realtime', () => ({
  useRealtimeStore: () => ({
    connect,
    get isConnected() {
      return connection.isConnected;
    },
    onChange: (resource: string, handler: () => void) => {
      const set = handlers.get(resource) ?? new Set();
      handlers.set(resource, set);
      set.add(handler);
      return () => set.delete(handler);
    },
  }),
}));

/** Returns a status of the timing device that is connected or not. */
function fakeStatus(connected: boolean): TimingStatusResponse {
  return {
    connected,
    port: connected ? faker.system.filePath() : '',
    configuredPort: '',
    simulateEvent: 0,
    replayFile: '',
  };
}

/** Reports a change of the resource to its handlers like the realtime connection. */
async function announce(resource: string): Promise<void> {
  for (const handler of handlers.get(resource) ?? []) handler();
  await vi.waitFor(() => expect(getStatus).toHaveBeenCalled());
}

describe('useTimingStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    handlers.clear();
    connection.isConnected = true;
    vi.clearAllMocks();
  });

  describe('connected', () => {
    it('is optimistic before the first fetch', () => {
      const store = useTimingStore();

      expect(store.connected).toBe(true);
    });
  });

  describe('start', () => {
    it('stores the status of the timing device', async () => {
      const status = fakeStatus(false);
      getStatus.mockResolvedValue(status);
      const store = useTimingStore();

      store.start();

      await vi.waitFor(() => expect(store.status).toEqual(status));
      expect(store.connected).toBe(false);
      expect(connect).toHaveBeenCalledOnce();
    });

    it('leaves the first fetch to a connection that opens', () => {
      connection.isConnected = false;
      const store = useTimingStore();

      store.start();

      expect(getStatus).not.toHaveBeenCalled();
    });

    it('fetches the status again once it changes on the server', async () => {
      getStatus.mockResolvedValue(fakeStatus(true));
      const store = useTimingStore();
      store.start();
      await vi.waitFor(() => expect(store.status).not.toBeNull());
      const changed = fakeStatus(false);
      getStatus.mockClear().mockResolvedValue(changed);

      await announce('timing');

      await vi.waitFor(() => expect(store.status).toEqual(changed));
    });

    it('keeps the last status when a fetch fails', async () => {
      const status = fakeStatus(true);
      getStatus.mockResolvedValue(status);
      const store = useTimingStore();
      store.start();
      await vi.waitFor(() => expect(store.status).toEqual(status));
      getStatus.mockClear().mockRejectedValue(new TypeError('Failed to fetch'));

      await announce('timing');

      expect(store.status).toEqual(status);
    });

    it('follows the changes only once when started again', async () => {
      getStatus.mockResolvedValue(fakeStatus(true));
      const store = useTimingStore();
      store.start();
      store.start();
      getStatus.mockClear();

      await announce('timing');

      expect(getStatus).toHaveBeenCalledOnce();
    });
  });

  describe('stop', () => {
    it('ends following the changes', async () => {
      getStatus.mockResolvedValue(fakeStatus(true));
      const store = useTimingStore();
      store.start();
      store.stop();
      getStatus.mockClear();

      for (const handler of handlers.get('timing') ?? []) handler();

      expect(getStatus).not.toHaveBeenCalled();
    });
  });
});
