import type { Link, Root } from 'mdast';
import { describe, expect, it } from 'vitest';

import { releaseUrl, remarkReleaseLinks } from './remark-release-links.ts';

const releases = 'https://example.com/owner/project/releases';

describe('remarkReleaseLinks', () => {
  it('points the links of a page at the release of the build', () => {
    const link: Link = {
      type: 'link',
      url: `${releases}/latest/download/project_linux_amd64.tar.gz`,
      children: [],
    };
    const tree: Root = {
      type: 'root',
      children: [
        {
          type: 'paragraph',
          children: [link],
        },
      ],
    };

    remarkReleaseLinks({
      releases,
      version: 'v1.2.3',
    })(tree);

    expect(link.url).toBe(`${releases}/download/v1.2.3/project_linux_amd64.tar.gz`);
  });
});

describe('releaseUrl', () => {
  it('returns the page of the release', () => {
    const url = releaseUrl(`${releases}/latest`, releases, 'v1.2.3');

    expect(url).toBe(`${releases}/tag/v1.2.3`);
  });

  it('returns a download of the release', () => {
    const url = releaseUrl(`${releases}/latest/download/checksums.txt`, releases, 'v1.2.3-rc.1');

    expect(url).toBe(`${releases}/download/v1.2.3-rc.1/checksums.txt`);
  });

  it('keeps the latest release for a build that is no release', () => {
    const urls = ['main', 'dev'].map((version) =>
      releaseUrl(`${releases}/latest`, releases, version),
    );

    expect(urls).toEqual([`${releases}/latest`, `${releases}/latest`]);
  });

  it('keeps links that do not point to the latest release', () => {
    const links = [`${releases}`, `${releases}/tag/v1.0.0`, 'https://example.com/latest'];

    const urls = links.map((link) => releaseUrl(link, releases, 'v1.2.3'));

    expect(urls).toEqual(links);
  });
});
