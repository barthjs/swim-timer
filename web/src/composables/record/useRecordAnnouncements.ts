import { onScopeDispose, ref, toValue, watch, type MaybeRefOrGetter } from 'vue';

import type { RecordBanner } from '@/lib/record/recordBanner';

/** How long a record stays on screen before the next one or the end. */
export const RECORD_DISPLAY_MS = 10_000;

/** Shows every record of the list of the heat once, one after another, each for RECORD_DISPLAY_MS. */
export function useRecordAnnouncements(
  banners: MaybeRefOrGetter<RecordBanner[]>,
  heat: MaybeRefOrGetter<string>,
) {
  // The list follows the live heat while the records of the previous one still wait, so they stay in the queue.
  // The keys of the records name their heat, so a heat that sends its records again only adds the new ones,
  // and a waiting record its heat no longer lists is dropped.
  const current = ref<RecordBanner | null>(null);
  const queue: RecordBanner[] = [];
  const shown = new Set<string>();
  let timeout: ReturnType<typeof setTimeout> | null = null;

  watch(
    () => toValue(banners),
    (list) => {
      const keys = new Set(list.map((banner) => banner.key));
      const listed = toValue(heat);
      const kept = queue.filter((banner) => banner.heat !== listed || keys.has(banner.key));
      queue.splice(0, queue.length, ...kept);
      for (const banner of list) {
        if (shown.has(banner.key)) continue;
        shown.add(banner.key);
        queue.push(banner);
      }
      showNext();
    },
    {
      immediate: true,
    },
  );

  /** Shows the next record in the queue, unless one is shown. */
  function showNext(): void {
    if (current.value) return;
    const next = queue.shift();
    if (!next) return;

    current.value = next;
    timeout = setTimeout(() => {
      timeout = null;
      current.value = null;
      showNext();
    }, RECORD_DISPLAY_MS);
  }

  onScopeDispose(() => {
    if (timeout) clearTimeout(timeout);
  });

  return {
    current,
  };
}
