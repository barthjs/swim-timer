import { describe, expect, it } from 'vitest';

import { tableRowClass } from '@/lib/table/rows';

describe('tableRowClass', () => {
  it('shades every row on hover', () => {
    const classes = tableRowClass({
      index: 0,
    });

    expect(classes).toBe('group/row hover:bg-table-hover');
  });

  it('stripes every second row', () => {
    const classes = tableRowClass({
      index: 1,
    });

    expect(classes).toBe('group/row hover:bg-table-hover bg-table-stripe');
  });
});
