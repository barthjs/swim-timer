import { globalIgnores } from 'eslint/config';
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript';
import pluginVue from 'eslint-plugin-vue';
import pluginVitest from '@vitest/eslint-plugin';
import pluginOxlint from 'eslint-plugin-oxlint';
import skipFormatting from 'eslint-config-prettier/flat';
import pluginComments from '@eslint-community/eslint-plugin-eslint-comments/configs';
import nuxtUI from '@nuxt/ui/eslint';

import uiOptions from './ui.config.ts';

const TEST_FILES = ['src/**/*.test.ts', 'src/**/test/**'];

/** The entry, router, layout, and pages of the overlays, which run without Nuxt UI and Tailwind. */
const OVERLAY_FILES = [
  'src/overlay.ts',
  'src/OverlayApp.vue',
  'src/layouts/OverlayLayout.vue',
  'src/pages/overlays/**',
  'src/router/overlay.ts',
];

/** The components the overlays show, styled with scoped CSS instead of Tailwind. */
const OVERLAY_CONTENT_FILES = [
  'src/components/overlay/ArLaneContent.vue',
  'src/components/overlay/HeatHeading.vue',
  'src/components/overlay/RecordBannerContent.vue',
];

/** Forbids imports from a parent folder, so moving a file keeps its imports valid. */
const PARENT_IMPORTS = {
  regex: '^\\.\\./',
  message: 'Use the alias "@/" instead of "../".',
};

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{vue,ts,mts,tsx}'],
  },

  globalIgnores(['**/dist/**', 'auto-imports.d.ts', 'components.d.ts']),

  {
    name: 'app/linter-options',
    linterOptions: {
      reportUnusedDisableDirectives: 'error',
    },
  },

  ...pluginVue.configs['flat/recommended'],
  vueTsConfigs.strictTypeChecked,
  vueTsConfigs.stylisticTypeChecked,
  pluginComments.recommended,

  {
    name: 'app/rules',
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
      // Short arrow functions such as `@click="() => open()"` only pass the return value on
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
      // vue-i18n binds t and te, so destructuring them from useI18n is safe
      '@typescript-eslint/unbound-method': 'off',
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
      'vue/block-lang': [
        'error',
        {
          script: {
            lang: 'ts',
          },
        },
      ],
      'vue/component-api-style': ['error', ['script-setup']],
      'vue/define-emits-declaration': 'error',
      'vue/define-macros-order': [
        'error',
        {
          order: ['defineOptions', 'defineProps', 'defineModel', 'defineEmits', 'defineSlots'],
        },
      ],
      'vue/define-props-declaration': 'error',
      'vue/html-button-has-type': 'error',
      // The default list of the rule, plus coordinate labels and units that read the same in every language
      'vue/no-bare-strings-in-template': [
        'error',
        {
          allowlist: [
            '(',
            ')',
            ',',
            '.',
            '&',
            '+',
            '-',
            '=',
            '*',
            '/',
            '#',
            '%',
            '!',
            '?',
            ':',
            '[',
            ']',
            '{',
            '}',
            '<',
            '>',
            '·',
            '•',
            '‐',
            '–',
            '—',
            '−',
            '|',
            'X:',
            'Y:',
            'm',
          ],
        },
      ],
      'vue/no-import-compiler-macros': 'error',
      'vue/no-ref-object-reactivity-loss': 'error',
      'vue/no-required-prop-with-default': 'error',
      'vue/no-static-inline-styles': 'error',
      // Nuxt UI components are used without imports, every other component is imported explicitly
      'vue/no-undef-components': [
        'error',
        {
          ignorePatterns: ['^U', '^RouterView$', '^RouterLink$'],
        },
      ],
      'vue/no-unused-emit-declarations': 'error',
      'vue/no-unused-properties': 'error',
      'vue/no-unused-refs': 'error',
      'vue/no-useless-mustaches': 'error',
      'vue/no-useless-v-bind': 'error',
      'vue/no-v-html': 'error',
      'vue/prefer-separate-static-class': 'error',
      'vue/prefer-true-attribute-shorthand': 'error',
      'vue/prefer-use-template-ref': 'error',
      'vue/require-explicit-emits': 'error',
      'vue/require-macro-variable-name': 'error',
      'vue/require-typed-ref': 'error',
    },
  },

  {
    // The files outside src run before Vite resolves the alias
    name: 'app/import-rules',
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
    // Type declarations merge into the interfaces of Vite and other libraries
    name: 'app/declaration-rules',
    files: ['**/*.d.ts'],
    rules: {
      '@typescript-eslint/consistent-indexed-object-style': 'off',
      '@typescript-eslint/consistent-type-definitions': 'off',
    },
  },

  {
    name: 'app/overlay-rules',
    files: OVERLAY_FILES,
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@/router',
              message: 'Overlays use the overlay router.',
            },
            {
              name: '@/assets/css/main.css',
              message: 'Overlays use overlay.css without Tailwind.',
            },
          ],
          patterns: [
            PARENT_IMPORTS,
            {
              regex: '^@nuxt/ui',
              message: 'Overlays run without Nuxt UI.',
            },
          ],
        },
      ],
      'vue/no-undef-components': [
        'error',
        {
          ignorePatterns: ['^RouterView$', '^RouterLink$'],
        },
      ],
    },
  },

  {
    ...pluginVitest.configs.recommended,
    files: TEST_FILES,
  },

  {
    name: 'app/test-rules',
    files: TEST_FILES,
    rules: {
      '@typescript-eslint/no-empty-function': 'off',
      // Fakes stand in for browser and library objects with only the parts a test uses
      '@typescript-eslint/consistent-type-assertions': 'off',
      '@typescript-eslint/no-unsafe-class-instance-assignment': 'off',
      '@typescript-eslint/no-unsafe-type-assertion': 'off',
      // Fakes such as `vi.fn(async () => true)` imitate an asynchronous interface without awaiting anything
      '@typescript-eslint/require-await': 'off',
      // Tests read the value of a ref to check it
      'vue/no-ref-object-reactivity-loss': 'off',
      // Tests define small host components next to each other
      'vue/one-component-per-file': 'off',
    },
  },

  // The classes of templates, `:ui` objects, and the theme use the semantic colors of the theme,
  // and a list never holds two classes that set the same property.
  ...nuxtUI({
    entryPoint: 'src/assets/css/main.css',
    colors: uiOptions.ui.colors,
    // The classes of scoped styles and of table.css, which the stylesheet does not generate
    ignore: [
      '^editor-grid$',
      '^handle$',
      '^label$',
      '^lane$',
      '^loupe-outline$',
      '^outline-only$',
      '^table-toolbar-controls$',
    ],
    rules: {
      'no-palette-colors': 'error',
      'no-conflicting-classes': 'error',
      'no-unknown-classes': 'error',
    },
  }),

  {
    // The overlays run without Tailwind and style their classes with scoped CSS
    name: 'app/overlay-classes',
    files: [...OVERLAY_FILES, ...OVERLAY_CONTENT_FILES],
    rules: {
      'better-tailwindcss/no-unknown-classes': 'off',
    },
  },

  ...pluginOxlint.buildFromOxlintConfigFile('.oxlintrc.json'),

  skipFormatting,

  {
    // Stands after eslint-config-prettier, which turns these rules off. Plain HTML elements get a closing tag,
    // since `<div />` is only valid for the Vue compiler and editors report it as invalid HTML
    name: 'app/after-prettier',
    rules: {
      curly: ['error', 'multi-line'],
      'vue/html-self-closing': [
        'error',
        {
          html: {
            void: 'always',
            normal: 'never',
            component: 'always',
          },
          svg: 'always',
          math: 'always',
        },
      ],
    },
  },
);
