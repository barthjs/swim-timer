import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';

import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { checkAppLinks, missingTarget } from './check-app-links.ts';

describe('missingTarget', () => {
  let dist = '';

  beforeEach(async () => {
    dist = await mkdtemp(path.join(tmpdir(), 'app-links-'));
    await mkdir(path.join(dist, 'guides', 'setup'), {
      recursive: true,
    });
    await writeFile(
      path.join(dist, 'guides', 'setup', 'index.html'),
      '<h2 id="install">Install</h2>',
    );
  });

  afterEach(async () => {
    await rm(dist, {
      recursive: true,
      force: true,
    });
  });

  it('accepts a page and a section that exist', async () => {
    const problem = await missingTarget(dist, 'guides/setup/#install');

    expect(problem).toBeUndefined();
  });

  it('accepts a page without a section', async () => {
    const problem = await missingTarget(dist, 'guides/setup/');

    expect(problem).toBeUndefined();
  });

  it('reports a missing page', async () => {
    const problem = await missingTarget(dist, 'guides/missing/');

    expect(problem).toContain('no page');
  });

  it('reports a missing section', async () => {
    const problem = await missingTarget(dist, 'guides/setup/#missing');

    expect(problem).toContain('no section');
  });
});

describe('checkAppLinks', () => {
  let dist = '';

  beforeEach(async () => {
    dist = await mkdtemp(path.join(tmpdir(), 'app-links-'));
    await mkdir(path.join(dist, 'guides', 'setup'), {
      recursive: true,
    });
    await writeFile(
      path.join(dist, 'guides', 'setup', 'index.html'),
      '<h2 id="install">Install</h2>',
    );
  });

  afterEach(async () => {
    await rm(dist, {
      recursive: true,
      force: true,
    });
  });

  it('accepts targets that all exist', async () => {
    const result = checkAppLinks(dist, ['guides/setup/', 'guides/setup/#install']);

    await expect(result).resolves.toBeUndefined();
  });

  it('rejects the targets that lead nowhere', async () => {
    const result = checkAppLinks(dist, ['guides/setup/', 'guides/missing/']);

    await expect(result).rejects.toThrow('guides/missing/: no page');
  });
});
