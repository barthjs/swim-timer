import { nextTick } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { KeyValueStore } from '@/lib/keyValueStore';
import { withSetup } from '@/test/withSetup';
import { useBackgroundImage } from '@/composables/editor/useBackgroundImage';

const store = {
  get: vi.fn<KeyValueStore<Blob>['get']>(),
  set: vi.fn<KeyValueStore<Blob>['set']>(),
  delete: vi.fn<KeyValueStore<Blob>['delete']>(),
};

describe('useBackgroundImage', () => {
  let unmount = () => {};

  beforeEach(() => {
    localStorage.clear();
    store.get.mockReset().mockResolvedValue(null);
    store.set.mockReset().mockResolvedValue(undefined);
    store.delete.mockReset().mockResolvedValue(undefined);
  });
  afterEach(() => unmount());

  /** mount creates the background image of a 1920 x 1080 content, or of an image cropped to fit. */
  function mount(cropped = false) {
    const mounted = withSetup(() =>
      useBackgroundImage({
        store,
        keyPrefix: 'doc',
        storageKey: 'image-test',
        content: cropped
          ? undefined
          : {
              width: 1920,
              height: 1080,
            },
      }),
    );
    unmount = mounted.unmount;
    return mounted.result;
  }

  /** loaded reports a loaded image of the given size, as the img element does. */
  function loaded(width: number, height: number): Event {
    return {
      target: {
        naturalWidth: width,
        naturalHeight: height,
      },
    } as unknown as Event;
  }

  describe('opacity', () => {
    it('starts light enough to see the items above it', () => {
      const image = mount();

      expect(image.opacity.value).toBe(0.6);
    });
  });

  describe('mismatch', () => {
    it('warns about an image with another aspect ratio', () => {
      const image = mount();

      image.onLoad(loaded(1000, 1000));

      expect(image.mismatch.value).toBe(true);
    });

    it('accepts an image with the aspect ratio of the content', () => {
      const image = mount();

      image.onLoad(loaded(1280, 720));

      expect(image.mismatch.value).toBe(false);
    });

    it('never warns about an image that is cropped to fit', () => {
      const image = mount(true);

      image.onLoad(loaded(1000, 1000));

      expect(image.mismatch.value).toBe(false);
    });
  });

  describe('show', () => {
    it('shows the image kept for a document', async () => {
      const image = mount();
      const blob = new Blob(['image']);
      store.get.mockResolvedValue(blob);

      await image.show('first');

      expect(store.get).toHaveBeenCalledWith('doc:first');
      expect(image.image.value).toBe(blob);
    });

    it('drops an image that arrives after another document opened', async () => {
      const image = mount();
      let resolve: (value: Blob | null) => void = () => {};
      store.get.mockReturnValueOnce(
        new Promise((done) => {
          resolve = done;
        }),
      );
      const first = image.show('first');
      await image.show('second');

      resolve(new Blob(['late']));
      await first;

      expect(image.image.value).toBeNull();
    });

    it('shows no image for an unsaved document', async () => {
      const image = mount();
      await image.set(new Blob(['image']), null);

      await image.show(null);

      expect(image.image.value).toBeNull();
      expect(store.get).not.toHaveBeenCalled();
    });
  });

  describe('set', () => {
    it('keeps the image of a saved document at once', async () => {
      const image = mount();
      const blob = new Blob(['image']);

      await image.set(blob, 'first');

      expect(image.image.value).toBe(blob);
      expect(store.set).toHaveBeenCalledWith('doc:first', blob);
    });

    it('deletes the image of a saved document at once', async () => {
      const image = mount();

      await image.set(null, 'first');

      expect(store.delete).toHaveBeenCalledWith('doc:first');
    });

    it('keeps the image of an unsaved document in memory', async () => {
      const image = mount();

      await image.set(new Blob(['image']), null);

      expect(store.set).not.toHaveBeenCalled();
    });
  });

  describe('persist', () => {
    it('keeps the shown image under the saved name', async () => {
      const image = mount();
      const blob = new Blob(['image']);
      await image.set(blob, null);

      await image.persist('first');

      expect(store.set).toHaveBeenCalledWith('doc:first', blob);
    });

    it('deletes the image under the earlier name of a renamed document', async () => {
      const image = mount();
      await image.set(new Blob(['image']), 'first');

      await image.persist('second', 'first');

      expect(store.delete).toHaveBeenCalledWith('doc:first');
    });

    it('keeps nothing without an image', async () => {
      const image = mount();

      await image.persist('first');

      expect(store.set).not.toHaveBeenCalled();
    });
  });

  describe('setOpacity', () => {
    it('keeps the opacity for the viewer', async () => {
      const image = mount();

      image.setOpacity(0.3);
      await nextTick();

      expect(localStorage.getItem('image-test.imageOpacity')).toBe('0.3');
    });
  });

  describe('remove', () => {
    it('hides the image and deletes it', async () => {
      const image = mount();
      await image.set(new Blob(['image']), 'first');

      await image.remove('first');

      expect(image.image.value).toBeNull();
      expect(store.delete).toHaveBeenCalledWith('doc:first');
    });
  });
});
