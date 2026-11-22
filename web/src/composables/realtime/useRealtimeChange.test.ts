import { createPinia, setActivePinia } from 'pinia';
import { effectScope } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { useRealtimeChange } from '@/composables/realtime/useRealtimeChange';
import { useRealtimeStore } from '@/stores/realtime';

beforeEach(() => {
  setActivePinia(createPinia());
});

describe('useRealtimeChange', () => {
  it('registers the handler for the changes of the resource', () => {
    const realtime = useRealtimeStore();
    const onChange = vi.spyOn(realtime, 'onChange');
    const resource = faker.lorem.slug();
    const handler = vi.fn<(id?: string) => void>();

    effectScope().run(() => useRealtimeChange(resource, handler));

    expect(onChange).toHaveBeenCalledExactlyOnceWith(resource, handler);
  });

  it('removes the handler when the scope ends', () => {
    const realtime = useRealtimeStore();
    const remove = vi.fn<() => void>();
    vi.spyOn(realtime, 'onChange').mockReturnValue(remove);
    const scope = effectScope();
    scope.run(() => useRealtimeChange(faker.lorem.slug(), vi.fn<(id?: string) => void>()));

    scope.stop();

    expect(remove).toHaveBeenCalledOnce();
  });
});
