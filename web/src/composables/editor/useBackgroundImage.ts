import { computed, ref, shallowRef, toValue, type MaybeRefOrGetter } from 'vue';
import { useLocalStorage, useObjectUrl } from '@vueuse/core';

import { aspectRatioDiffers, type Size } from '@/lib/editor/geometry';
import type { KeyValueStore } from '@/lib/keyValueStore';

// The opacity of an image before the viewer changes it, light enough to see the items above it.
const DEFAULT_OPACITY = 0.6;

export type BackgroundImage = ReturnType<typeof useBackgroundImage>;

export type BackgroundImageOptions = {
  // Where the images are kept in the browser.
  store: KeyValueStore<Blob>;
  // The kind of document an image belongs to, which prefixes its key.
  keyPrefix: string;
  // The prefix of the key the browser keeps the opacity under, such as "editor".
  storageKey: string;
  // The size of the content the image fills, to warn when it would be stretched.
  // Without it, the image is cropped to fit and never stretched.
  content?: MaybeRefOrGetter<Size>;
};

/**
 * useBackgroundImage holds an image an editor shows behind its items, such as a photo to trace.
 * The image of a saved document is kept in the browser under its name and never reaches the server.
 * An unsaved document keeps it in memory until persist stores it under the name of its first save.
 * The opacity is a matter of taste and kept per viewer.
 */
export function useBackgroundImage(options: BackgroundImageOptions) {
  const image = shallowRef<Blob | null>(null);
  const url = useObjectUrl(image);
  const opacity = useLocalStorage(`${options.storageKey}.imageOpacity`, DEFAULT_OPACITY);
  const naturalSize = ref<Size | null>(null);
  // The document whose image is shown, so an image that arrives after another document opened is dropped
  let shown: string | null = null;

  const mismatch = computed(() => {
    if (!naturalSize.value || !options.content) return false;
    return aspectRatioDiffers(naturalSize.value, toValue(options.content));
  });

  /** keyOf returns the key the image of a document is kept under. */
  function keyOf(name: string): string {
    return `${options.keyPrefix}:${name}`;
  }

  /** replace shows another image, whose size is read again once it loads. */
  function replace(next: Blob | null): void {
    image.value = next;
    naturalSize.value = null;
  }

  /** show shows the image kept for a document, or none for an unsaved one. */
  async function show(name: string | null): Promise<void> {
    shown = name;
    replace(null);
    if (name === null) return;

    const stored = await options.store.get(keyOf(name));
    if (shown === name) replace(stored);
  }

  /**
   * set shows an image, or none with null. The image of a saved document is kept or deleted at once,
   * the image of an unsaved one when persist runs.
   */
  async function set(next: Blob | null, name: string | null): Promise<void> {
    replace(next);
    if (name === null) return;

    if (next) {
      await options.store.set(keyOf(name), next);
      return;
    }
    await options.store.delete(keyOf(name));
  }

  /**
   * persist keeps the shown image under the name a document was saved with. A document saved under a new name
   * passes its earlier one, so no copy stays behind under it.
   */
  async function persist(name: string, previous: string | null = null): Promise<void> {
    shown = name;
    if (image.value) await options.store.set(keyOf(name), image.value);
    if (previous !== null && previous !== name) await options.store.delete(keyOf(previous));
  }

  /** remove hides the image of the open document, which was deleted, and deletes it from the browser. */
  async function remove(name: string): Promise<void> {
    shown = null;
    replace(null);
    await options.store.delete(keyOf(name));
  }

  /** setOpacity changes the opacity of the image for the viewer. */
  function setOpacity(value: number): void {
    opacity.value = value;
  }

  /** onLoad reads the size of the loaded image to compare its aspect ratio with the content. */
  function onLoad(event: Event): void {
    const element = event.target as HTMLImageElement;
    naturalSize.value = {
      width: element.naturalWidth,
      height: element.naturalHeight,
    };
  }

  return {
    image,
    url,
    opacity,
    mismatch,
    show,
    set,
    persist,
    remove,
    setOpacity,
    onLoad,
  };
}
