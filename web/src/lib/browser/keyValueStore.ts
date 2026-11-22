/**
 * Keeps values in the browser by key, and its operations succeed without storing anything
 * where the browser refuses IndexedDB.
 */
export type KeyValueStore<T> = {
  get: (key: string) => Promise<T | null>;
  set: (key: string, value: T) => Promise<void>;
  delete: (key: string) => Promise<void>;
};

/** The version of the database layout, raised when the object stores change. */
const DATABASE_VERSION = 1;

/**
 * Opens an object store of an IndexedDB database on first use.
 * The factory is the IndexedDB of the browser, or undefined where there is none.
 */
export function createKeyValueStore<T>(
  database: string,
  store: string,
  factory: () => IDBFactory | undefined = () => globalThis.indexedDB,
): KeyValueStore<T> {
  let opening: Promise<IDBDatabase> | null = null;

  /** Opens the database once and creates the object store on the first visit. */
  async function open(): Promise<IDBDatabase> {
    if (opening) return opening;

    opening = new Promise<IDBDatabase>((resolve, reject) => {
      const indexedDB = factory();
      if (!indexedDB) {
        reject(new Error('IndexedDB is not available'));
        return;
      }
      const request = indexedDB.open(database, DATABASE_VERSION);
      request.addEventListener('upgradeneeded', () => {
        if (!request.result.objectStoreNames.contains(store)) {
          request.result.createObjectStore(store);
        }
      });
      request.addEventListener('success', () => resolve(request.result));
      request.addEventListener('error', () =>
        reject(request.error ?? new Error('IndexedDB could not be opened')),
      );
    });
    // A failed open is tried again next time, since the browser may allow it later
    opening.catch(() => {
      opening = null;
    });
    return opening;
  }

  /** Performs one request in a transaction of the object store and resolves with its result. */
  async function run<R>(
    mode: IDBTransactionMode,
    operation: (objectStore: IDBObjectStore) => IDBRequest<R>,
  ): Promise<R> {
    const db = await open();
    return new Promise<R>((resolve, reject) => {
      const request = operation(db.transaction(store, mode).objectStore(store));
      request.addEventListener('success', () => resolve(request.result));
      request.addEventListener('error', () =>
        reject(request.error ?? new Error('IndexedDB request failed')),
      );
    });
  }

  /** Reads the value of a key, or null if it is missing or cannot be read. */
  async function get(key: string): Promise<T | null> {
    try {
      const value = await run<unknown>('readonly', (objectStore) => objectStore.get(key));
      // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- the store only holds the values it wrote
      return value === undefined ? null : (value as T);
    } catch {
      return null;
    }
  }

  /** Stores the value of a key if the browser allows it. */
  async function set(key: string, value: T): Promise<void> {
    try {
      await run('readwrite', (objectStore) => objectStore.put(value, key));
    } catch {
      // Without storage, the value only lives in memory
    }
  }

  /** Deletes the value of a key if the browser allows it. */
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
