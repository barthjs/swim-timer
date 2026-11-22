import { readFileSync } from 'node:fs';
import { mkdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { build, type Plugin } from 'vite';

import type { OfflineTexts, OfflineTheme } from '../src/lib/pwa/offline.ts';

/** Options that configure the service worker of a build. */
export type PwaOptions = {
  srcDir: string;
  locales: Record<string, string>;
  defaultLanguage: string;
  theme: OfflineTheme;
  outDir?: string;
};

const SERVICE_WORKER_ENTRY = fileURLToPath(new URL('./serviceWorker.ts', import.meta.url));
const SERVICE_WORKER_FILE = 'sw.js';

/** Adds the service worker to the root of a build. */
export function pwa(options: PwaOptions): Plugin {
  return {
    name: 'pwa',
    // A framework that renders on the server builds it as an environment of its own, which needs no service worker
    applyToEnvironment: (environment) => environment.config.consumer === 'client',
    async generateBundle() {
      const source = await bundleServiceWorker(options);

      if (options.outDir) {
        await mkdir(options.outDir, {
          recursive: true,
        });
        await writeFile(join(options.outDir, SERVICE_WORKER_FILE), source);
        return;
      }

      this.emitFile({
        type: 'asset',
        fileName: SERVICE_WORKER_FILE,
        source,
      });
    },
  };
}

/** Builds the service worker into one script with the texts of the offline page. */
async function bundleServiceWorker(options: PwaOptions): Promise<string> {
  const result = await build({
    configFile: false,
    envDir: false,
    publicDir: false,
    logLevel: 'warn',
    resolve: {
      alias: {
        '@': options.srcDir,
      },
    },
    define: {
      __PWA_OFFLINE_TEXTS__: JSON.stringify(offlineTexts(options)),
      __PWA_DEFAULT_LANGUAGE__: JSON.stringify(options.defaultLanguage),
      __PWA_OFFLINE_THEME__: JSON.stringify(options.theme),
    },
    build: {
      write: false,
      lib: {
        entry: SERVICE_WORKER_ENTRY,
        formats: ['iife'],
        name: 'serviceWorker',
      },
    },
  });

  const outputs = Array.isArray(result) ? result : [result];
  for (const output of outputs) {
    if (!('output' in output)) {
      continue;
    }
    const chunk = output.output.find((file) => file.type === 'chunk');
    if (chunk?.type === 'chunk') {
      return chunk.code;
    }
  }

  throw new Error('the build of the service worker produced no script');
}

/** Reads the texts of the offline page from the locale files and fails on a missing one. */
function offlineTexts(options: PwaOptions): Record<string, OfflineTexts> {
  if (!(options.defaultLanguage in options.locales)) {
    throw new Error(`the default language ${options.defaultLanguage} has no locale file`);
  }

  const texts: Record<string, OfflineTexts> = {};
  for (const [language, file] of Object.entries(options.locales)) {
    const locale: unknown = JSON.parse(readFileSync(file, 'utf8'));
    texts[language] = {
      title: localeText(locale, language, 'title'),
      message: localeText(locale, language, 'message'),
      retry: localeText(locale, language, 'retry'),
    };
  }

  return texts;
}

/** Returns one text of the section pwa.offline of a locale and fails if it is missing. */
function localeText(locale: unknown, language: string, key: keyof OfflineTexts): string {
  const text = property(property(property(locale, 'pwa'), 'offline'), key);
  if (typeof text !== 'string') {
    throw new Error(`the locale ${language} has no text pwa.offline.${key}`);
  }

  return text;
}

/** Returns a property of an object, or undefined for anything else. */
function property(value: unknown, key: string): unknown {
  if (typeof value !== 'object' || value === null) {
    return undefined;
  }

  return Object.entries(value).find(([name]) => name === key)?.[1];
}
