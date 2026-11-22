import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

const alias = {
  '@': fileURLToPath(new URL('./src', import.meta.url)),
};

// Unit tests run in node. Only tests that need browser APIs, such as pointer events
// or component lifecycle hooks, carry the .dom.test.ts suffix and run in happy-dom.
export default defineConfig({
  test: {
    coverage: {
      provider: 'v8',
      include: ['src/**/*.ts'],
      // The entry only mounts the app in a browser, which the unit tests do not cover
      exclude: ['src/main.ts', 'src/**/*.test.ts', 'src/**/*.d.ts'],
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
          include: ['src/**/*.test.ts'],
          exclude: ['src/**/*.dom.test.ts'],
        },
      },
      {
        resolve: {
          alias,
        },
        test: {
          name: 'dom',
          environment: 'happy-dom',
          include: ['src/**/*.dom.test.ts'],
        },
      },
    ],
  },
});
