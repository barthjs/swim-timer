import { defineConfig, globalIgnores } from 'eslint/config';
import tseslint from 'typescript-eslint';
import pluginAstro from 'eslint-plugin-astro';
import pluginVitest from '@vitest/eslint-plugin';
import pluginOxlint from 'eslint-plugin-oxlint';
import skipFormatting from 'eslint-config-prettier/flat';
import pluginComments from '@eslint-community/eslint-plugin-eslint-comments/configs';

const TEST_FILES = ['**/*.test.ts'];

/** Forbids imports from a parent folder, so moving a file keeps its imports valid. */
const PARENT_IMPORTS = {
  regex: '^\\.\\./',
  message: 'Use the alias "@/" instead of "../".',
};

export default defineConfig(
  {
    name: 'docs/files-to-lint',
    files: ['**/*.{astro,ts,mts,tsx}'],
  },

  globalIgnores(['**/dist/**', '**/.astro/**']),

  {
    name: 'docs/linter-options',
    linterOptions: {
      reportUnusedDisableDirectives: 'error',
    },
  },

  tseslint.configs.strictTypeChecked,
  tseslint.configs.stylisticTypeChecked,
  pluginAstro.configs.recommended,
  pluginComments.recommended,

  {
    name: 'docs/parser-options',
    languageOptions: {
      parserOptions: {
        projectService: true,
        extraFileExtensions: ['.astro'],
        tsconfigRootDir: import.meta.dirname,
      },
    },
  },

  {
    // astro-eslint-parser has no project service and reads the tsconfig instead
    name: 'docs/astro-parser-options',
    files: ['**/*.astro'],
    languageOptions: {
      parserOptions: {
        projectService: false,
        project: true,
      },
    },
  },

  {
    name: 'docs/rules',
    rules: {
      '@typescript-eslint/consistent-type-assertions': [
        'error',
        {
          assertionStyle: 'as',
          objectLiteralTypeAssertions: 'never',
        },
      ],
      '@typescript-eslint/consistent-type-definitions': ['error', 'type'],
      '@typescript-eslint/consistent-type-exports': 'error',
      // Vitest types the original module of a mock through `typeof import()`
      '@typescript-eslint/consistent-type-imports': [
        'error',
        {
          disallowTypeAnnotations: false,
        },
      ],
      '@typescript-eslint/method-signature-style': 'error',
      '@typescript-eslint/no-confusing-void-expression': [
        'error',
        {
          ignoreArrowShorthand: true,
        },
      ],
      '@typescript-eslint/no-import-type-side-effects': 'error',
      // noUncheckedIndexedAccess makes every index access optional, so `items[0]!` after a length check is intended
      '@typescript-eslint/no-non-null-assertion': 'off',
      '@typescript-eslint/no-unsafe-type-assertion': 'error',
      // An empty text falls back like a missing one, such as an unset name
      '@typescript-eslint/prefer-nullish-coalescing': [
        'error',
        {
          ignorePrimitives: {
            string: true,
          },
        },
      ],
      '@typescript-eslint/prefer-readonly': 'error',
      '@typescript-eslint/promise-function-async': 'error',
      '@typescript-eslint/require-array-sort-compare': 'error',
      '@typescript-eslint/restrict-template-expressions': [
        'error',
        {
          allowNumber: true,
        },
      ],
      // A missing text or flag counts like an empty text or false, as with prefer-nullish-coalescing
      '@typescript-eslint/strict-boolean-expressions': [
        'error',
        {
          allowNullableBoolean: true,
          allowNullableString: true,
        },
      ],
      // A default case handles the remaining members of a union on purpose
      '@typescript-eslint/switch-exhaustiveness-check': [
        'error',
        {
          considerDefaultExhaustiveForUnions: true,
        },
      ],
      '@eslint-community/eslint-comments/require-description': [
        'error',
        {
          ignore: ['eslint-enable'],
        },
      ],
      eqeqeq: 'error',
      'no-console': [
        'error',
        {
          allow: ['error', 'warn'],
        },
      ],
      'no-param-reassign': 'error',
      'object-shorthand': 'error',
      'prefer-template': 'error',
    },
  },

  {
    // The scripts and plugins run in node or in the Astro config before Vite resolves the alias
    name: 'docs/import-rules',
    files: ['src/**'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [PARENT_IMPORTS],
        },
      ],
    },
  },

  {
    // Type declarations merge into the interfaces of Astro and other libraries
    name: 'docs/declaration-rules',
    files: ['**/*.d.ts'],
    rules: {
      '@typescript-eslint/consistent-indexed-object-style': 'off',
      '@typescript-eslint/consistent-type-definitions': 'off',
    },
  },

  {
    ...pluginVitest.configs.recommended,
    files: TEST_FILES,
  },

  {
    name: 'docs/test-rules',
    files: TEST_FILES,
    rules: {
      '@typescript-eslint/no-empty-function': 'off',
      // Fakes stand in for library objects with only the parts a test uses
      '@typescript-eslint/consistent-type-assertions': 'off',
      '@typescript-eslint/no-unsafe-type-assertion': 'off',
      '@typescript-eslint/require-await': 'off',
    },
  },

  ...pluginOxlint.buildFromOxlintConfigFile('.oxlintrc.json'),

  skipFormatting,

  {
    // Stands after eslint-config-prettier, which turns this rule off
    name: 'docs/after-prettier',
    rules: {
      curly: ['error', 'multi-line'],
    },
  },
);
