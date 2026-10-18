import { describe, expect, it } from 'vitest';

import { isAdditiveSelect, useEditorSelection } from '@/composables/editor/useEditorSelection';

/** setup creates a selection over three items. */
function setup() {
  return useEditorSelection(() => ['a', 'b', 'c']);
}

describe('isAdditiveSelect', () => {
  it.each([
    {
      name: 'adds with Shift',
      init: {
        shiftKey: true,
        ctrlKey: false,
        metaKey: false,
      },
      expected: true,
    },
    {
      name: 'adds with Ctrl',
      init: {
        shiftKey: false,
        ctrlKey: true,
        metaKey: false,
      },
      expected: true,
    },
    {
      name: 'adds with Cmd',
      init: {
        shiftKey: false,
        ctrlKey: false,
        metaKey: true,
      },
      expected: true,
    },
    {
      name: 'replaces without a modifier',
      init: {
        shiftKey: false,
        ctrlKey: false,
        metaKey: false,
      },
      expected: false,
    },
  ])('$name', ({ init, expected }) => {
    const additive = isAdditiveSelect(init);

    expect(additive).toBe(expected);
  });
});

describe('useEditorSelection', () => {
  describe('active', () => {
    it('is the single selected item', () => {
      const selection = setup();

      selection.set(['b']);

      expect(selection.active.value).toBe('b');
    });

    it('is null for several selected items', () => {
      const selection = setup();

      selection.set(['a', 'b']);

      expect(selection.active.value).toBeNull();
    });
  });

  describe('has', () => {
    it('reports whether an item is selected', () => {
      const selection = setup();
      selection.set(['a']);

      const result = [selection.has('a'), selection.has('b')];

      expect(result).toEqual([true, false]);
    });
  });

  describe('set', () => {
    it('replaces the selection', () => {
      const selection = setup();
      selection.set(['a']);

      selection.set(['b', 'c']);

      expect([...selection.selected.value]).toEqual(['b', 'c']);
    });
  });

  describe('clear', () => {
    it('selects no item', () => {
      const selection = setup();
      selection.set(['a', 'b']);

      selection.clear();

      expect(selection.selected.value.size).toBe(0);
    });
  });

  describe('selectOnly', () => {
    it('selects only the item', () => {
      const selection = setup();
      selection.set(['a', 'b']);

      selection.selectOnly('c');

      expect([...selection.selected.value]).toEqual(['c']);
    });

    it('ignores an unknown item', () => {
      const selection = setup();
      selection.set(['a']);

      selection.selectOnly('x');

      expect([...selection.selected.value]).toEqual(['a']);
    });
  });

  describe('toggle', () => {
    it('adds an unselected item', () => {
      const selection = setup();
      selection.set(['a']);

      selection.toggle('b');

      expect([...selection.selected.value]).toEqual(['a', 'b']);
    });

    it('removes a selected item', () => {
      const selection = setup();
      selection.set(['a', 'b']);

      selection.toggle('a');

      expect([...selection.selected.value]).toEqual(['b']);
    });
  });

  describe('selectAll', () => {
    it('selects every item', () => {
      const selection = setup();

      selection.selectAll();

      expect([...selection.selected.value]).toEqual(['a', 'b', 'c']);
    });
  });

  describe('press', () => {
    it('selects only an unselected item', () => {
      const selection = setup();
      selection.set(['a', 'b']);

      selection.press('c', false);

      expect([...selection.selected.value]).toEqual(['c']);
    });

    it('keeps the selection on a selected item', () => {
      const selection = setup();
      selection.set(['a', 'b']);

      selection.press('b', false);

      expect([...selection.selected.value]).toEqual(['a', 'b']);
    });

    it('toggles the item when additive', () => {
      const selection = setup();
      selection.set(['a', 'b']);

      selection.press('b', true);

      expect([...selection.selected.value]).toEqual(['a']);
    });
  });

  describe('selectNext', () => {
    it('selects the item after the active one', () => {
      const selection = setup();
      selection.set(['a']);

      selection.selectNext(1);

      expect(selection.active.value).toBe('b');
    });

    it('wraps around at the ends', () => {
      const selection = setup();
      selection.set(['a']);

      selection.selectNext(-1);

      expect(selection.active.value).toBe('c');
    });

    it('starts at the first item without an active one', () => {
      const selection = setup();
      selection.set(['a', 'b']);

      selection.selectNext(1);

      expect(selection.active.value).toBe('a');
    });

    it('starts at the last item backwards without an active one', () => {
      const selection = setup();

      selection.selectNext(-1);

      expect(selection.active.value).toBe('c');
    });

    it('ignores an editor without items', () => {
      const selection = useEditorSelection<string>(() => []);

      selection.selectNext(1);

      expect(selection.selected.value.size).toBe(0);
    });
  });
});
