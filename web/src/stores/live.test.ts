import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { Lane, LiveMessage, LiveHeat } from '@/api/types/live';
import { useLiveStore } from '@/stores/live';
import { configureRealtime, useRealtimeStore } from '@/stores/realtime';

/** Records every opened websocket and lets a test drive its events. */
class FakeWebSocket extends EventTarget {
  static instances: FakeWebSocket[] = [];

  readonly url: string;

  constructor(url: string) {
    super();
    this.url = url;
    FakeWebSocket.instances.push(this);
  }

  /** Signals that the connection is established. */
  open(): void {
    this.dispatchEvent(new Event('open'));
  }

  /** Delivers a message, serializing it unless it is raw text. */
  receive(message: LiveMessage | string): void {
    this.dispatchEvent(
      new MessageEvent('message', {
        data: typeof message === 'string' ? message : JSON.stringify(message),
      }),
    );
  }

  /** Signals that the connection was closed. */
  close(): void {
    this.dispatchEvent(new Event('close'));
  }
}

/** Builds a lane without a start list entry and with the given time. */
function lane(number: number, time: number): Lane {
  return {
    number,
    firstName: '',
    lastName: '',
    birthYear: '',
    club: '',
    lap: 0,
    meter: 0,
    time,
    rank: 0,
    finished: false,
  };
}

/** Builds a live heat with the given overrides. */
function liveHeat(overrides: Partial<LiveHeat> = {}): LiveHeat {
  return {
    event: 1,
    phase: 0,
    phaseName: '',
    heat: 1,
    heatCount: 1,
    lapCount: 2,
    name: 'Event A',
    status: '',
    firstLane: 1,
    lanes: {},
    ...overrides,
  };
}

/** Returns the most recently opened websocket. */
function latestSocket(): FakeWebSocket {
  const socket = FakeWebSocket.instances.at(-1);
  if (!socket) throw new Error('no websocket opened');
  return socket;
}

