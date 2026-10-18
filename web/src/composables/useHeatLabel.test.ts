import { describe, expect, it, vi } from 'vitest';

import { useHeatLabel } from '@/composables/useHeatLabel';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params: Record<string, number>) => `${key}:${JSON.stringify(params)}`,
  }),
}));

describe('useHeatLabel', () => {
  describe('heatLabel', () => {
    it('shows the heat count while the heat lies within it', () => {
      const { heatLabel } = useHeatLabel();

      const label = heatLabel(2, 5);

      expect(label).toBe('overlays.heat:{"number":2,"max":5}');
    });

    it('leaves the heat count out beyond it', () => {
      const { heatLabel } = useHeatLabel();

      const label = heatLabel(6, 5);

      expect(label).toBe('overlays.heatNumber:{"number":6}');
    });
  });
});
