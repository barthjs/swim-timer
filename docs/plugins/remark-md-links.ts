import path from 'node:path';

import type { Link, Root } from 'mdast';
import type { VFile } from 'vfile';
import { visit } from 'unist-util-visit';

/** Holds the content directory the page URLs are derived from and the base they start with. */
export type RemarkMdLinksOptions = {
  contentDir: string;
  base: string;
};

/**
 * Rewrites relative links to Markdown files into the URLs of their pages and adds the base to links to a path of the site,
 * so the same links work in the repository and on the built site.
 */
export function remarkMdLinks(options: RemarkMdLinksOptions) {
  return (tree: Root, file: VFile) => {
    visit(tree, 'link', (node: Link) => {
      node.url = pageUrl(node.url, file.path, options);
    });
  };
}

// Only relative links to .md or .mdx files, without a scheme and not starting at the root
const MARKDOWN_LINK = /^(?![a-z][a-z0-9+.-]*:|\/|#)([^#?]+\.mdx?)(#.*)?$/i;

/**
 * Returns the URL of the page a relative Markdown link points to, a path of the site with the base,
 * or the link itself for any other target.
 */
export function pageUrl(url: string, from: string, options: RemarkMdLinksOptions): string {
  const base = options.base.replace(/\/$/, '');
  if (url.startsWith('/') && !url.startsWith('//')) {
    return `${base}${url}`;
  }

  const match = MARKDOWN_LINK.exec(url);
  if (!match?.[1]) {
    return url;
  }

  const target = path.resolve(path.dirname(from), match[1]);
  const slug = path
    .relative(options.contentDir, target)
    .split(path.sep)
    .join('/')
    .replace(/\.mdx?$/i, '')
    .replace(/(^|\/)index$/, '')
    .toLowerCase();

  return `${base}/${slug ? `${slug}/` : ''}${match[2] ?? ''}`;
}