describe('useLiveStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    FakeWebSocket.instances = [];
    vi.stubGlobal('WebSocket', FakeWebSocket);
    vi.useFakeTimers();
    // Unattended screens reconnect at a short fixed interval
    configureRealtime({
      url: 'ws://test/api/ws',
      reconnectBaseMs: 500,
      reconnectMaxMs: 500,
    });
  });

  afterEach(() => {
    configureRealtime({
      url: undefined,
      reconnectBaseMs: 1_000,
      reconnectMaxMs: 30_000,
    });
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  describe('connect', () => {
    it('opens the websocket of the live heat', () => {
      const store = useLiveStore();

      store.connect();

      expect(FakeWebSocket.instances).toHaveLength(1);
      expect(latestSocket().url).toBe('ws://test/api/ws');
    });

    it('keeps a single websocket when connected again', () => {
      const store = useLiveStore();
      store.connect();

      store.connect();

      expect(FakeWebSocket.instances).toHaveLength(1);
    });

    it('replaces the state with an init message', () => {
      const store = useLiveStore();
      store.connect();

      latestSocket().receive({
        type: 'init',
        serverTime: '2026-10-01T10:00:00Z',
        data: liveHeat(),
      });

      expect(store.state?.heat).toBe(1);
    });

    it('takes the live heat the server sent before the store subscribed', () => {
      useRealtimeStore().connect();
      latestSocket().receive({
        type: 'init',
        serverTime: '2026-10-01T10:00:00Z',
        data: liveHeat(),
      });
      latestSocket().receive({
        type: 'update',
        serverTime: '2026-10-01T10:00:01Z',
        data: liveHeat({
          heat: 2,
        }),
      });
      const store = useLiveStore();

      store.connect();

      expect(store.state?.heat).toBe(2);
    });

    it('replaces the state with an update message', () => {
      const store = useLiveStore();
      store.connect();
      latestSocket().receive({
        type: 'init',
        serverTime: '2026-10-01T10:00:00Z',
        data: liveHeat(),
      });

      latestSocket().receive({
        type: 'update',
        serverTime: '2026-10-01T10:00:01Z',
        data: liveHeat({
          heat: 2,
        }),
      });

      expect(store.state?.heat).toBe(2);
    });

    it('replaces the state with an init without a heat while no heat is known', () => {
      const store = useLiveStore();
      store.connect();

      latestSocket().receive({
        type: 'init',
        serverTime: '2026-10-01T10:00:00Z',
        data: liveHeat(),
      });

      expect(store.state?.status).toBe('');
    });

    it('keeps a known heat when the init after a server restart carries no heat', () => {
      const store = useLiveStore();
      store.connect();
      latestSocket().receive({
        type: 'update',
        serverTime: '2026-10-01T10:00:00Z',
        data: liveHeat({
          status: 'official',
        }),
      });
      latestSocket().close();
      vi.advanceTimersByTime(500);

      latestSocket().receive({
        type: 'init',
        serverTime: '2026-10-01T10:05:00Z',
        data: liveHeat({
          event: 0,
          heat: 0,
          name: '',
        }),
      });

      expect(store.state?.status).toBe('official');
      expect(store.state?.name).toBe('Event A');
    });

    it('replaces a kept heat with the next update', () => {
      const store = useLiveStore();
      store.connect();
      latestSocket().receive({
        type: 'update',
        serverTime: '2026-10-01T10:00:00Z',
        data: liveHeat({
          status: 'official',
        }),
      });
      latestSocket().close();
      vi.advanceTimersByTime(500);
      latestSocket().receive({
        type: 'init',
        serverTime: '2026-10-01T10:05:00Z',
        data: liveHeat({
          event: 0,
          heat: 0,
          name: '',
        }),
      });

      latestSocket().receive({
        type: 'update',
        serverTime: '2026-10-01T10:05:01Z',
        data: liveHeat({
          heat: 2,
          status: 'start',
        }),
      });

      expect(store.state?.heat).toBe(2);
    });

    it('keeps the last state when a message cannot be parsed', () => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      const store = useLiveStore();
      store.connect();
      latestSocket().receive({
        type: 'init',
        serverTime: '2026-10-01T10:00:00Z',
        data: liveHeat(),
      });

      latestSocket().receive('not json');

      expect(store.state?.name).toBe('Event A');
    });

    it('reconnects after the websocket closes and keeps the last state', () => {
      const store = useLiveStore();
      store.connect();
      latestSocket().open();
      latestSocket().receive({
        type: 'init',
        serverTime: '2026-10-01T10:00:00Z',
        data: liveHeat(),
      });

      latestSocket().close();
      vi.advanceTimersByTime(500);

      expect(store.state).not.toBeNull();
      expect(FakeWebSocket.instances).toHaveLength(2);
    });
  });

  describe('disconnect', () => {
    it('closes the websocket and stops reconnecting', () => {
      const store = useLiveStore();
      store.connect();
      latestSocket().open();

      store.disconnect();
      vi.advanceTimersByTime(500);

      expect(FakeWebSocket.instances).toHaveLength(1);
    });

    it('cancels a pending reconnect', () => {
      const store = useLiveStore();
      store.connect();
      latestSocket().close();

      store.disconnect();
      vi.advanceTimersByTime(500);

      expect(FakeWebSocket.instances).toHaveLength(1);
    });
  });

  describe('displayTime', () => {
    it('returns the highest lane time', () => {
      const store = useLiveStore();
      store.connect();

      latestSocket().receive({
        type: 'update',
        serverTime: '2026-10-01T10:00:00Z',
        data: liveHeat({
          lanes: {
            '1': lane(1, 61230),
            '2': lane(2, 59990),
          },
        }),
      });

      expect(store.displayTime).toBe('01:01,23');
    });

    it('returns zero without a heat', () => {
      const store = useLiveStore();

      const displayTime = store.displayTime;

      expect(displayTime).toBe('00:00,00');
    });

    it('returns zero while no lane has a time', () => {
      const store = useLiveStore();
      store.connect();

      latestSocket().receive({
        type: 'update',
        serverTime: '2026-10-01T10:00:00Z',
        data: liveHeat({
          lanes: {
            '1': lane(1, 0),
            '2': lane(2, 0),
          },
        }),
      });

      expect(store.displayTime).toBe('00:00,00');
    });
  });

  describe('startTimeLocal', () => {
    it('returns the start time on the local clock using the server time', () => {
      vi.spyOn(performance, 'now').mockReturnValue(1000);
      const store = useLiveStore();
      store.connect();
      latestSocket().receive({
        type: 'update',
        serverTime: '2026-10-01T10:00:10.000Z',
        data: liveHeat({
          status: 'start',
          startTime: '2026-10-01T10:00:04.000Z',
        }),
      });

      const startTime = store.startTimeLocal();

      expect(startTime).toBe(1000 - 6000);
    });

    it('returns null without a start', () => {
      const store = useLiveStore();
      store.connect();

      const startTime = store.startTimeLocal();

      expect(startTime).toBeNull();
    });
  });
});
