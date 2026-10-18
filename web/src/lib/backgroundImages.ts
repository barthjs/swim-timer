import { createKeyValueStore } from '@/lib/keyValueStore';

/** The background images of the preset editor, kept per preset in the browser. They never reach the server. */
export const backgroundImageStore = createKeyValueStore<Blob>('swim-timer', 'background-images');
