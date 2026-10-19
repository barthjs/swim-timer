/** SidebarNode is the part of a Starlight sidebar entry that the breadcrumbs read. */
export type SidebarNode =
  | {
      type: 'link';
      label: string;
      href: string;
      isCurrent: boolean;
    }
  | {
      type: 'group';
      label: string;
      entries: SidebarNode[];
    };

/** Crumb is one step of the breadcrumbs, a group of the sidebar without a link or a page with one. */
export interface Crumb {
  label: string;
  href?: string;
}

/**
 * breadcrumbs returns the groups of the sidebar that lead to the current page, followed by the page itself,
 * or an empty list if the sidebar does not list the current page.
 */
export function breadcrumbs(entries: SidebarNode[]): Crumb[] {
  for (const entry of entries) {
    if (entry.type === 'link') {
      if (entry.isCurrent) {
        return [
          {
            label: entry.label,
            href: entry.href,
          },
        ];
      }
      continue;
    }

    const trail = breadcrumbs(entry.entries);
    if (trail.length > 0) {
      return [
        {
          label: entry.label,
        },
        ...trail,
      ];
    }
  }

  return [];
}
