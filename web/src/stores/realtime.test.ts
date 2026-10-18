import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  configureRealtime,
  reconnectDelay,
  RESYNC_MESSAGE,
  useRealtimeStore,
  type RealtimeHandler,
} from '@/stores/realtime';

/** FakeWebSocket records the sockets the store opens and lets a test drive their events. */
class FakeWebSocket {
  static instances: FakeWebSocket[] = [];

  url: string;
  closed = false;
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: unknown }) => void) | null = null;
  onclose: (() => void) | null = null;

  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
  }

  /** close marks the socket closed and reports it like a browser. */
  close(): void {
    this.closed = true;
    this.onclose?.();
  }

  /** latest returns the socket opened last. */
  static latest(): FakeWebSocket {
    const socket = FakeWebSocket.instances.at(-1);
    if (!socket) throw new Error('no socket was opened');
    return socket;
  }
}

beforeEach(() => {
  setActivePinia(createPinia());
  FakeWebSocket.instances = [];
  vi.stubGlobal('WebSocket', FakeWebSocket);
  vi.stubGlobal('location', {
    protocol: 'https:',
    host: 'example.test',
  });
});

afterEach(() => {
  configureRealtime({
    url: undefined,
    reconnectBaseMs: 1_000,
    reconnectMaxMs: 30_000,
  });
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe('configureRealtime', () => {
  it('opens the configured url', () => {
    configureRealtime({
      url: 'ws://localhost:8080/api/ws',
    });
    const store = useRealtimeStore();

    store.connect();

    expect(FakeWebSocket.latest().url).toBe('ws://localhost:8080/api/ws');
  });
});

describe('useRealtimeStore', () => {
  describe('reconnects', () => {
    it('counts the connections opened after a lost or failed one', async () => {
      vi.useFakeTimers();
      const store = useRealtimeStore();
      store.connect();
      FakeWebSocket.latest().onopen?.();
      FakeWebSocket.latest().onclose?.();
      await vi.advanceTimersByTimeAsync(1_000);
      FakeWebSocket.latest().onclose?.();
      await vi.advanceTimersByTimeAsync(2_000);

      FakeWebSocket.latest().onopen?.();

      expect(store.reconnects).toBe(1);
    });

    it('counts the first connection after a failed attempt', async () => {
      vi.useFakeTimers();
      const store = useRealtimeStore();
      store.connect();
      FakeWebSocket.latest().onclose?.();
      await vi.advanceTimersByTimeAsync(1_000);

      FakeWebSocket.latest().onopen?.();

      expect(store.reconnects).toBe(1);
    });

    it('does not count the first connection or one after a disconnect', () => {
      const store = useRealtimeStore();
      store.connect();
      FakeWebSocket.latest().onopen?.();
      store.disconnect();
      store.connect();

      FakeWebSocket.latest().onopen?.();

      expect(store.reconnects).toBe(0);
    });
  });

  describe('connect', () => {
    it('opens the realtime endpoint on the origin of the page and resyncs once open', () => {
      const store = useRealtimeStore();
      const resync = vi.fn<RealtimeHandler>();
      store.on(RESYNC_MESSAGE, resync);

      store.connect();
      FakeWebSocket.latest().onopen?.();

      expect(FakeWebSocket.latest().url).toBe('wss://example.test/api/ws');
      expect(store.isConnected).toBe(true);
      expect(resync).toHaveBeenCalledOnce();
    });

    it('keeps an open connection', () => {
      const store = useRealtimeStore();

      store.connect();
      store.connect();

      expect(FakeWebSocket.instances).toHaveLength(1);
    });

    it('opens a lost connection again after a delay', async () => {
      vi.useFakeTimers();
      const store = useRealtimeStore();
      store.connect();
      FakeWebSocket.latest().onopen?.();

      FakeWebSocket.latest().onclose?.();
      await vi.advanceTimersByTimeAsync(1_000);

      expect(store.isConnected).toBe(false);
      expect(FakeWebSocket.instances).toHaveLength(2);
    });
  });

  describe('disconnect', () => {
    it('closes the connection without opening it again', async () => {
      vi.useFakeTimers();
      const store = useRealtimeStore();
      store.connect();

      store.disconnect();
      await vi.advanceTimersByTimeAsync(60_000);

      expect(FakeWebSocket.instances).toHaveLength(1);
      expect(FakeWebSocket.latest().closed).toBe(true);
      expect(store.isConnected).toBe(false);
    });
  });

  describe('reconnect', () => {
    it('replaces the connection with a new one', () => {
      const store = useRealtimeStore();
      store.connect();
      const first = FakeWebSocket.latest();

      store.reconnect();

      expect(first.closed).toBe(true);
      expect(FakeWebSocket.instances).toHaveLength(2);
    });
  });

  describe('on', () => {
    it('hands a message to the handlers of its type', () => {
      const store = useRealtimeStore();
      const changed = vi.fn<RealtimeHandler>();
      const other = vi.fn<RealtimeHandler>();
      store.on('notifications.changed', changed);
      store.on('other', other);
      store.connect();

      FakeWebSocket.latest().onmessage?.({
        data: '{"type":"notifications.changed"}',
      });

      expect(changed).toHaveBeenCalledWith({
        type: 'notifications.changed',
      });
      expect(other).not.toHaveBeenCalled();
    });

    it('skips a message that is not a message of the server', () => {
      const store = useRealtimeStore();
      const handler = vi.fn<RealtimeHandler>();
      store.on('notifications.changed', handler);
      store.connect();

      FakeWebSocket.latest().onmessage?.({
        data: 'not json',
      });
      FakeWebSocket.latest().onmessage?.({
        data: '{"kind":"notifications.changed"}',
      });

      expect(handler).not.toHaveBeenCalled();
    });

    it('returns the function that removes the handler', () => {
      const store = useRealtimeStore();
      const handler = vi.fn<RealtimeHandler>();
      const off = store.on('notifications.changed', handler);
      store.connect();

      off();
      FakeWebSocket.latest().onmessage?.({
        data: '{"type":"notifications.changed"}',
      });

      expect(handler).not.toHaveBeenCalled();
    });
  });
});

describe('reconnectDelay', () => {
  it('doubles the delay with every failure up to the cap', () => {
    vi.spyOn(Math, 'random').mockReturnValue(1);

    const delays = [1, 2, 3, 6, 10].map(reconnectDelay);

    expect(delays).toEqual([1_000, 2_000, 4_000, 30_000, 30_000]);
  });

  it('follows the configured delays', () => {
    vi.spyOn(Math, 'random').mockReturnValue(1);
    configureRealtime({
      reconnectBaseMs: 500,
      reconnectMaxMs: 500,
    });

    const delays = [1, 5].map(reconnectDelay);

    expect(delays).toEqual([500, 500]);
  });

  it('spreads the delay over its upper half', () => {
    vi.spyOn(Math, 'random').mockReturnValue(0);

    const delay = reconnectDelay(2);

    expect(delay).toBe(1_000);
  });
});
