import { reactive } from 'vue';
import { describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import { useDirtyForm } from '@/composables/form/useDirtyForm';
import { fakeDistinct } from '@/test/fake';

/** Creates a dirty form over a nested state with faker values and captures its baseline. */
function setup() {
  const [name, otherName] = fakeDistinct(() => faker.lorem.word(), 2);
  const [width, otherWidth] = fakeDistinct(() => faker.number.int(4000), 2);
  const [x, otherX] = fakeDistinct(() => faker.number.int(1000), 2);
  const [y, otherY, thirdY] = fakeDistinct(() => faker.number.int(1000), 3);
  const initial = {
    name,
    width,
    items: {
      '1': {
        x,
        y,
      },
    } as Record<string, { x: number; y: number }>,
  };
  const state = reactive(structuredClone(initial));
  const form = useDirtyForm(state);
  form.capture();
  return {
    initial,
    other: {
      name: otherName,
      width: otherWidth,
      x: otherX,
      y: otherY,
      thirdY,
    },
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
      const { state, form, other } = setup();
      state.width = other.width;

      form.capture();

      expect(form.isDirty.value).toBe(false);
    });
  });

  describe('isFieldDirty', () => {
    it('marks changed top-level and nested fields', () => {
      const { state, form, other } = setup();
      const item = state.items['1'];

      state.width = other.width;
      if (item) item.x = other.x;

      expect(form.isDirty.value).toBe(true);
      expect(form.isFieldDirty('width')).toBe(true);
      expect(form.isFieldDirty('name')).toBe(false);
      expect(form.isFieldDirty('items.1.x')).toBe(true);
      expect(form.isFieldDirty('items.1.y')).toBe(false);
    });

    it('is clean again when a field returns to its captured value', () => {
      const { state, form, initial, other } = setup();
      state.name = other.name;

      state.name = initial.name;

      expect(form.isDirty.value).toBe(false);
    });
  });

  describe('reset', () => {
    it('restores the captured values without sharing objects', () => {
      const { state, form, initial, other } = setup();
      state.name = other.name;
      const item = state.items['1'];
      if (item) item.y = other.y;

      form.reset();

      expect(state).toEqual(initial);
      const restored = state.items['1'];
      if (restored) restored.y = other.thirdY;
      expect(form.isFieldDirty('items.1.y')).toBe(true);
    });
  });

  describe('fieldProps', () => {
    it('highlights a changed field', () => {
      const { state, form, other } = setup();
      state.width = other.width;

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
