import '@/polyfills';
import '@/assets/css/overlay.css';

import { createApp, watch } from 'vue';
import { createPinia } from 'pinia';

import { useRetry } from '@/composables/useRetry';
import { i18n, setLocaleFromBackend } from '@/i18n';
import OverlayApp from '@/OverlayApp.vue';
import { overlayRouter } from '@/router/overlay';
import { useLiveStore } from '@/stores/live';
import { useMetaStore } from '@/stores/meta';

/**
 * The overlays are a separate app from the admin pages, so browser sources load neither Nuxt UI nor Tailwind
 * (see overlay.css). The backend serves overlay.html for every path under /overlays.
 */
const app = createApp(OverlayApp);
const pinia = createPinia();

app.use(pinia);
app.use(i18n);
app.use(overlayRouter);

app.mount('#app');

/**
 * The overlays run unattended, so the language is fetched again until it arrives. A reconnect of the websocket
 * fetches it anew at once, since the server may have restarted with another language.
 */
const metaStore = useMetaStore(pinia);
const liveStore = useLiveStore(pinia);

const localeLoad = useRetry(
  () => metaStore.reload(),
  (meta) => setLocaleFromBackend(meta.language),
);
localeLoad.start();

watch(
  () => liveStore.reconnects,
  () => localeLoad.start(),
);
