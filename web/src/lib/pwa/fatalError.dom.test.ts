import { createApp, effectScope, shallowRef, type EffectScope } from 'vue';
import { START_LOCATION, type RouteLocationNormalized, type Router } from 'vue-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { ApiError } from '@/api/client/errors';
import { catchFatalErrors, FATAL_RELOAD_DELAY_MS, fatalError } from '@/lib/pwa/fatalError';

type RouterErrorHandler = (error: unknown, to: Pick<RouteLocationNormalized, 'fullPath'>) => void;

/** Returns the error of a module that could not be loaded. */
function chunkError(): TypeError {
  return new TypeError(
    `Failed to fetch dynamically imported module: /assets/${faker.system.fileName()}`,
  );
}

/** Reports a promise rejection that nobody handled, like the browser. */
function rejectUnhandled(reason: unknown): void {
  const event = new Event('unhandledrejection') as PromiseRejectionEvent;
  Object.defineProperty(event, 'reason', {
    value: reason,
  });
  window.dispatchEvent(event);
}

describe('catchFatalErrors', () => {
  let scope: EffectScope;
  let app: ReturnType<typeof createApp>;
  let routerErrorHandler: RouterErrorHandler;
  const currentRoute = shallowRef<unknown>(START_LOCATION);
  const location = {
    href: '',
    assign: vi.fn<(url: string) => void>(),
    reload: vi.fn<() => void>(),
  };

  /** Catches the errors of the app and its router. */
  function run(): void {
    const router = {
      currentRoute,
      onError: (handler: RouterErrorHandler) => {
        routerErrorHandler = handler;
        return () => {};
      },
    } as unknown as Pick<Router, 'currentRoute' | 'onError'>;
    scope.run(() => catchFatalErrors(app, router));
  }

  beforeEach(() => {
    vi.useFakeTimers();
    sessionStorage.clear();
    fatalError.value = null;
    scope = effectScope();
    app = createApp({});
    currentRoute.value = {
      fullPath: `/${faker.lorem.slug()}`,
    };
    location.href = faker.internet.url();
    vi.stubGlobal('location', location);
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    scope.stop();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    vi.useRealTimers();
    location.assign.mockReset();
    location.reload.mockReset();
  });

  it('keeps an error of the app', () => {
    const error = new Error(faker.lorem.sentence());
    run();

    app.config.errorHandler?.(error, null, '');

    expect(fatalError.value).toBe(error);
  });

  it('reloads the page after the first delay', () => {
    run();
    rejectUnhandled(new Error(faker.lorem.sentence()));

    vi.advanceTimersByTime(FATAL_RELOAD_DELAY_MS - 1);
    const before = location.reload.mock.calls.length;
    vi.advanceTimersByTime(1);

    expect(before).toBe(0);
    expect(location.reload).toHaveBeenCalledOnce();
  });

  it('doubles the delay for an error that returns after the reload', () => {
    run();
    rejectUnhandled(new Error(faker.lorem.sentence()));
    vi.runAllTimers();
    location.reload.mockReset();
    fatalError.value = null;

    rejectUnhandled(new Error(faker.lorem.sentence()));
    vi.advanceTimersByTime(2 * FATAL_RELOAD_DELAY_MS - 1);
    const before = location.reload.mock.calls.length;
    vi.advanceTimersByTime(1);

    expect(before).toBe(0);
    expect(location.reload).toHaveBeenCalledOnce();
  });

  it('keeps an uncaught error of a script', () => {
    const error = new Error(faker.lorem.sentence());
    run();

    window.dispatchEvent(
      new ErrorEvent('error', {
        error,
        message: error.message,
      }),
    );

    expect(fatalError.value).toBe(error);
  });

  it('keeps an unhandled rejection', () => {
    const error = new Error(faker.lorem.sentence());
    run();

    rejectUnhandled(error);

    expect(fatalError.value).toBe(error);
  });

  it('keeps an error of a navigation', () => {
    const error = new Error(faker.lorem.sentence());
    run();

    routerErrorHandler(error, {
      fullPath: `/${faker.lorem.slug()}`,
    });

    expect(fatalError.value).toBe(error);
  });

  it('keeps only the first error', () => {
    const first = new Error(faker.lorem.sentence());
    run();

    rejectUnhandled(first);
    rejectUnhandled(new Error(faker.lorem.sentence()));

    expect(fatalError.value).toBe(first);
  });

  it('loads the target of a navigation as a new page when its module is gone', () => {
    const path = `/${faker.lorem.slug()}`;
    run();

    routerErrorHandler(chunkError(), {
      fullPath: path,
    });

    expect(location.assign).toHaveBeenCalledExactlyOnceWith(path);
    expect(fatalError.value).toBeNull();
  });

  it('loads the page anew when another module is gone', () => {
    run();

    rejectUnhandled(chunkError());

    expect(location.assign).toHaveBeenCalledExactlyOnceWith(location.href);
    expect(fatalError.value).toBeNull();
  });

  it('keeps a missing module of the starting page', () => {
    const error = chunkError();
    currentRoute.value = START_LOCATION;
    run();

    routerErrorHandler(error, {
      fullPath: `/${faker.lorem.slug()}`,
    });

    expect(location.assign).not.toHaveBeenCalled();
    expect(fatalError.value).toBe(error);
  });

  it('loads the target of a navigation as a new page when its styles are gone', () => {
    const path = `/${faker.lorem.slug()}`;
    run();

    routerErrorHandler(new Error(`Unable to preload CSS for /assets/${faker.system.fileName()}`), {
      fullPath: path,
    });

    expect(location.assign).toHaveBeenCalledExactlyOnceWith(path);
  });

  it('ignores a failed request', () => {
    run();

    rejectUnhandled(
      new ApiError({
        message: faker.lorem.sentence(),
        status: 0,
        code: 'network',
      }),
    );

    expect(fatalError.value).toBeNull();
  });

  it('ignores a delayed callback of a ResizeObserver', () => {
    run();

    window.dispatchEvent(
      new ErrorEvent('error', {
        message: 'ResizeObserver loop completed with undelivered notifications.',
      }),
    );

    expect(fatalError.value).toBeNull();
  });
});
