import { fileURLToPath } from 'node:url';

import { unified } from '@astrojs/markdown-remark';
import starlight from '@astrojs/starlight';
import { defineConfig } from 'astro/config';
import starlightLinksValidator from 'starlight-links-validator';

import { remarkMdLinks } from './plugins/remark-md-links.ts';
import { remarkReleaseLinks } from './plugins/remark-release-links.ts';
import { BASE_PLACEHOLDER } from './scripts/postbuild.ts';

// A build uses the placeholder that scripts/postbuild.ts turns into relative paths, the dev server the path of an instance
const base = process.argv.includes('build') ? BASE_PLACEHOLDER.replace(/\/$/, '') : '/docs';

// The release sets the version, which the downloads and the version select of the documentation follow
const version = process.env.DOCS_VERSION ?? 'dev';

// https://astro.build/config
export default defineConfig({
  base,
  trailingSlash: 'always',
  markdown: {
    processor: unified({
      remarkPlugins: [
        [
          remarkMdLinks,
          {
            contentDir: fileURLToPath(new URL('./src/content/docs', import.meta.url)),
            base,
          },
        ],
        [
          remarkReleaseLinks,
          {
            releases: 'https://github.com/barthjs/swim-timer/releases',
            version,
          },
        ],
      ],
    }),
  },
  vite: {
    build: {
      // The main chunk of Scalar holds about 1.6 MB that cannot be split further
      chunkSizeWarningLimit: 2000,
      rolldownOptions: {
        output: {
          // zod-config.ts has to run before the chunks of Scalar, which validate their configuration while loading
          strictExecutionOrder: true,
        },
      },
    },
  },
  integrations: [
    starlight({
      title: 'Swim-Timer',
      // English is the main language, so its pages stay at the root, and pages without a translation fall back to it
      defaultLocale: 'root',
      locales: {
        root: {
          label: 'English',
          lang: 'en',
        },
        de: {
          label: 'Deutsch',
          lang: 'de',
        },
      },
      favicon: '/favicon.svg',
      customCss: ['./src/styles/theme.css'],
      components: {
        LanguageSelect: './src/components/LanguageSelect.astro',
        PageTitle: './src/components/PageTitle.astro',
        SiteTitle: './src/components/SiteTitle.astro',
      },
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/barthjs/swim-timer',
        },
      ],
      plugins: [
        // The API reference is a page of its own, which the validator cannot check. The start page links
        // relative to itself, since the attributes of its components get no base.
        starlightLinksValidator({
          exclude: [`${base}/api/`],
          errorOnRelativeLinks: false,
        }),
      ],
      sidebar: [
        {
          slug: 'index',
          label: 'Introduction',
          translations: {
            de: 'Einführung',
          },
        },
        {
          label: 'Installation',
          translations: {
            de: 'Installation',
          },
          collapsed: true,
          items: [{ autogenerate: { directory: 'installation' } }],
        },
        {
          label: 'Guides',
          translations: {
            de: 'Anleitungen',
          },
          collapsed: true,
          items: [{ autogenerate: { directory: 'guides' } }],
        },
        {
          label: 'How it works',
          translations: {
            de: 'Funktionsweise',
          },
          collapsed: true,
          items: [{ autogenerate: { directory: 'concepts' } }],
        },
        {
          label: 'Reference',
          translations: {
            de: 'Referenz',
          },
          collapsed: true,
          items: [
            { autogenerate: { directory: 'reference' } },
            {
              label: 'API reference',
              translations: {
                de: 'API-Referenz',
              },
              link: '/api/',
            },
          ],
        },
        {
          label: 'Development',
          translations: {
            de: 'Entwicklung',
          },
          collapsed: true,
          items: [{ autogenerate: { directory: 'development' } }],
        },
      ],
    }),
  ],
});
