import { reactive } from 'vue';
import { describe, expect, it } from 'vitest';

import { useDirtyForm } from '@/composables/form/useDirtyForm';

/** setup creates a dirty form over a nested state and captures its baseline. */
function setup() {
  const state = reactive({
    name: 'draft',
    width: 2000,
    items: {
      '1': {
        x: 10,
        y: 20,
      },
    } as Record<string, { x: number; y: number }>,
  });
  const form = useDirtyForm(state);
  form.capture();
  return {
    state,
    form,
  };
}

describe('useDirtyForm', () => {
  describe('capture', () => {
    it('is clean right after capturing', () => {
      const { form } = setup();

      const dirty = form.isDirty.value;

      expect(dirty).toBe(false);
    });

    it('uses the new baseline after capturing again', () => {
      const { state, form } = setup();
      state.width = 1000;

      form.capture();

      expect(form.isDirty.value).toBe(false);
    });
  });

  describe('isFieldDirty', () => {
    it('marks changed top-level and nested fields', () => {
      const { state, form } = setup();
      const item = state.items['1'];

      state.width = 1000;
      if (item) item.x = 11;

      expect(form.isDirty.value).toBe(true);
      expect(form.isFieldDirty('width')).toBe(true);
      expect(form.isFieldDirty('name')).toBe(false);
      expect(form.isFieldDirty('items.1.x' as never)).toBe(true);
      expect(form.isFieldDirty('items.1.y' as never)).toBe(false);
    });

    it('is clean again when a field returns to its captured value', () => {
      const { state, form } = setup();
      state.name = 'other';

      state.name = 'draft';

      expect(form.isDirty.value).toBe(false);
    });
  });

  describe('reset', () => {
    it('restores the captured values without sharing objects', () => {
      const { state, form } = setup();
      state.name = 'other';
      const item = state.items['1'];
      if (item) item.y = 99;

      form.reset();

      expect(state).toEqual({
        name: 'draft',
        width: 2000,
        items: {
          '1': {
            x: 10,
            y: 20,
          },
        },
      });
      const restored = state.items['1'];
      if (restored) restored.y = 50;
      expect(form.isFieldDirty('items.1.y' as never)).toBe(true);
    });
  });

  describe('fieldProps', () => {
    it('highlights a changed field', () => {
      const { state, form } = setup();
      state.width = 1000;

      const props = form.fieldProps('width');

      expect(props).toEqual({
        color: 'warning',
        highlight: true,
      });
    });

    it('keeps an unchanged field plain', () => {
      const { form } = setup();

      const props = form.fieldProps('name');

      expect(props).toEqual({
        color: undefined,
        highlight: false,
      });
    });
  });
});
