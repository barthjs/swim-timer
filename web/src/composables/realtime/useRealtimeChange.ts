import { onScopeDispose } from 'vue';

import { useRealtimeStore } from '@/stores/realtime';

/** Runs the handler whenever the resource changes on the server while the calling scope lives, see onChange. */
export function useRealtimeChange(resource: string, handler: (id?: string) => void): void {
  onScopeDispose(useRealtimeStore().onChange(resource, handler));
}
