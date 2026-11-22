import { afterEach, describe, expect, it, vi } from 'vitest';

import { registerServiceWorker } from '@/lib/pwa/serviceWorker';

/** Gives the navigator a service worker container whose register resolves or fails. */
function stubServiceWorker(registers: boolean) {
  const register = vi.fn<(url: string) => Promise<unknown>>(async () =>
    registers ? Promise.resolve({}) : Promise.reject(new Error('insecure')),
  );
  Object.defineProperty(navigator, 'serviceWorker', {
    value: {
      register,
    },
    configurable: true,
  });
  return register;
}

describe('registerServiceWorker', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    Reflect.deleteProperty(navigator, 'serviceWorker');
  });

  it('registers the service worker of a production build', () => {
    vi.stubEnv('PROD', true);
    const register = stubServiceWorker(true);

    registerServiceWorker();

    expect(register).toHaveBeenCalledWith('/sw.js');
  });

  it('logs a failed registration', async () => {
    vi.stubEnv('PROD', true);
    stubServiceWorker(false);
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});

    registerServiceWorker();
    await vi.waitFor(() => expect(warn).toHaveBeenCalled());

    warn.mockRestore();
  });

  it('registers nothing during development', () => {
    vi.stubEnv('PROD', false);
    const register = stubServiceWorker(true);

    registerServiceWorker();

    expect(register).not.toHaveBeenCalled();
  });

  it('registers nothing without service workers', () => {
    vi.stubEnv('PROD', true);

    expect(() => registerServiceWorker()).not.toThrow();
  });
});
