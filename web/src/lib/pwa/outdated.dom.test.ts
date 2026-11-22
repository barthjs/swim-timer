import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { outdatedCheck, reloadOnce } from '@/lib/pwa/outdated';

describe('outdatedCheck', () => {
  const version = faker.system.semver();
  const meta = faker.string.hexadecimal();

  it('keeps a page on the version and the settings of the first greeting', () => {
    const isOutdated = outdatedCheck(version);

    const outdated = [
      isOutdated({
        version,
        meta,
      }),
      isOutdated({
        version,
        meta,
      }),
    ];

    expect(outdated).toEqual([false, false]);
  });

  it('reports another version', () => {
    const isOutdated = outdatedCheck(version);

    const outdated = isOutdated({
      version: `${version}-new`,
      meta,
    });

    expect(outdated).toBe(true);
  });

  it('reports settings that changed since the first greeting', () => {
    const isOutdated = outdatedCheck(version);
    isOutdated({
      version,
      meta,
    });

    const outdated = isOutdated({
      version,
      meta: `${meta}0`,
    });

    expect(outdated).toBe(true);
  });
});

describe('reloadOnce', () => {
  const hello = {
    version: faker.system.semver(),
    meta: faker.string.hexadecimal(),
  };
  const location = {
    reload: vi.fn<() => void>(),
  };

  beforeEach(() => {
    sessionStorage.clear();
    vi.stubGlobal('location', location);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    location.reload.mockReset();
  });

  it('reloads the page for a greeting', () => {
    const reloaded = reloadOnce(hello);

    expect(reloaded).toBe(true);
    expect(location.reload).toHaveBeenCalledOnce();
  });

  it('skips a greeting the page already reloaded for', () => {
    reloadOnce(hello);
    location.reload.mockReset();

    const reloaded = reloadOnce({
      ...hello,
    });

    expect(reloaded).toBe(false);
    expect(location.reload).not.toHaveBeenCalled();
  });

  it('reloads again for another greeting', () => {
    reloadOnce(hello);
    location.reload.mockReset();

    const reloaded = reloadOnce({
      ...hello,
      meta: `${hello.meta}0`,
    });

    expect(reloaded).toBe(true);
  });
});
