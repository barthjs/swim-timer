/** MAIN_VERSION is the version of the documentation built from the main branch. */
export const MAIN_VERSION = 'main';

/** VersionIndex lists the versions of the public documentation, newest first, and the latest stable release. */
export interface VersionIndex {
  latest: string | null;
  versions: string[];
}

/** parseVersionIndex returns the version index of a parsed versions.json, or undefined for anything else. */
export function parseVersionIndex(data: unknown): VersionIndex | undefined {
  if (typeof data !== 'object' || data === null) {
    return undefined;
  }

  const { latest, versions } = data as Record<string, unknown>;
  if (latest !== null && typeof latest !== 'string') {
    return undefined;
  }
  if (!Array.isArray(versions) || !versions.every((version) => typeof version === 'string')) {
    return undefined;
  }

  return {
    latest,
    versions,
  };
}

/** VersionLabels holds the translated marks of the main branch and of the latest release. */
export interface VersionLabels {
  development: string;
  latest: string;
}

/** versionLabel returns the name of a version in the version select. */
export function versionLabel(version: string, index: VersionIndex, labels: VersionLabels): string {
  if (version === MAIN_VERSION) {
    return `${version} (${labels.development})`;
  }
  if (version === index.latest) {
    return `${version} (${labels.latest})`;
  }

  return version;
}

/**
 * versionUrl returns the URL of the same page in another version. Every version lies in a folder of its own
 * next to the others, so the path of the page below the root of its version stays the same.
 */
export function versionUrl(page: URL, root: URL, version: string): URL {
  const path = page.pathname.startsWith(root.pathname)
    ? page.pathname.slice(root.pathname.length)
    : '';

  return new URL(`../${version}/${path}${page.hash}`, root);
}
