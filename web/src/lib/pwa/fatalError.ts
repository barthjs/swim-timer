import { shallowRef, type App } from 'vue';
import { START_LOCATION, type Router } from 'vue-router';
import { useEventListener } from '@vueuse/core';

import { ApiError } from '@/api/client/errors';

/**
 * Browsers report a delayed ResizeObserver callback as an error event,
 * although they recover by the next frame, so it is no crash.
 */
const RESIZE_OBSERVER_LOOP_MESSAGE = 'ResizeObserver loop';
const CHUNK_ERROR_MESSAGE =
  /dynamically imported module|Importing a module script failed|Unable to preload CSS/;

/**
 * A crashed page reloads on its own after the first delay, and an error that returns after the reload doubles it
 * up to the maximum, so a bug that crashes every load cannot loop quickly. Reloads within the window count.
 */
export const FATAL_RELOAD_DELAY_MS = 10_000;
const FATAL_RELOAD_MAX_DELAY_MS = 5 * 60_000;
const FATAL_RELOAD_WINDOW_MS = 2 * FATAL_RELOAD_MAX_DELAY_MS;
const FATAL_RELOADS_KEY = 'pwa:fatalReloads';

/** Holds the first uncaught error, after which only a reload brings the page back to a known state. */
export const fatalError = shallowRef<unknown>(null);

/**
 * Keeps every uncaught error but failed requests in fatalError and reloads the page after a delay,
 * since an unattended screen has nobody to press the button of the error screen.
 */
export function catchFatalErrors(app: App, router: Pick<Router, 'currentRoute' | 'onError'>): void {
  let leaving = false;

  /** Keeps an uncaught error, or leaves for the target when a module is missing. */
  function report(error: unknown, target: string): void {
    // A failed request is no crash, since the page that sent it stays intact
    if (leaving || error instanceof ApiError || fatalError.value !== null) return;

    // A module of an older build is gone from the server after an update, so its target loads as a new page,
    // except while the page starts, where the new page would fail the same way.
    if (isChunkError(error) && router.currentRoute.value !== START_LOCATION) {
      leaving = true;
      window.location.assign(target);
      return;
    }
    fatalError.value = error;
    setTimeout(() => window.location.reload(), fatalReloadDelay(Date.now()));
  }

  // A handler a framework set before replaces nothing, since the error screen is the one answer to a crash
  app.config.errorHandler = (error) => {
    console.error(error);
    report(error, window.location.href);
  };
  router.onError((error: unknown, to) => report(error, to.fullPath));
  useEventListener(window, 'error', (event: ErrorEvent) => {
    if (event.message.includes(RESIZE_OBSERVER_LOOP_MESSAGE)) return;
    report(event.error ?? event.message, window.location.href);
  });
  useEventListener(window, 'unhandledrejection', (event: PromiseRejectionEvent) =>
    report(event.reason, window.location.href),
  );
}

/** Reports whether an error tells that a module or its styles could not be loaded. */
function isChunkError(error: unknown): boolean {
  return error instanceof Error && CHUNK_ERROR_MESSAGE.test(error.message);
}

/**
 * Records a reload after a crash at the given time and returns the delay before it,
 * doubled for every reload after a crash within the window. Without session storage, it returns the first delay.
 */
function fatalReloadDelay(now: number): number {
  let reloads: number[] = [];
  try {
    const stored: unknown = JSON.parse(sessionStorage.getItem(FATAL_RELOADS_KEY) ?? '[]');
    if (Array.isArray(stored)) {
      reloads = stored.filter(
        (at): at is number => typeof at === 'number' && now - at < FATAL_RELOAD_WINDOW_MS,
      );
    }
    sessionStorage.setItem(FATAL_RELOADS_KEY, JSON.stringify([...reloads, now]));
  } catch {
    // The page reloads after the first delay
  }

  return Math.min(FATAL_RELOAD_DELAY_MS * 2 ** reloads.length, FATAL_RELOAD_MAX_DELAY_MS);
}
