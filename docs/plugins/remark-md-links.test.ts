import type { Link, Root } from 'mdast';
import { VFile } from 'vfile';
import { describe, expect, it } from 'vitest';

import { pageUrl, remarkMdLinks } from './remark-md-links.ts';

const options = {
  contentDir: '/site/src/content/docs',
  base: '/docs',
};

describe('remarkMdLinks', () => {
  it('rewrites the links of a page into page urls', () => {
    const link: Link = {
      type: 'link',
      url: './guides/setup.md#ports',
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
    const file = new VFile({
      path: '/site/src/content/docs/index.md',
    });

    remarkMdLinks(options)(tree, file);

    expect(link.url).toBe('/docs/guides/setup/#ports');
  });
});

describe('pageUrl', () => {
  it('returns the url of the page a relative markdown link points to', () => {
    const url = pageUrl('./usage.md', '/site/src/content/docs/index.md', options);

    expect(url).toBe('/docs/usage/');
  });

  it('keeps the anchor of the link', () => {
    const url = pageUrl('../usage.md#details', '/site/src/content/docs/guides/setup.md', options);

    expect(url).toBe('/docs/usage/#details');
  });

  it('returns the root for an index page', () => {
    const url = pageUrl('../index.md', '/site/src/content/docs/guides/setup.md', options);

    expect(url).toBe('/docs/');
  });

  it('accepts a base with a trailing slash', () => {
    const url = pageUrl('./usage.md', '/site/src/content/docs/index.md', {
      ...options,
      base: '/docs/',
    });

    expect(url).toBe('/docs/usage/');
  });

  it('adds the base to a path of the site', () => {
    const url = pageUrl('/api/', '/site/src/content/docs/guides/setup.md', options);

    expect(url).toBe('/docs/api/');
  });

  it('keeps links that do not point to a markdown file', () => {
    const links = [
      'https://example.com/readme.md',
      '//example.com/usage.md',
      '#ports',
      './image.png',
      'mailto:info@example.com',
    ];

    const urls = links.map((link) => pageUrl(link, '/site/src/content/docs/index.md', options));

    expect(urls).toEqual(links);
  });
});
