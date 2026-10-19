import { readdir, writeFile } from 'node:fs/promises';
import path from 'node:path';

import { MAIN_VERSION, type VersionIndex } from '../src/lib/versions.ts';

const RELEASE_VERSION = /^v(\d+)\.(\d+)\.(\d+)(?:-(.+))?$/;

/**
 * compareVersions orders release versions newest first. A prerelease comes before the release of the same number,
 * and prereleases of one number are ordered by their name.
 */
export function compareVersions(a: string, b: string): number {
  const left = RELEASE_VERSION.exec(a);
  const right = RELEASE_VERSION.exec(b);
  if (!left || !right) {
    return a.localeCompare(b);
  }

  for (let part = 1; part <= 3; part++) {
    const difference = Number(right[part]) - Number(left[part]);
    if (difference !== 0) {
      return difference;
    }
  }

  const leftPre = left[4];
  const rightPre = right[4];
  if (leftPre === rightPre) {
    return 0;
  }
  if (leftPre === undefined) {
    return -1;
  }
  if (rightPre === undefined) {
    return 1;
  }

  return rightPre.localeCompare(leftPre, 'en', {
    numeric: true,
  });
}

/**
 * versionIndex lists the main branch first and the releases newest first. The latest version is the newest release
 * without a prerelease suffix. Folders that are no version are left out.
 */
export function versionIndex(folders: string[]): VersionIndex {
  const releases = folders.filter((folder) => RELEASE_VERSION.test(folder)).sort(compareVersions);
  const latest = releases.find((release) => !release.includes('-')) ?? null;
  const versions = folders.includes(MAIN_VERSION) ? [MAIN_VERSION, ...releases] : releases;

  return {
    latest,
    versions,
  };
}

/** redirectPage returns a page that sends the browser on to a URL, also without JavaScript. */
export function redirectPage(title: string, url: string): string {
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>${title}</title>
    <meta http-equiv="refresh" content="0; url=${url}" />
    <link rel="canonical" href="${url}" />
  </head>
  <body>
    <a href="${url}">${title}</a>
  </body>
</html>
`;
}

/**
 * writeSiteIndex writes versions.json, the start page, and the 404 page at the root of the public documentation.
 * The start page leads to the latest release, or to the main branch before the first release. The 404 page
 * is served for any missing path, so it links to the start page with the absolute base path of the site.
 */
export async function writeSiteIndex(site: string, basePath: string): Promise<VersionIndex> {
  const entries = await readdir(site, {
    withFileTypes: true,
  });
  const index = versionIndex(
    entries.filter((entry) => entry.isDirectory()).map((entry) => entry.name),
  );
  const start = index.latest ?? index.versions[0];
  if (start === undefined) {
    throw new Error(`no version of the documentation in ${site}`);
  }

  const base = basePath.endsWith('/') ? basePath : `${basePath}/`;
  await writeFile(path.join(site, 'versions.json'), `${JSON.stringify(index, null, 2)}\n`);
  await writeFile(path.join(site, 'index.html'), redirectPage('Documentation', `./${start}/`));
  await writeFile(path.join(site, '404.html'), redirectPage('Page not found', `${base}${start}/`));

  return index;
}

if (import.meta.main) {
  const [site = 'site', basePath = '/'] = process.argv.slice(2);
  await writeSiteIndex(path.resolve(site), basePath);
}
