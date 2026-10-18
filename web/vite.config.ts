import { fileURLToPath, URL } from 'node:url';

import { defineConfig, type Plugin } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import vue from '@vitejs/plugin-vue';
import ui from '@nuxt/ui/vite';

import uiOptions from './ui.config.ts';

/**
 * The oldest browser the overlays must run in. Streaming software often embeds an older Chromium,
 * for example vMix with CEF builds around Chromium 90, so the bundles are transpiled down to it.
 */
const OVERLAY_BROWSER_TARGET = 'chrome90';

// overlayEntry serves overlay.html for every page under /overlays during development,
// as the backend does in production, since the overlays are a separate app from the admin pages.
// Only page requests are rewritten, so module and asset requests keep their own paths.
function overlayEntry(): Plugin {
  return {
    name: 'overlay-entry',
    configureServer(server) {
      server.middlewares.use((req, _res, next) => {
        const isPage = req.headers.accept?.includes('text/html') ?? false;
        if (isPage && req.url?.startsWith('/overlays/')) {
          req.url = '/overlay.html';
        }
        next();
      });
    },
  };
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [overlayEntry(), tailwindcss(), vue(), ui(uiOptions)],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    target: OVERLAY_BROWSER_TARGET,
    cssTarget: OVERLAY_BROWSER_TARGET,
    rolldownOptions: {
      input: {
        main: fileURLToPath(new URL('./index.html', import.meta.url)),
        overlay: fileURLToPath(new URL('./overlay.html', import.meta.url)),
      },
    },
  },
  server: {
    host: true,
    proxy: {
      // ws forwards the upgrade of the realtime connection as well
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        ws: true,
      },
      '/readyz': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
});
