import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';

import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { compareVersions, redirectPage, versionIndex, writeSiteIndex } from './pages.ts';

describe('compareVersions', () => {
  it('orders releases newest first', () => {
    const versions = ['v1.2.0', 'v1.10.0', 'v2.0.0', 'v1.2.10', 'v1.2.9'];

    const sorted = versions.sort(compareVersions);

    expect(sorted).toEqual(['v2.0.0', 'v1.10.0', 'v1.2.10', 'v1.2.9', 'v1.2.0']);
  });

  it('puts a prerelease before the release of the same number', () => {
    const versions = ['v1.2.0', 'v1.3.0', 'v1.3.0-rc.2', 'v1.3.0-rc.10'];

    const sorted = versions.sort(compareVersions);

    expect(sorted).toEqual(['v1.3.0', 'v1.3.0-rc.10', 'v1.3.0-rc.2', 'v1.2.0']);
  });
});

describe('versionIndex', () => {
  it('lists the main branch first and marks the newest stable release as latest', () => {
    const index = versionIndex(['v1.0.0', 'main', 'v1.1.0-rc.1', 'v1.0.1']);

    expect(index).toEqual({
      latest: 'v1.0.1',
      versions: ['main', 'v1.1.0-rc.1', 'v1.0.1', 'v1.0.0'],
    });
  });

  it('has no latest version before the first stable release', () => {
    const index = versionIndex(['main', 'v1.0.0-rc.1']);

    expect(index.latest).toBeNull();
  });

  it('leaves out folders that are no version', () => {
    const index = versionIndex(['main', 'assets', 'v1.0', 'v1.0.0']);

    expect(index.versions).toEqual(['main', 'v1.0.0']);
  });
});

describe('redirectPage', () => {
  it('sends the browser on with and without JavaScript', () => {
    const page = redirectPage('Docs', './v1.0.0/');

    expect(page).toContain('<meta http-equiv="refresh" content="0; url=./v1.0.0/" />');
    expect(page).toContain('<a href="./v1.0.0/">Docs</a>');
  });
});

describe('writeSiteIndex', () => {
  let site = '';

  beforeEach(async () => {
    site = await mkdtemp(path.join(tmpdir(), 'pages-'));
  });

  afterEach(async () => {
    await rm(site, {
      recursive: true,
      force: true,
    });
  });

  /** addVersions creates a folder per version, as the workflow unpacks the documentation of each release. */
  async function addVersions(...versions: string[]): Promise<void> {
    for (const version of versions) {
      await mkdir(path.join(site, version));
    }
    await writeFile(path.join(site, 'stray.txt'), '');
  }

  it('writes the index and leads to the latest release', async () => {
    await addVersions('main', 'v1.0.0', 'v1.1.0');

    await writeSiteIndex(site, '/docs');

    const versions = JSON.parse(await readFile(path.join(site, 'versions.json'), 'utf8'));
    expect(versions).toEqual({
      latest: 'v1.1.0',
      versions: ['main', 'v1.1.0', 'v1.0.0'],
    });
    expect(await readFile(path.join(site, 'index.html'), 'utf8')).toContain('url=./v1.1.0/');
    expect(await readFile(path.join(site, '404.html'), 'utf8')).toContain('url=/docs/v1.1.0/');
  });

  it('leads to the main branch before the first release', async () => {
    await addVersions('main');

    await writeSiteIndex(site, '/');

    expect(await readFile(path.join(site, 'index.html'), 'utf8')).toContain('url=./main/');
    expect(await readFile(path.join(site, '404.html'), 'utf8')).toContain('url=/main/');
  });

  it('rejects a site without any version', async () => {
    const result = writeSiteIndex(site, '/');

    await expect(result).rejects.toThrow('no version');
  });
});
