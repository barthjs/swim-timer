import { createPinia, disposePinia, setActivePinia, type Pinia } from 'pinia';
import { effectScope, nextTick } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { meetRepository } from '@/api/repositories/meet';
import type { MeetResponse } from '@/api/types/meet';
import { useMeetStore } from '@/stores/meet';

const { changeHandlers, connection, get, live } = vi.hoisted(() => ({
  changeHandlers: new Set<() => void>(),
  connection: {
    isConnected: true,
  },
  get: vi.fn<typeof meetRepository.get>(),
  live: {
    connect: vi.fn<() => void>(),
  },
}));

vi.mock('@/api/repositories/meet', () => ({
  meetRepository: {
    get,
  },
}));

vi.mock('@/stores/live', () => ({
  useLiveStore: () => live,
}));

vi.mock('@/stores/realtime', () => ({
  useRealtimeStore: () => ({
    get isConnected() {
      return connection.isConnected;
    },
    onChange: (resource: string, handler: () => void) => {
      if (resource !== 'meet') return () => {};
      changeHandlers.add(handler);
      return () => changeHandlers.delete(handler);
    },
  }),
}));

const meet: MeetResponse = {
  loadedAt: '2026-05-01T08:00:00Z',
  events: [
    {
      number: 1,
      phase: 0,
      phaseName: '',
      heatCount: 1,
      name: 'Event A',
      date: '2026-05-01',
      time: '10:00',
      distance: null,
      style: null,
      category: null,
    },
  ],
  startEntries: [
    {
      event: 1,
      phase: 0,
      heat: 1,
      lane: 4,
      relayPosition: 0,
      competitorId: 7,
    },
  ],
  competitors: [
    {
      id: 7,
      firstName: 'A',
      lastName: 'B',
      birthYear: '2010',
      club: 'C',
      category: 'O',
    },
  ],
};

/** Waits for pending promises and the watchers they trigger. */
async function flush(): Promise<void> {
  await Promise.resolve();
  await nextTick();
}

/** Reports a change of the meet on the server like the realtime connection. */
function announceChange(): void {
  for (const handler of changeHandlers) handler();
}

describe('useMeetStore', () => {
  let pinia: Pinia;

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    get.mockReset();
    live.connect.mockClear();
    changeHandlers.clear();
    connection.isConnected = true;
  });

  // The watcher of a store lives as long as its Pinia, so the store of a finished test must not reload
  afterEach(() => {
    disposePinia(pinia);
  });

  describe('index', () => {
    it('is empty without a loaded meet', () => {
      const store = useMeetStore();

      const index = store.index;

      expect(index).toBeNull();
    });
  });

  describe('load', () => {
    it('stores the meet and builds its index', async () => {
      get.mockResolvedValue(meet);
      const store = useMeetStore();

      await store.load();

      expect(store.meet).toEqual(meet);
      expect(store.index?.rows).toHaveLength(1);
      expect(store.loading).toBe(false);
      expect(store.error).toBeNull();
    });

    it('keeps only the answer of the latest load', async () => {
      const newer: MeetResponse = {
        ...meet,
        loadedAt: '2026-05-01T09:00:00Z',
      };
      let resolveOlder: (value: MeetResponse) => void = () => {};
      get.mockImplementationOnce(
        async () =>
          new Promise<MeetResponse>((resolve) => {
            resolveOlder = resolve;
          }),
      );
      get.mockResolvedValueOnce(newer);
      const store = useMeetStore();
      const older = store.load();

      await store.load();
      resolveOlder(meet);
      await older;

      expect(store.meet).toEqual(newer);
      expect(store.loading).toBe(false);
    });

    it('keeps the error of a failed load', async () => {
      const failure = new Error('offline');
      get.mockRejectedValue(failure);
      const store = useMeetStore();

      await store.load();

      expect(store.error).toBe(failure);
      expect(store.loading).toBe(false);
    });
  });

  describe('start', () => {
    it('leaves the first load to a connection that opens', async () => {
      connection.isConnected = false;
      const store = useMeetStore();

      store.start();
      await flush();

      expect(get).not.toHaveBeenCalled();
      expect(live.connect).toHaveBeenCalledOnce();
    });

    it('loads the meet and opens the websocket', async () => {
      get.mockResolvedValue(meet);
      const store = useMeetStore();

      store.start();
      await flush();

      expect(live.connect).toHaveBeenCalledOnce();
      expect(get).toHaveBeenCalledOnce();
      expect(store.meet).toEqual(meet);
    });

    it('keeps the loaded meet when started again', async () => {
      get.mockResolvedValue(meet);
      const store = useMeetStore();
      store.start();
      await flush();

      store.start();
      await flush();

      expect(live.connect).toHaveBeenCalledTimes(2);
      expect(get).toHaveBeenCalledOnce();
    });

    it('keeps a meet loaded before the start', async () => {
      get.mockResolvedValue(meet);
      const store = useMeetStore();
      await store.load();

      store.start();
      await flush();

      expect(get).toHaveBeenCalledOnce();
    });

    it('reloads once the meet changes on the server', async () => {
      get.mockResolvedValue(meet);
      const store = useMeetStore();
      store.start();
      await flush();
      const newer: MeetResponse = {
        ...meet,
        loadedAt: '2026-05-01T09:00:00Z',
      };
      get.mockResolvedValue(newer);

      announceChange();
      await flush();

      expect(store.meet).toEqual(newer);
    });

    it('keeps reloading after the component that started it is unmounted', async () => {
      get.mockResolvedValue(meet);
      const store = useMeetStore();
      const componentScope = effectScope();
      componentScope.run(() => store.start());
      await flush();
      componentScope.stop();

      announceChange();
      await flush();

      expect(get).toHaveBeenCalledTimes(2);
    });
  });
});
