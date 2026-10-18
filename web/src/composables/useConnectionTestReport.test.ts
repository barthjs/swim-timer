import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useConnectionTestReport } from '@/composables/useConnectionTestReport';

const add = vi.fn<(toast: object) => void>();

vi.mock('@nuxt/ui/composables', () => ({
  useToast: () => ({
    add,
  }),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('useConnectionTestReport', () => {
  beforeEach(() => {
    add.mockReset();
  });

  describe('reportTestConnection', () => {
    it('reports a working connection as a success', () => {
      const { reportTestConnection } = useConnectionTestReport();

      reportTestConnection({
        ok: true,
      });

      expect(add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'pages.config.testConnectionSuccess',
          color: 'success',
        }),
      );
    });

    it('reports a failed check with its message', () => {
      const { reportTestConnection } = useConnectionTestReport();

      reportTestConnection({
        ok: false,
        message: 'unreachable',
      });

      expect(add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'pages.config.testConnectionFailure',
          description: 'unreachable',
          color: 'error',
        }),
      );
    });
  });
});
