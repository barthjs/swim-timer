import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

const alias = {
  '@': fileURLToPath(new URL('./src', import.meta.url)),
};

// Unit tests run in node. Only tests that need browser APIs, such as the document or the location,
// carry the .dom.test.ts suffix and run in happy-dom.
export default defineConfig({
  test: {
    coverage: {
      provider: 'v8',
      include: ['plugins/**/*.ts', 'scripts/**/*.ts', 'src/**/*.ts'],
      // The API reference and the content config only run in a browser or in Astro, which the unit tests do not cover
      exclude: ['src/api/**', 'src/content.config.ts', '**/*.test.ts', '**/*.d.ts'],
      reporter: ['text-summary'],
    },
    projects: [
      {
        resolve: {
          alias,
        },
        test: {
          name: 'unit',
          environment: 'node',
          include: ['{plugins,scripts,src}/**/*.test.ts'],
          exclude: ['**/*.dom.test.ts'],
        },
      },
      {
        resolve: {
          alias,
        },
        test: {
          name: 'dom',
          environment: 'happy-dom',
          include: ['{plugins,scripts,src}/**/*.dom.test.ts'],
        },
      },
    ],
  },
});
