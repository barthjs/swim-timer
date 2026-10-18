import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

const alias = {
  '@': fileURLToPath(new URL('./src', import.meta.url)),
};

// Unit tests run in node. Only tests that need browser APIs, such as pointer events
// or component lifecycle hooks, carry the .dom.test.ts suffix and run in happy-dom.
export default defineConfig({
  test: {
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
