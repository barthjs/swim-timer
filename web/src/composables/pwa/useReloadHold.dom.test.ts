import { effectScope, ref } from 'vue';
import { describe, expect, it } from 'vitest';

import { isReloadHeld, useReloadHold } from '@/composables/pwa/useReloadHold';

describe('useReloadHold', () => {
  it('holds the reload while the condition is true', () => {
    const holding = ref(true);
    const scope = effectScope();
    scope.run(() => useReloadHold(() => holding.value));

    const held = [isReloadHeld()];
    holding.value = false;
    held.push(isReloadHeld());
    scope.stop();

    expect(held).toEqual([true, false]);
  });

  it('ends the hold with the scope', () => {
    const scope = effectScope();
    scope.run(() => useReloadHold(() => true));

    scope.stop();

    expect(isReloadHeld()).toBe(false);
  });
});
