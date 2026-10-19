import { describe, expect, it } from 'vitest';

import { parseVersionIndex, versionLabel, versionUrl } from './versions.ts';

const index = {
  latest: 'v1.2.0',
  versions: ['main', 'v1.3.0-rc.1', 'v1.2.0', 'v1.1.0'],
};

describe('parseVersionIndex', () => {
  it('returns the latest version and the versions', () => {
    const parsed = parseVersionIndex(index);

    expect(parsed).toEqual(index);
  });

  it('accepts an index without a stable release', () => {
    const parsed = parseVersionIndex({
      latest: null,
      versions: ['main'],
    });

    expect(parsed?.latest).toBeNull();
  });

  it('rejects anything that is not a version index', () => {
    const values = [
      null,
      'main',
      {
        latest: 1,
        versions: [],
      },
      {
        latest: 'v1.0.0',
        versions: ['main', 2],
      },
    ];

    const parsed = values.map((value) => parseVersionIndex(value));

    expect(parsed).toEqual([undefined, undefined, undefined, undefined]);
  });
});

describe('versionLabel', () => {
  it('marks the main branch and the latest release with the translated labels', () => {
    const labels = {
      development: 'Entwicklung',
      latest: 'aktuell',
    };

    const names = index.versions.map((version) => versionLabel(version, index, labels));

    expect(names).toEqual(['main (Entwicklung)', 'v1.3.0-rc.1', 'v1.2.0 (aktuell)', 'v1.1.0']);
  });
});

describe('versionUrl', () => {
  it('returns the same page in another version', () => {
    const page = new URL('https://example.com/docs/v1.2.0/guides/setup/#install');
    const root = new URL('https://example.com/docs/v1.2.0/');

    const url = versionUrl(page, root, 'main');

    expect(url.href).toBe('https://example.com/docs/main/guides/setup/#install');
  });

  it('returns the start page of the version for a page outside the root', () => {
    const page = new URL('https://example.com/other/');
    const root = new URL('https://example.com/docs/v1.2.0/');

    const url = versionUrl(page, root, 'v1.1.0');

    expect(url.href).toBe('https://example.com/docs/v1.1.0/');
  });
});
