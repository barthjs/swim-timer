import { rmSync } from 'node:fs';

import { defineConfig, type Plugin } from 'vite';

// cleanAssets removes the bundles of the previous build, so dist keeps its tracked README for go:embed
// while the hashed file names of an old build do not pile up next to the new ones.
function cleanAssets(): Plugin {
  return {
    name: 'clean-assets',
    apply: 'build',
    buildStart() {
      rmSync('dist/assets', {
        recursive: true,
        force: true,
      });
    },
  };
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [cleanAssets()],
  // The page is served below the API, so its assets are referenced relative to it
  base: './',
  build: {
    target: 'es2024',
    // The main chunk of Scalar holds about 1.6 MB that cannot be split further
    chunkSizeWarningLimit: 2000,
    emptyOutDir: false,
    rolldownOptions: {
      output: {
        // zod-config.ts has to run before the chunks of Scalar, which validate their configuration while loading
        strictExecutionOrder: true,
      },
    },
  },
});
