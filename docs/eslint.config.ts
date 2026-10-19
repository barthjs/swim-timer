import { defineConfig, globalIgnores } from 'eslint/config';
import tseslint from 'typescript-eslint';
import pluginAstro from 'eslint-plugin-astro';
import pluginVitest from '@vitest/eslint-plugin';
import skipFormatting from 'eslint-config-prettier/flat';

export default defineConfig(
  globalIgnores(['**/dist/**', '**/.astro/**']),

  tseslint.configs.recommended,
  pluginAstro.configs.recommended,

  {
    ...pluginVitest.configs.recommended,
    files: ['**/*.test.ts'],
  },

  skipFormatting,
);
