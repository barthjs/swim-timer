import { describe, expect, it } from 'vitest';

import { breadcrumbs, type SidebarNode } from './breadcrumbs.ts';

/** link returns a sidebar link to a page. */
function link(label: string, isCurrent = false): SidebarNode {
  return {
    type: 'link',
    label,
    href: `/${label.toLowerCase()}/`,
    isCurrent,
  };
}

describe('breadcrumbs', () => {
  it('returns the groups that lead to the current page and the page', () => {
    const sidebar: SidebarNode[] = [
      link('Overview'),
      {
        type: 'group',
        label: 'Guides',
        entries: [
          link('Setup'),
          {
            type: 'group',
            label: 'Advanced',
            entries: [link('Options', true)],
          },
        ],
      },
    ];

    const trail = breadcrumbs(sidebar);

    expect(trail).toEqual([
      {
        label: 'Guides',
      },
      {
        label: 'Advanced',
      },
      {
        label: 'Options',
        href: '/options/',
      },
    ]);
  });

  it('returns no crumbs for a page the sidebar does not list', () => {
    const sidebar: SidebarNode[] = [
      link('Overview'),
      {
        type: 'group',
        label: 'Guides',
        entries: [link('Setup')],
      },
    ];

    const trail = breadcrumbs(sidebar);

    expect(trail).toEqual([]);
  });
});
