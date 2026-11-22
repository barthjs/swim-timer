import { describe, expect, it } from 'vitest';

import { shortcutDefinition } from '@/lib/ui/shortcuts';

describe('shortcutDefinition', () => {
  it.each([
    {
      keys: ['meta', 'z'],
      expected: 'meta_z',
    },
    {
      keys: ['meta', 'shift', 'z'],
      expected: 'meta_shift_z',
    },
    {
      keys: ['f'],
      expected: 'f',
    },
    {
      keys: ['-'],
      expected: '-',
    },
  ])('joins $keys with underscores', ({ keys, expected }) => {
    const definition = shortcutDefinition(keys);

    expect(definition).toBe(expected);
  });
});
