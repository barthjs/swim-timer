import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';

import { afterEach, describe, expect, it } from 'vitest';

import {
  BASE_PLACEHOLDER,
  extractInlineScripts,
  postbuild,
  relativeBase,
  replaceModuleBase,
  replacePageBase,
  setSpecVersion,
} from './postbuild.ts';

describe('extractInlineScripts', () => {
  it('moves an inline script into a file of its own', () => {
    const html = '<head><script>document.title = "x";</script></head>';

    const result = extractInlineScripts(html);

    expect(result.scripts).toHaveLength(1);
    expect(result.scripts[0]?.content).toBe('document.title = "x";');
    expect(result.scripts[0]?.module).toBe(false);
    expect(result.html).toBe(
      `<head><script src="${BASE_PLACEHOLDER}${result.scripts[0]?.name}"></script></head>`,
    );
  });

  it('keeps the attributes and marks a module', () => {
    const html = '<script type="module" aria-hidden="true">import "./a.js";</script>';

    const result = extractInlineScripts(html);

    expect(result.scripts[0]?.module).toBe(true);
    expect(result.html).toContain('<script type="module" aria-hidden="true" src=');
  });

  it('names a script after its content', () => {
    const html = '<script>a();</script><script>a();</script><script>b();</script>';

    const result = extractInlineScripts(html);

    expect(result.scripts[0]?.name).toBe(result.scripts[1]?.name);
    expect(result.scripts[0]?.name).not.toBe(result.scripts[2]?.name);
  });

  it('keeps scripts with a source, data scripts, and empty scripts', () => {
    const html =
      '<script src="a.js"></script><script type="application/json">{}</script><script type="module"> </script>';

    const result = extractInlineScripts(html);

    expect(result.scripts).toEqual([]);
    expect(result.html).toBe(html);
  });
});

describe('relativeBase', () => {
  it('returns the relative path from a file to the root', () => {
    const base = relativeBase('/dist/guides/setup/index.html', '/dist');

    expect(base).toBe('../../');
  });

  it('returns the current directory for a file at the root', () => {
    const base = relativeBase('/dist/index.html', '/dist');

    expect(base).toBe('./');
  });
});

describe('replacePageBase', () => {
  it('replaces every placeholder with the relative base', () => {
    const text = `<a href="${BASE_PLACEHOLDER}usage/"></a><link href="${BASE_PLACEHOLDER}_astro/a.css">`;

    const replaced = replacePageBase(text, '../');

    expect(replaced).toBe('<a href="../usage/"></a><link href="../_astro/a.css">');
  });
});

describe('replaceModuleBase', () => {
  it('replaces the placeholder strings with the root of the module', () => {
    const text = `const base = \`${BASE_PLACEHOLDER}\`; const other = "${BASE_PLACEHOLDER}";`;

    const replaced = replaceModuleBase(text);

    expect(replaced).not.toContain(BASE_PLACEHOLDER);
    expect(replaced).toContain('const base = (new URL("../",import.meta.url).pathname);');
  });

  it('keeps a placeholder inside a longer string', () => {
    const text = `const page = "${BASE_PLACEHOLDER}usage/";`;

    const replaced = replaceModuleBase(text);

    expect(replaced).toBe(text);
  });
});

describe('setSpecVersion', () => {
  it('sets the version and keeps the other fields', () => {
    const spec = JSON.stringify({
      swagger: '2.0',
      info: {
        title: 'API',
      },
    });

    const result = JSON.parse(setSpecVersion(spec, 'v1.2.3'));

    expect(result).toEqual({
      swagger: '2.0',
      info: {
        title: 'API',
        version: 'v1.2.3',
      },
    });
  });
});

describe('postbuild', () => {
  let dist = '';

  afterEach(async () => {
    await rm(dist, {
      recursive: true,
      force: true,
    });
  });

  /** writeSite writes a minimal built site with a nested page, a module, and an OpenAPI document. */
  async function writeSite(page: string): Promise<void> {
    dist = await mkdtemp(path.join(tmpdir(), 'postbuild-'));
    await mkdir(path.join(dist, 'guide'));
    await mkdir(path.join(dist, '_astro'));
    await writeFile(path.join(dist, 'guide', 'index.html'), page);
    await writeFile(path.join(dist, '_astro', 'search.js'), `load(\`${BASE_PLACEHOLDER}\`);`);
    await writeFile(path.join(dist, 'api.json'), '{"info":{}}');
  }

  it('makes the site relative and stamps the version', async () => {
    await writeSite(`<a href="${BASE_PLACEHOLDER}"></a><script type="module">go();</script>`);

    await postbuild(dist, 'v1.2.3');

    const page = await readFile(path.join(dist, 'guide', 'index.html'), 'utf8');
    expect(page).toMatch(
      /^<a href="\.\.\/"><\/a><script type="module" src="\.\.\/_astro\/inline\.\w+\.js"><\/script>$/,
    );
    expect(await readFile(path.join(dist, '_astro', 'search.js'), 'utf8')).not.toContain(
      BASE_PLACEHOLDER,
    );
    expect(JSON.parse(await readFile(path.join(dist, 'api.json'), 'utf8')).info.version).toBe(
      'v1.2.3',
    );
  });

  it('rejects a classic inline script that needs the base', async () => {
    await writeSite(`<script>location.href = "${BASE_PLACEHOLDER}";</script>`);

    const result = postbuild(dist, 'v1.2.3');

    await expect(result).rejects.toThrow(/absolute base left in _astro\/inline\./);
  });
});
