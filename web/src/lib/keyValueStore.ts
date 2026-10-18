/**
 * KeyValueStore keeps values in the browser by key. Every operation succeeds without storing anything
 * if the browser offers no IndexedDB or refuses it, for example in a private window,
 * so callers keep working with the values they hold in memory.
 */
export type KeyValueStore<T> = {
  get: (key: string) => Promise<T | null>;
  set: (key: string, value: T) => Promise<void>;
  delete: (key: string) => Promise<void>;
};

/** The version of the database layout, raised when the object stores change. */
const DATABASE_VERSION = 1;

/**
 * createKeyValueStore opens an object store of an IndexedDB database on first use.
 * The factory is the IndexedDB of the browser, or undefined where there is none.
 */
export function createKeyValueStore<T>(
  database: string,
  store: string,
  factory: () => IDBFactory | undefined = () => globalThis.indexedDB,
): KeyValueStore<T> {
  let opening: Promise<IDBDatabase> | null = null;

  /** open opens the database once and creates the object store on the first visit. */
  function open(): Promise<IDBDatabase> {
    if (opening) return opening;

    opening = new Promise<IDBDatabase>((resolve, reject) => {
      const indexedDB = factory();
      if (!indexedDB) {
        reject(new Error('IndexedDB is not available'));
        return;
      }
      const request = indexedDB.open(database, DATABASE_VERSION);
      request.onupgradeneeded = () => {
        if (!request.result.objectStoreNames.contains(store)) {
          request.result.createObjectStore(store);
        }
      };
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error ?? new Error('IndexedDB could not be opened'));
    });
    // A failed open is tried again next time, since the browser may allow it later
    opening.catch(() => {
      opening = null;
    });
    return opening;
  }

  /** run performs one request in a transaction of the object store and resolves with its result. */
  async function run<R>(
    mode: IDBTransactionMode,
    operation: (objectStore: IDBObjectStore) => IDBRequest<R>,
  ): Promise<R> {
    const db = await open();
    return new Promise<R>((resolve, reject) => {
      const request = operation(db.transaction(store, mode).objectStore(store));
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'));
    });
  }

  /** get reads the value of a key, or null if it is missing or cannot be read. */
  async function get(key: string): Promise<T | null> {
    try {
      const value = await run<unknown>('readonly', (objectStore) => objectStore.get(key));
      return value === undefined ? null : (value as T);
    } catch {
      return null;
    }
  }

  /** set stores the value of a key if the browser allows it. */
  async function set(key: string, value: T): Promise<void> {
    try {
      await run('readwrite', (objectStore) => objectStore.put(value, key));
    } catch {
      // Without storage, the value only lives in memory
    }
  }

  /** remove deletes the value of a key if the browser allows it. */
  async function remove(key: string): Promise<void> {
    try {
      await run('readwrite', (objectStore) => objectStore.delete(key));
    } catch {
      // Without storage, there is nothing to delete
    }
  }

  return {
    get,
    set,
    delete: remove,
  };
}
