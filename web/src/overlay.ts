import '@/polyfills';
import '@/assets/css/overlay.css';

import { createApp } from 'vue';
import { createPinia } from 'pinia';

import { useRetry } from '@/composables/ui/useRetry';
import { i18n, setLocaleFromBackend } from '@/i18n';
import { outdatedCheck, reloadOnce } from '@/lib/pwa/outdated';
import OverlayApp from '@/OverlayApp.vue';
import { overlayRouter } from '@/router/overlay';
import { useMetaStore } from '@/stores/meta';
import {
  configureRealtime,
  HELLO_MESSAGE,
  type HelloMessage,
  useRealtimeStore,
} from '@/stores/realtime';

// The overlays run unattended in the streaming software, so a lost connection returns at a short fixed interval
configureRealtime({
  reconnectBaseMs: 500,
  reconnectMaxMs: 500,
});

/**
 * The overlays are a separate app from the admin pages, so browser sources load neither Nuxt UI nor Tailwind (see overlay.css).
 * The backend serves overlay.html for every path under /overlays.
 */
const app = createApp(OverlayApp);
const pinia = createPinia();

app.use(pinia);
app.use(i18n);
app.use(overlayRouter);

app.mount('#app');

// The overlays run unattended, so the language is fetched again until it arrives
const metaStore = useMetaStore(pinia);
useRetry(
  async () => metaStore.reload(),
  (meta) => setLocaleFromBackend(meta.language),
).start();

// The server names its version and settings on every connect, and the unattended overlays load new ones at once
const isOutdated = outdatedCheck(import.meta.env.VITE_APP_VERSION || 'dev');
useRealtimeStore(pinia).onState<HelloMessage>(HELLO_MESSAGE, (message) => {
  if (isOutdated(message.data)) reloadOnce(message.data);
});
