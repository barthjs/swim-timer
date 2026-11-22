import '@/assets/css/main.css';

import { createApp } from 'vue';
import { createPinia } from 'pinia';
import ui from '@nuxt/ui/vue-plugin';

import App from '@/App.vue';
import { setAuthenticationRequiredHandler } from '@/api/client/client';
import { loginLocation } from '@/lib/auth/redirect';
import { applyThemeColors } from '@/lib/app/theme';
import { syncThemeColor } from '@/lib/browser/themeColor';
import { catchFatalErrors } from '@/lib/pwa/fatalError';
import { registerServiceWorker } from '@/lib/pwa/serviceWorker';
import { i18n, setLocaleFromBackend } from '@/i18n';
import { router } from '@/router';
import { useAuthStore } from '@/stores/auth';
import { useMetaStore } from '@/stores/meta';
import { configureRealtime, useRealtimeStore } from '@/stores/realtime';

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(i18n);
app.use(router);
app.use(ui);

// Only an expired or missing admin session bounces to the login page
setAuthenticationRequiredHandler(() => {
  useAuthStore().clearSession();
  void router.push(
    loginLocation(
      {
        name: 'login',
      },
      router.currentRoute.value.fullPath,
    ),
  );
});

catchFatalErrors(app, router);

// The scoreboard runs unattended during a meet, so a lost connection returns at a short fixed interval
configureRealtime({
  reconnectBaseMs: 500,
  reconnectMaxMs: 500,
});

syncThemeColor();
registerServiceWorker();
app.mount('#app');

// Every page learns a new version of the server through the realtime connection
useRealtimeStore().connect();

void useMetaStore()
  .load()
  .then((meta) => {
    setLocaleFromBackend(meta.language);
    applyThemeColors(meta.theme);
  })
  .catch((err: unknown) => {
    console.error(err);
  });
