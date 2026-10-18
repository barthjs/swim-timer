import { useI18n } from 'vue-i18n';

import { showsHeatCount } from '@/lib/heat';

/** useHeatLabel names heats with their number and, while it fits, the heat count of the event. */
export function useHeatLabel() {
  const { t } = useI18n();

  /** heatLabel names a heat with the heat count, or with its number only beyond the heat count. */
  function heatLabel(heat: number, heatCount: number): string {
    if (showsHeatCount(heat, heatCount)) {
      return t('overlays.heat', {
        number: heat,
        max: heatCount,
      });
    }
    return t('overlays.heatNumber', {
      number: heat,
    });
  }

  return {
    heatLabel,
  };
}
