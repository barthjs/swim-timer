import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  configureRealtime,
  reconnectDelay,
  RESYNC_MESSAGE,
  useRealtimeStore,
  type RealtimeHandler,
} from '@/stores/realtime';

/** Records the sockets the store opens and lets a test drive their events. */
class FakeWebSocket extends EventTarget {
  static instances: FakeWebSocket[] = [];

  url: string;
  closed = false;

  constructor(url: string) {
    super();
    this.url = url;
    FakeWebSocket.instances.push(this);
  }

  /** Signals that the connection is established. */
  open(): void {
    this.dispatchEvent(new Event('open'));
  }

  /** Delivers a message with the given data. */
  receive(data: unknown): void {
    this.dispatchEvent(
      new MessageEvent('message', {
        data,
      }),
    );
  }

  /** Marks the socket closed and reports it like a browser. */
  close(): void {
    this.closed = true;
    this.dispatchEvent(new Event('close'));
  }

  /** Returns the socket opened last. */
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
  describe('connect', () => {
    it('opens the realtime endpoint on the origin of the page and resyncs once open', () => {
      const store = useRealtimeStore();
      const resync = vi.fn<RealtimeHandler>();
      store.on(RESYNC_MESSAGE, resync);

      store.connect();
      FakeWebSocket.latest().open();

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
      FakeWebSocket.latest().open();

      FakeWebSocket.latest().close();
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
      store.on('status', changed);
      store.on('other', other);
      store.connect();

      FakeWebSocket.latest().receive('{"type":"status"}');

      expect(changed).toHaveBeenCalledWith({
        type: 'status',
      });
      expect(other).not.toHaveBeenCalled();
    });

    it('hands a resync of the server to the handlers after a random part of the jitter', () => {
      vi.useFakeTimers();
      vi.spyOn(Math, 'random').mockReturnValue(1);
      const store = useRealtimeStore();
      const handler = vi.fn<RealtimeHandler>();
      store.on(RESYNC_MESSAGE, handler);
      store.connect();

      FakeWebSocket.latest().receive('{"type":"resync"}');
      vi.advanceTimersByTime(2_999);
      const before = handler.mock.calls.length;
      vi.advanceTimersByTime(1);

      expect(before).toBe(0);
      expect(handler).toHaveBeenCalledOnce();
    });

    it('skips a message that is not a message of the server', () => {
      const store = useRealtimeStore();
      const handler = vi.fn<RealtimeHandler>();
      store.on('status', handler);
      store.connect();

      FakeWebSocket.latest().receive('not json');
      FakeWebSocket.latest().receive('{"kind":"status"}');

      expect(handler).not.toHaveBeenCalled();
    });

    it('returns the function that removes the handler', () => {
      const store = useRealtimeStore();
      const handler = vi.fn<RealtimeHandler>();
      const off = store.on('status', handler);
      store.connect();

      off();
      FakeWebSocket.latest().receive('{"type":"status"}');

      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe('onState', () => {
    it('hands the latest message of the type at once and every later one', () => {
      const store = useRealtimeStore();
      const handler = vi.fn<RealtimeHandler>();
      store.connect();
      FakeWebSocket.latest().receive('{"type":"status","data":1}');
      FakeWebSocket.latest().receive('{"type":"status","data":2}');

      store.onState('status', handler);
      FakeWebSocket.latest().receive('{"type":"status","data":3}');

      expect(handler.mock.calls).toEqual([
        [
          {
            type: 'status',
            data: 2,
          },
        ],
        [
          {
            type: 'status',
            data: 3,
          },
        ],
      ]);
    });

    it('hands the newest message among several types at once', () => {
      const store = useRealtimeStore();
      const handler = vi.fn<RealtimeHandler>();
      store.connect();
      FakeWebSocket.latest().receive('{"type":"update","data":1}');
      FakeWebSocket.latest().receive('{"type":"init","data":2}');

      store.onState(['init', 'update'], handler);

      expect(handler).toHaveBeenCalledExactlyOnceWith({
        type: 'init',
        data: 2,
      });
    });

    it('waits for the first message of a type the server did not send yet', () => {
      const store = useRealtimeStore();
      const handler = vi.fn<RealtimeHandler>();

      store.onState('status', handler);

      expect(handler).not.toHaveBeenCalled();
    });

    it('returns the function that removes the handler', () => {
      const store = useRealtimeStore();
      const handler = vi.fn<RealtimeHandler>();
      const off = store.onState('status', handler);
      store.connect();

      off();
      FakeWebSocket.latest().receive('{"type":"status"}');

      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe('onChange', () => {
    /** Delivers the change of a resource like the server. */
    function announce(resource: string, id?: string): void {
      FakeWebSocket.latest().receive(
        JSON.stringify({
          type: 'changed',
          data: {
            resource,
            id,
          },
        }),
      );
    }

    it('hands the announced change of the resource with its item to the handler', () => {
      const store = useRealtimeStore();
      const handler = vi.fn<(id?: string) => void>();
      store.onChange('preset', handler);
      store.connect();

      announce('preset', 'main');

      expect(handler).toHaveBeenCalledExactlyOnceWith('main');
    });

    it('runs the handler after every connect', () => {
      const store = useRealtimeStore();
      const handler = vi.fn<(id?: string) => void>();
      store.onChange('preset', handler);
      store.connect();

      FakeWebSocket.latest().open();

      expect(handler).toHaveBeenCalledExactlyOnceWith();
    });

    it('skips the changes of other resources', () => {
      const store = useRealtimeStore();
      const handler = vi.fn<(id?: string) => void>();
      store.onChange('preset', handler);
      store.connect();

      announce('meet');

      expect(handler).not.toHaveBeenCalled();
    });

    it('returns the function that removes the handler', () => {
      const store = useRealtimeStore();
      const handler = vi.fn<(id?: string) => void>();
      const off = store.onChange('preset', handler);
      store.connect();

      off();
      FakeWebSocket.latest().open();
      announce('preset');

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
