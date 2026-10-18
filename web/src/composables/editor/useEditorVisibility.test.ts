import { describe, expect, it } from 'vitest';

import { useEditorVisibility } from '@/composables/editor/useEditorVisibility';

describe('useEditorVisibility', () => {
  describe('isHidden', () => {
    it('reports a hidden item', () => {
      const visibility = useEditorVisibility<string>();
      visibility.toggle('a');

      const hidden = [visibility.isHidden('a'), visibility.isHidden('b')];

      expect(hidden).toEqual([true, false]);
    });
  });

  describe('toggle', () => {
    it('shows a hidden item again', () => {
      const visibility = useEditorVisibility<string>();
      visibility.toggle('a');

      visibility.toggle('a');

      expect(visibility.isHidden('a')).toBe(false);
    });
  });

  describe('showAll', () => {
    it('shows every item', () => {
      const visibility = useEditorVisibility<string>();
      visibility.toggle('a');
      visibility.toggle('b');

      visibility.showAll();

      expect(visibility.hidden.value.size).toBe(0);
    });
  });
});
