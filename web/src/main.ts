import '@/assets/css/main.css';

import { createApp } from 'vue';
import { createPinia } from 'pinia';
import ui from '@nuxt/ui/vue-plugin';

import App from '@/App.vue';
import { setAuthenticationRequiredHandler } from '@/api/client';
import { loginLocation } from '@/lib/auth/redirect';
import { applyThemeColors } from '@/lib/theme';
import { i18n, setLocaleFromBackend } from '@/i18n';
import { router } from '@/router';
import { useAuthStore } from '@/stores/auth';
import { useMetaStore } from '@/stores/meta';
import { configureRealtime } from '@/stores/realtime';

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

// Overlays run unattended during a meet, so a lost connection returns at a short fixed interval
configureRealtime({
  reconnectBaseMs: 500,
  reconnectMaxMs: 500,
});

app.mount('#app');

void useMetaStore()
  .load()
  .then((meta) => {
    setLocaleFromBackend(meta.language);
    applyThemeColors(meta.theme);
  })
  .catch((err) => {
    console.error(err);
  });
