import { defineStore } from 'pinia';
import { ref } from 'vue';

/** RealtimeMessage is a message the server sends over the realtime connection, named by its type. */
export type RealtimeMessage = {
  type: string;
};

/** RealtimeHandler receives the messages of one type. */
export type RealtimeHandler<T extends RealtimeMessage = RealtimeMessage> = (message: T) => void;

/** RealtimeOptions configures the connection of the app. */
export type RealtimeOptions = {
  // The WebSocket URL, the realtime endpoint on the origin of the page when unset
  url?: string;
  // The delay before the first reconnect, each further failure doubles it up to reconnectMaxMs
  reconnectBaseMs: number;
  reconnectMaxMs: number;
};

/** RESYNC_MESSAGE tells the handlers to reload their state, since messages may have been missed. */
export const RESYNC_MESSAGE = 'resync';

const options: RealtimeOptions = {
  reconnectBaseMs: 1_000,
  reconnectMaxMs: 30_000,
};

/**
 * configureRealtime changes the options of the connection. The app calls it once before it connects,
 * for example to point the development server at the backend or to reconnect faster.
 */
export function configureRealtime(changes: Partial<RealtimeOptions>): void {
  Object.assign(options, changes);
}

/**
 * useRealtimeStore keeps the realtime connection to the server open and hands every message to the handlers
 * of its type. A lost connection is opened again with backoff, and every handler of RESYNC_MESSAGE runs
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

  /** connect opens the connection unless it is already open. */
  function connect(): void {
    wanted = true;
    if (socket) return;
    open();
  }

  /** disconnect closes the connection and stops reconnecting. */
  function disconnect(): void {
    wanted = false;
    lostConnection = false;
    clearTimeout(reconnectTimer);
    const current = socket;
    socket = null;
    isConnected.value = false;
    current?.close();
  }

  /** reconnect opens a new connection, so the server assigns it the topics of the current session. */
  function reconnect(): void {
    disconnect();
    connect();
  }

  /**
   * on registers a handler for the messages of a type and returns the function that removes it.
   * The type parameter declares the shape the server sends for that type.
   */
  function on<T extends RealtimeMessage>(type: string, handler: RealtimeHandler<T>): () => void {
    let set = handlers.get(type);
    if (!set) {
      set = new Set();
      handlers.set(type, set);
    }
    const registered = handler as RealtimeHandler;
    set.add(registered);
    return () => set.delete(registered);
  }

  /** open creates the socket and wires its events. Events of a replaced socket are ignored. */
  function open(): void {
    const current = new WebSocket(options.url ?? sameOriginURL());
    socket = current;

    current.onopen = () => {
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
    };
    current.onmessage = (event: MessageEvent) => {
      if (socket !== current) return;
      const message = parseMessage(event.data);
      if (message) dispatch(message);
    };
    current.onclose = () => {
      if (socket !== current) return;
      socket = null;
      isConnected.value = false;
      lostConnection = true;
      if (wanted) scheduleReconnect();
    };
  }

  /** dispatch hands a message to the handlers of its type. */
  function dispatch(message: RealtimeMessage): void {
    for (const handler of handlers.get(message.type) ?? []) {
      handler(message);
    }
  }

  /** scheduleReconnect opens the connection again after a delay that doubles with every failure. */
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

/** sameOriginURL returns the WebSocket URL of the realtime endpoint on the origin of the page. */
function sameOriginURL(): string {
  const protocol = globalThis.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${globalThis.location.host}/api/ws`;
}

/** parseMessage reads a message of the server, or returns null for anything else. */
function parseMessage(raw: unknown): RealtimeMessage | null {
  if (typeof raw !== 'string') return null;
  try {
    const value: unknown = JSON.parse(raw);
    if (
      typeof value === 'object' &&
      value !== null &&
      typeof (value as RealtimeMessage).type === 'string'
    ) {
      return value as RealtimeMessage;
    }
  } catch {
    // A message that is not JSON is skipped like any other unknown message
  }
  return null;
}

/**
 * reconnectDelay returns the delay before the next connect, doubling with every failure up to the maximum,
 * with jitter, so many clients that lost the server at once do not return at the same moment.
 */
export function reconnectDelay(failures: number): number {
  const base = Math.min(
    options.reconnectBaseMs * 2 ** Math.max(failures - 1, 0),
    options.reconnectMaxMs,
  );
  return base / 2 + Math.random() * (base / 2);
}
