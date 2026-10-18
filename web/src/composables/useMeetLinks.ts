import { h, type ConcreteComponent } from 'vue';
import { useI18n } from 'vue-i18n';

import type { CompetitorResponse } from '@/api/types';
import { competitorName } from '@/lib/meet';

/**
 * MeetLinkComponents are the Nuxt UI components a link renders with, as resolveComponent returns them.
 * The calling component resolves them in its setup, where the build resolves Nuxt UI components.
 */
export type MeetLinkComponents = {
  button: ConcreteComponent | string;
  tooltip: ConcreteComponent | string;
};

/**
 * useMeetLinks renders the links of the meet overview in table cells, with an icon and a tooltip
 * that say what the click does.
 */
export function useMeetLinks(components: MeetLinkComponents) {
  const { t } = useI18n();

  /** link renders a button with a tooltip. */
  function link(label: string, icon: string, tooltip: string, onClick: () => void) {
    return h(
      components.tooltip,
      {
        text: tooltip,
        delayDuration: 300,
      },
      () =>
        h(components.button, {
          label,
          trailingIcon: icon,
          color: 'neutral',
          variant: 'link',
          class: 'text-default hover:text-highlighted p-0 text-left',
          ui: {
            trailingIcon: 'text-dimmed',
          },
          onClick,
        }),
    );
  }

  /** competitorLink filters the overview by the competitor. */
  function competitorLink(competitor: CompetitorResponse, onClick: () => void) {
    return link(
      competitorName(competitor),
      'i-lucide-filter',
      t('pages.meet.links.competitor'),
      onClick,
    );
  }

  /** clubLink filters the overview by the club. */
  function clubLink(club: string, onClick: () => void) {
    if (!club) return '';
    return link(club, 'i-lucide-filter', t('pages.meet.links.club'), onClick);
  }

  /** eventLink jumps to a heat of an event in the event list. */
  function eventLink(label: string, onClick: () => void) {
    return link(label, 'i-lucide-corner-down-right', t('pages.meet.links.event'), onClick);
  }

  return {
    competitorLink,
    clubLink,
    eventLink,
  };
}
