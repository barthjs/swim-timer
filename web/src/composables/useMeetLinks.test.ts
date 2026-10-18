import type { VNode } from 'vue';
import { describe, expect, it, vi } from 'vitest';

import { useMeetLinks } from '@/composables/useMeetLinks';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

const buttonStub = {
  name: 'UButton',
};
const tooltipStub = {
  name: 'UTooltip',
};

type ButtonProps = { label: string; trailingIcon: string; onClick: () => void };

/** button renders the tooltip, checks that it holds a button, and returns the props of the button. */
function button(node: VNode | string): ButtonProps {
  const tooltip = node as VNode;
  expect(tooltip.type).toBe(tooltipStub);
  const slot = (tooltip.children as { default: () => VNode }).default;
  const link = slot();
  expect(link.type).toBe(buttonStub);
  return link.props as ButtonProps;
}

const competitor = {
  id: 1,
  firstName: 'First',
  lastName: 'Last',
  birthYear: '2011',
  club: 'Club A',
  category: 'O',
};

describe('useMeetLinks', () => {
  const { competitorLink, clubLink, eventLink } = useMeetLinks({
    button: buttonStub,
    tooltip: tooltipStub,
  });

  describe('competitorLink', () => {
    it('renders a competitor as a filter link', () => {
      const onClick = vi.fn<() => void>();

      const node = competitorLink(competitor, onClick) as VNode;

      expect((node.props as { text: string }).text).toBe('pages.meet.links.competitor');
      const props = button(node);
      expect(props.label).toBe('Last, First');
      expect(props.trailingIcon).toBe('i-lucide-filter');
    });

    it('calls the callback on a click', () => {
      const onClick = vi.fn<() => void>();
      const props = button(competitorLink(competitor, onClick));

      props.onClick();

      expect(onClick).toHaveBeenCalledOnce();
    });
  });

  describe('clubLink', () => {
    it('renders a club as a filter link', () => {
      const onClick = vi.fn<() => void>();

      const node = clubLink('Club A', onClick);

      expect(button(node).label).toBe('Club A');
    });

    it('renders nothing without a club', () => {
      const onClick = vi.fn<() => void>();

      const node = clubLink('', onClick);

      expect(node).toBe('');
    });
  });

  describe('eventLink', () => {
    it('renders an event as a jump link', () => {
      const onClick = vi.fn<() => void>();

      const node = eventLink('Event A', onClick);

      const props = button(node);
      expect(props.label).toBe('Event A');
      expect(props.trailingIcon).toBe('i-lucide-corner-down-right');
    });
  });
});
