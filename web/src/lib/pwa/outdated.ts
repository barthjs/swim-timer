import type { Hello } from '@/stores/realtime';

/**
 * Returns a check that reports whether a greeting of the server names another version than the page
 * or other settings than the first greeting. The page reads its settings from the server, so the first greeting
 * names the settings it runs with.
 */
export function outdatedCheck(version: string): (hello: Hello) => boolean {
  let meta: string | null = null;

  return (hello) => {
    meta ??= hello.meta;
    return hello.version !== version || hello.meta !== meta;
  };
}

const RELOADED_FOR_KEY = 'pwa:reloadedFor';

/**
 * Loads the page anew once per greeting and reports whether it does,
 * since a server whose parts do not match yet would reload it endlessly.
 */
export function reloadOnce(hello: Hello): boolean {
  const key = `${hello.version} ${hello.meta}`;
  try {
    if (sessionStorage.getItem(RELOADED_FOR_KEY) === key) return false;
    sessionStorage.setItem(RELOADED_FOR_KEY, key);
  } catch {
    // Without session storage the page reloads without the guard
  }

  window.location.reload();
  return true;
}
