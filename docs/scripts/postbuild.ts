import { createHash } from 'node:crypto';
import { readdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';

/** The base the site is built with, which the postbuild replaces with relative paths, so one build works under every path. */
export const BASE_PLACEHOLDER = '/__base__/';

// Bundles live in _astro directly below the root, so the root is one level above the module that asks
const SCRIPT_ROOT = '(new URL("../",import.meta.url).pathname)';

const INLINE_SCRIPT = /<script(\s[^>]*)?>([\s\S]*?)<\/script>/g;
const SOURCE_ATTRIBUTE = /\ssrc\s*=/i;
const TYPE_ATTRIBUTE = /\stype\s*=\s*["']?([^"'\s>]+)/i;
const EXECUTABLE_TYPES = new Set(['', 'module', 'text/javascript']);

/** The content of an inline script moved into its own file below _astro. */
export type InlineScript = {
  name: string;
  content: string;
  module: boolean;
};

/**
 * Moves every executable inline script of a page into a file of its own,
 * since the content security policy of an instance only allows scripts from its own origin.
 */
export function extractInlineScripts(html: string): { html: string; scripts: InlineScript[] } {
  const scripts: InlineScript[] = [];

  const rewritten = html.replace(
    INLINE_SCRIPT,
    (tag, rawAttributes: string | undefined, content: string) => {
      const attributes = rawAttributes ?? '';
      const type = TYPE_ATTRIBUTE.exec(attributes)?.[1]?.toLowerCase() ?? '';
      if (
        SOURCE_ATTRIBUTE.test(attributes) ||
        !EXECUTABLE_TYPES.has(type) ||
        content.trim() === ''
      ) {
        return tag;
      }

      const hash = createHash('sha256').update(content).digest('hex').slice(0, 10);
      const name = `_astro/inline.${hash}.js`;
      scripts.push({
        name,
        content,
        module: type === 'module',
      });

      return `<script${attributes} src="${BASE_PLACEHOLDER}${name}"></script>`;
    },
  );

  return {
    html: rewritten,
    scripts,
  };
}

/** Returns the relative path from a file to the root of the site, ending with a slash. */
export function relativeBase(file: string, root: string): string {
  const up = path.relative(path.dirname(file), root);
  if (up === '') {
    return './';
  }

  return `${up.split(path.sep).join('/')}/`;
}

/** Replaces the placeholder in a page or stylesheet with the relative path to the root. */
export function replacePageBase(text: string, base: string): string {
  return text.replaceAll(BASE_PLACEHOLDER, base);
}

/**
 * Replaces the placeholder in a module with the root derived from the URL of the module,
 * since a module resolves relative paths against its own URL, not against the page that loads it.
 */
export function replaceModuleBase(text: string): string {
  return text.replace(/([`"'])\/__base__\/\1/g, SCRIPT_ROOT);
}

/** Sets the version of the API reference to the version of the documentation. */
export function setSpecVersion(spec: string, version: string): string {
  const document: unknown = JSON.parse(spec);
  if (!isRecord(document)) {
    throw new Error('The API reference is not a JSON object.');
  }

  document.info = {
    ...(isRecord(document.info) ? document.info : {}),
    version,
  };

  return `${JSON.stringify(document, null, 2)}\n`;
}

/** Reports whether a parsed JSON value is an object with named fields. */
function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/**
 * Makes the built site independent of its path and of inline scripts, and stamps the version
 * into the API reference. It fails if a file still holds the placeholder, since its links would lead nowhere.
 */
export async function postbuild(dist: string, version: string): Promise<void> {
  const classicScripts = new Set<string>();

  for (const file of await listFiles(dist, '.html')) {
    const { html, scripts } = extractInlineScripts(await readFile(file, 'utf8'));
    for (const script of scripts) {
      await writeFile(path.join(dist, script.name), script.content);
      if (!script.module) {
        classicScripts.add(script.name);
      }
    }
    await writeFile(file, html);
  }

  const leftovers: string[] = [];
  for (const file of await listFiles(dist, '.html', '.css', '.js')) {
    const text = await readFile(file, 'utf8');
    if (!text.includes(BASE_PLACEHOLDER)) {
      continue;
    }

    const name = path.relative(dist, file).split(path.sep).join('/');
    let replaced: string;
    if (!file.endsWith('.js')) {
      replaced = replacePageBase(text, relativeBase(file, dist));
    } else if (classicScripts.has(name)) {
      // A classic script has no URL of its own to derive the root from
      replaced = text;
    } else {
      replaced = replaceModuleBase(text);
    }

    if (replaced.includes(BASE_PLACEHOLDER)) {
      leftovers.push(name);
    }
    await writeFile(file, replaced);
  }

  if (leftovers.length > 0) {
    throw new Error(`absolute base left in ${leftovers.join(', ')}`);
  }

  const specFile = path.join(dist, 'api.json');
  await writeFile(specFile, setSpecVersion(await readFile(specFile, 'utf8'), version));
}

/** Returns every file below a directory that ends with one of the extensions. */
async function listFiles(dir: string, ...extensions: string[]): Promise<string[]> {
  const entries = await readdir(dir, {
    recursive: true,
    withFileTypes: true,
  });

  return entries
    .filter((entry) => entry.isFile() && extensions.includes(path.extname(entry.name)))
    .map((entry) => path.join(entry.parentPath, entry.name));
}

if (import.meta.main) {
  const dist = path.resolve(process.argv[2] ?? 'dist');
  await postbuild(dist, process.env.DOCS_VERSION ?? 'dev');
}
