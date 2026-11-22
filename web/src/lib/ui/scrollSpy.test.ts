import { describe, expect, it } from 'vitest';

import { activeSection } from '@/lib/ui/scrollSpy';

describe('activeSection', () => {
  const tops = [
    {
      section: 'a',
      top: -300,
    },
    {
      section: 'b',
      top: 80,
    },
    {
      section: 'c',
      top: 600,
    },
  ];

  it('returns the last section whose top passed the line', () => {
    expect(activeSection(tops, 200, false)).toBe('b');
  });

  it('returns the first section before any passed the line', () => {
    expect(activeSection(tops, -400, false)).toBe('a');
  });

  it('returns the last section at the end of the page', () => {
    expect(activeSection(tops, 200, true)).toBe('c');
  });

  it('returns undefined without sections', () => {
    expect(activeSection([], 200, false)).toBeUndefined();
  });
});
