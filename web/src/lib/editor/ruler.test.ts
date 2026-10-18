import { describe, expect, it } from 'vitest';

import { rulerTicks } from '@/lib/editor/ruler';

describe('rulerTicks', () => {
  it('labels a readable step at the original size and marks the steps between', () => {
    const ticks = rulerTicks(300, 0, 1);
    const labels = ticks.filter((tick) => tick.label !== null);

    expect(labels.map((tick) => tick.label)).toEqual(['0', '100', '200', '300']);
    expect(ticks[1]).toEqual({
      position: 10,
      size: 4,
      label: null,
    });
    expect(ticks[5]).toEqual({
      position: 50,
      size: 8,
      label: null,
    });
  });

  it('places the ticks with the offset and the scale of the content', () => {
    const labels = rulerTicks(400, 50, 0.5).filter((tick) => tick.label !== null);

    expect(labels.map((tick) => [tick.position, tick.label])).toEqual([
      [50, '0'],
      [150, '200'],
      [250, '400'],
      [350, '600'],
    ]);
  });

  it('labels smaller steps when zoomed in', () => {
    const labels = rulerTicks(200, 0, 8).filter((tick) => tick.label !== null);

    expect(labels.map((tick) => tick.label)).toEqual(['0', '10', '20']);
  });

  it('starts at the first value visible after panning', () => {
    const ticks = rulerTicks(100, -1000, 1);

    expect(ticks[0]?.position).toBe(0);
    expect(ticks[0]?.label).toBe('1000');
  });

  it.each([
    {
      name: 'a length',
      length: 0,
      scale: 1,
    },
    {
      name: 'a scale',
      length: 100,
      scale: 0,
    },
  ])('returns no ticks without $name', ({ length, scale }) => {
    const ticks = rulerTicks(length, 0, scale);

    expect(ticks).toEqual([]);
  });
});
