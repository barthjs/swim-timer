import { defineStore } from 'pinia';
import { ref } from 'vue';

import { isRecord } from '@/lib/types/guards';

/** A message the server sends over the realtime connection, named by its type. */
export type RealtimeMessage = {
  type: string;
};

/** Receives the messages of one type. */
export type RealtimeHandler<T extends RealtimeMessage = RealtimeMessage> = (message: T) => void;

/** Configures the connection of the app. */
export type RealtimeOptions = {
  // The WebSocket URL, the realtime endpoint on the origin of the page when unset
  url?: string;
  // The delay before the first reconnect, each further failure doubles it up to reconnectMaxMs
  reconnectBaseMs: number;
  reconnectMaxMs: number;
};

/** Tells the handlers to reload their state, since messages may have been missed. */
export const RESYNC_MESSAGE = 'resync';

const options: RealtimeOptions = {
  reconnectBaseMs: 1_000,
  reconnectMaxMs: 30_000,
};

/** Changes the options of the connection, once before the app connects. */
export function configureRealtime(changes: Partial<RealtimeOptions>): void {
  Object.assign(options, changes);
}

/**
 * Keeps the realtime connection to the server open and hands every message to the handlers of its type.
 * A lost connection is opened again with backoff, and every handler of RESYNC_MESSAGE runs
 * after each connect, since the server does not repeat what was sent while the connection was down.
 */
export const useRealtimeStore = defineStore('realtime', () => {
  const isConnected = ref(false);
  // Counts the connections opened after a lost or failed one
  const reconnects = ref(0);

  const handlers = new Map<string, Set<RealtimeHandler>>();
  let socket: WebSocket | null = null;
  let wanted = false;
  let failures = 0;
  // Set once a connection closed without being asked to, so the next open counts as a reconnect
  let lostConnection = false;
  let reconnectTimer: ReturnType<typeof setTimeout> | undefined;

  /** Opens the connection unless it is already open. */
  function connect(): void {
    wanted = true;
    if (socket) return;
    open();
  }

  /** Closes the connection and stops reconnecting. */
  function disconnect(): void {
    wanted = false;
    lostConnection = false;
    clearTimeout(reconnectTimer);
    const current = socket;
    socket = null;
    isConnected.value = false;
    current?.close();
  }

  /** Opens a new connection, so the server assigns it the topics of the current session. */
  function reconnect(): void {
    disconnect();
    connect();
  }

  /**
   * Registers a handler for the messages of a type and returns the function that removes it.
   * The type parameter declares the shape the server sends for that type.
   */
  function on<T extends RealtimeMessage>(type: string, handler: RealtimeHandler<T>): () => void {
    let set = handlers.get(type);
    if (!set) {
      set = new Set();
      handlers.set(type, set);
    }
    // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- messages of a type have the shape its handlers declare
    const registered = handler as RealtimeHandler;
    set.add(registered);
    return () => set.delete(registered);
  }

  /** Creates the socket and wires its events. Events of a replaced socket are ignored. */
  function open(): void {
    const current = new WebSocket(options.url ?? sameOriginURL());
    socket = current;

    current.addEventListener('open', () => {
      if (socket !== current) return;
      failures = 0;
      isConnected.value = true;
      if (lostConnection) {
        lostConnection = false;
        reconnects.value++;
      }
      dispatch({
        type: RESYNC_MESSAGE,
      });
    });
    current.addEventListener('message', (event: MessageEvent) => {
      if (socket !== current) return;
      const message = parseMessage(event.data);
      if (message) dispatch(message);
    });
    current.addEventListener('close', () => {
      if (socket !== current) return;
      socket = null;
      isConnected.value = false;
      lostConnection = true;
      if (wanted) scheduleReconnect();
    });
  }

  /** Hands a message to the handlers of its type. */
  function dispatch(message: RealtimeMessage): void {
    for (const handler of handlers.get(message.type) ?? []) {
      handler(message);
    }
  }

  /** Opens the connection again after a delay that doubles with every failure. */
  function scheduleReconnect(): void {
    failures++;
    reconnectTimer = setTimeout(() => {
      if (wanted && !socket) open();
    }, reconnectDelay(failures));
  }

  return {
    isConnected,
    reconnects,
    connect,
    disconnect,
    reconnect,
    on,
  };
});

/** Returns the WebSocket URL of the realtime endpoint on the origin of the page. */
function sameOriginURL(): string {
  const protocol = globalThis.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${globalThis.location.host}/api/ws`;
}

/** Reads a message of the server, or returns null for anything else. */
function parseMessage(raw: unknown): RealtimeMessage | null {
  if (typeof raw !== 'string') return null;
  try {
    const value: unknown = JSON.parse(raw);
    if (isRecord(value) && typeof value.type === 'string') {
      return {
        ...value,
        type: value.type,
      };
    }
  } catch {
    // A message that is not JSON is skipped like any other unknown message
  }
  return null;
}

/**
 * Returns the delay before the next connect, doubling with every failure up to the maximum,
 * with jitter, so many clients that lost the server at once do not return at the same moment.
 */
export function reconnectDelay(failures: number): number {
  const base = Math.min(
    options.reconnectBaseMs * 2 ** Math.max(failures - 1, 0),
    options.reconnectMaxMs,
  );
  return base / 2 + Math.random() * (base / 2);
}
