import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { useHealthStore } from '@/stores/health';

const { handlers, latest } = vi.hoisted(() => ({
  handlers: new Set<(message: { type: string; data: Record<string, string> }) => void>(),
  latest: {
    message: undefined as { type: string; data: Record<string, string> } | undefined,
  },
}));

vi.mock('@/stores/realtime', () => ({
  useRealtimeStore: () => ({
    onState: (
      type: string,
      handler: (message: { type: string; data: Record<string, string> }) => void,
    ) => {
      if (type !== 'health') return () => {};
      if (latest.message) handler(latest.message);
      handlers.add(handler);
      return () => handlers.delete(handler);
    },
  }),
}));

/** Delivers the state of the dependencies like the realtime connection. */
function report(checks: Record<string, string>): void {
  for (const handler of handlers) {
    handler({
      type: 'health',
      data: checks,
    });
  }
}

describe('useHealthStore', () => {
  const name = faker.lorem.word();

  beforeEach(() => {
    setActivePinia(createPinia());
    handlers.clear();
    latest.message = undefined;
  });

  describe('isReady', () => {
    it('reports a reachable dependency as ready', () => {
      const store = useHealthStore();
      report({
        [name]: 'ok',
      });

      const ready = store.isReady(name);

      expect(ready).toBe(true);
    });

    it('reports a dependency as ready before the server named its state', () => {
      const store = useHealthStore();

      const ready = store.isReady(name);

      expect(ready).toBe(true);
    });

    it('starts from the state the server sent before the store existed', () => {
      latest.message = {
        type: 'health',
        data: {
          [name]: 'unreachable',
        },
      };
      const store = useHealthStore();

      const ready = store.isReady(name);

      expect(ready).toBe(false);
    });

    it('reports an unreachable dependency as not ready', () => {
      const store = useHealthStore();
      report({
        [name]: 'unreachable',
      });

      const ready = store.isReady(name);

      expect(ready).toBe(false);
    });
  });
});
