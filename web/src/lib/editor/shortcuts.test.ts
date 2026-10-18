import { describe, expect, it } from 'vitest';

import de from '@/locales/de.json';
import en from '@/locales/en.json';
import { EDITOR_SHORTCUTS, shortcutDefinition, shortcutKeys } from '@/lib/editor/shortcuts';

describe('EDITOR_SHORTCUTS', () => {
  const definitions = Object.values(EDITOR_SHORTCUTS).map(shortcutDefinition);

  it('gives every action its own shortcut', () => {
    const unique = new Set(definitions);

    expect(unique.size).toBe(definitions.length);
  });

  it.each(Object.keys(EDITOR_SHORTCUTS))('describes %s in the help of every locale', (shortcut) => {
    expect(en.editor.shortcuts).toHaveProperty(shortcut);
    expect(de.editor.shortcuts).toHaveProperty(shortcut);
  });

  it('leaves the digits and the arrow keys to the items of an editor', () => {
    const taken = definitions.filter((definition) => /^\d$|arrow/.test(definition));

    expect(taken).toEqual([]);
  });
});

describe('shortcutKeys', () => {
  it('returns a copy of the keys', () => {
    const keys = shortcutKeys(EDITOR_SHORTCUTS.undo);
    keys.push('x');

    const again = shortcutKeys(EDITOR_SHORTCUTS.undo);

    expect(again).toEqual(['meta', 'z']);
  });
});

describe('shortcutDefinition', () => {
  it.each([
    {
      shortcut: 'undo',
      expected: 'meta_z',
    },
    {
      shortcut: 'resetZoom',
      expected: 'shift_f',
    },
    {
      shortcut: 'fullscreen',
      expected: 'f',
    },
    {
      shortcut: 'zoomOut',
      expected: '-',
    },
  ] as const)('joins the keys of $shortcut with underscores', ({ shortcut, expected }) => {
    const definition = shortcutDefinition(EDITOR_SHORTCUTS[shortcut]);

    expect(definition).toBe(expected);
  });
});
