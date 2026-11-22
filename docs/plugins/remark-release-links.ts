import type { Link, Root } from 'mdast';
import { visit } from 'unist-util-visit';

const RELEASE_VERSION = /^v\d+\.\d+\.\d+(?:-.+)?$/;

/** Holds the URL of the releases page and the version the documentation is built for. */
export type RemarkReleaseLinksOptions = {
  releases: string;
  version: string;
};

/**
 * Points the links to the latest release at the release the documentation is built for,
 * so the documentation of a version offers the downloads of that version. The pages link to the latest release,
 * which the repository and the documentation of main keep.
 */
export function remarkReleaseLinks(options: RemarkReleaseLinksOptions) {
  return (tree: Root) => {
    visit(tree, 'link', (node: Link) => {
      node.url = releaseUrl(node.url, options.releases, options.version);
    });
  };
}

/** Returns a link to the latest release or one of its downloads for the given release version. */
export function releaseUrl(url: string, releases: string, version: string): string {
  if (!RELEASE_VERSION.test(version)) {
    return url;
  }
  if (url === `${releases}/latest`) {
    return `${releases}/tag/${version}`;
  }
  if (url.startsWith(`${releases}/latest/download/`)) {
    return `${releases}/download/${version}/${url.slice(`${releases}/latest/download/`.length)}`;
  }

  return url;
}
