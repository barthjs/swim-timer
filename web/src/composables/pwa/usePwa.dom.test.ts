import type { useToast } from '@nuxt/ui/composables';
import { effectScope, nextTick, ref, type EffectScope } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { usePwa } from '@/composables/pwa/usePwa';

type Greeting = {
  type: string;
  data: {
    version: string;
    meta: string;
  };
};

const { guards, helloHandlers, reloadOnce, toast } = vi.hoisted(() => ({
  guards: [] as ((to: { fullPath: string }) => boolean)[],
  helloHandlers: new Set<(message: Greeting) => void>(),
  reloadOnce: vi.fn<() => boolean>(),
  toast: {
    add: vi.fn<ReturnType<typeof useToast>['add']>(),
  },
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock('vue-router', () => ({
  useRouter: () => ({
    beforeEach: (guard: (to: { fullPath: string }) => boolean) => {
      guards.push(guard);
      return () => guards.splice(guards.indexOf(guard), 1);
    },
  }),
}));

vi.mock('@nuxt/ui/composables', () => ({
  useToast: () => toast,
}));

vi.mock('@/composables/pwa/useReloadHold', () => ({
  isReloadHeld: () => heldRef.value,
}));

vi.mock('@/lib/pwa/outdated', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/pwa/outdated')>()),
  reloadOnce,
}));

vi.mock('@/stores/realtime', () => ({
  HELLO_MESSAGE: 'hello',
  useRealtimeStore: () => ({
    onState: (_type: string, handler: (message: Greeting) => void) => {
      helloHandlers.add(handler);
      return () => helloHandlers.delete(handler);
    },
  }),
}));

const heldRef = ref(false);

/** Delivers the greeting of a server with the version and the settings like the realtime connection. */
function greet(version: string, meta = 'settings'): void {
  for (const handler of helloHandlers) {
    handler({
      type: 'hello',
      data: {
        version,
        meta,
      },
    });
  }
}

/** Sets the visibility of the document and reports the change like the browser. */
function setVisibility(state: DocumentVisibilityState): void {
  Object.defineProperty(document, 'visibilityState', {
    value: state,
    configurable: true,
  });
  document.dispatchEvent(new Event('visibilitychange'));
}

/** Replaces the reload and the navigation of the page with spies. */
function stubLocation() {
  const location = {
    reload: vi.fn<() => void>(),
    assign: vi.fn<(url: string) => void>(),
  };
  vi.stubGlobal('location', location);
  return location;
}

/** Runs the navigation guards for a path and returns whether the app may navigate. */
function navigate(path: string): boolean {
  return guards.every((guard) =>
    guard({
      fullPath: path,
    }),
  );
}

describe('usePwa', () => {
  let scope: EffectScope;
  const version = faker.system.semver();
  const newer = `${version}-new`;

  /** Starts the composable on a page that runs unattended or not. */
  function run(unattended = false): void {
    scope.run(() => usePwa(version, () => unattended));
  }

  beforeEach(() => {
    scope = effectScope();
    heldRef.value = false;
    reloadOnce.mockReturnValue(true);
    setVisibility('visible');
  });

  afterEach(() => {
    scope.stop();
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('offers the reload on a page somebody reads', () => {
    run();

    greet(newer);

    expect(reloadOnce).not.toHaveBeenCalled();
    expect(toast.add).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({
        title: 'pwa.update.title',
        actions: [
          expect.objectContaining({
            label: 'pwa.reload',
          }),
        ],
      }),
    );
  });

  it('offers the reload for changed settings', () => {
    run();
    greet(version);

    greet(version, 'changed');

    expect(toast.add).toHaveBeenCalledOnce();
  });

  it('offers the reload only once', () => {
    run();

    greet(newer);
    greet(`${newer}-again`);

    expect(toast.add).toHaveBeenCalledOnce();
  });

  it('reloads a hidden page at once', () => {
    setVisibility('hidden');
    run();

    greet(newer);

    expect(reloadOnce).toHaveBeenCalledOnce();
    expect(toast.add).not.toHaveBeenCalled();
  });

  it('reloads an unattended page at once without an offer', () => {
    run(true);

    greet(newer);

    expect(reloadOnce).toHaveBeenCalledOnce();
    expect(toast.add).not.toHaveBeenCalled();
  });

  it('reloads a page once it becomes hidden', async () => {
    run();
    greet(newer);

    setVisibility('hidden');
    await nextTick();

    expect(reloadOnce).toHaveBeenCalledOnce();
  });

  it('keeps a page whose reload would lose something until it loses nothing', async () => {
    heldRef.value = true;
    setVisibility('hidden');
    run();
    greet(newer);
    const before = reloadOnce.mock.calls.length;

    heldRef.value = false;
    await nextTick();

    expect(before).toBe(0);
    expect(reloadOnce).toHaveBeenCalledOnce();
  });

  it('offers the reload when the page already reloaded for the same greeting', () => {
    reloadOnce.mockReturnValue(false);
    setVisibility('hidden');
    run();

    greet(newer);

    expect(toast.add).toHaveBeenCalledOnce();
  });

  it('keeps the page on the same version and settings', async () => {
    run();
    greet(version);

    setVisibility('hidden');
    await nextTick();

    expect(reloadOnce).not.toHaveBeenCalled();
    expect(toast.add).not.toHaveBeenCalled();
  });

  it('loads the target of a navigation on a new version as a new page', () => {
    const location = stubLocation();
    const path = `/${faker.lorem.slug()}`;
    run();
    greet(newer);

    const allowed = navigate(path);

    expect(allowed).toBe(false);
    expect(location.assign).toHaveBeenCalledExactlyOnceWith(path);
  });

  it('navigates in the app while a reload would lose something', () => {
    const location = stubLocation();
    heldRef.value = true;
    run();
    greet(newer);

    const allowed = navigate(`/${faker.lorem.slug()}`);

    expect(allowed).toBe(true);
    expect(location.assign).not.toHaveBeenCalled();
  });

  it('navigates in the app on the same version', () => {
    const location = stubLocation();
    run();
    greet(version);

    const allowed = navigate(`/${faker.lorem.slug()}`);

    expect(allowed).toBe(true);
    expect(location.assign).not.toHaveBeenCalled();
  });
});
