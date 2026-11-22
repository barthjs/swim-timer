import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { mountVersionSelect } from '@/lib/versionSelect.ts';

const root = new URL('https://example.com/docs/v1.2.0/');

const index = {
  latest: 'v1.2.0',
  versions: ['main', 'v1.2.0', 'v1.1.0'],
};

describe('mountVersionSelect', () => {
  beforeEach(() => {
    document.body.innerHTML = `
      <span data-docs-version="v1.2.0" data-label-development="dev" data-label-latest="latest">
        <select hidden></select>
      </span>
    `;
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('fills the select with the versions and shows it', async () => {
    const fetch = vi.fn<typeof globalThis.fetch>(async () => Response.json(index));
    vi.stubGlobal('fetch', fetch);

    await mountVersionSelect(root);

    const select = selectElement();
    expect(fetch).toHaveBeenCalledWith(new URL('https://example.com/docs/versions.json'));
    expect(select.hidden).toBe(false);
    expect(select.value).toBe('v1.2.0');
    expect([...select.options].map((option) => option.text)).toEqual([
      'main (dev)',
      'v1.2.0 (latest)',
      'v1.1.0',
    ]);
  });

  it('opens the current page in the chosen version', async () => {
    const assign = vi.spyOn(window.location, 'assign').mockImplementation(() => undefined);
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => Response.json(index)),
    );
    await mountVersionSelect(root);

    const select = selectElement();
    select.value = 'v1.1.0';
    select.dispatchEvent(new Event('change'));

    await vi.waitFor(() => {
      expect(assign).toHaveBeenCalled();
    });
    expect(assign).toHaveBeenCalledWith(new URL('https://example.com/docs/v1.1.0/'));
  });

  it('opens the start page of the chosen version if it lacks the page', async () => {
    const assign = vi.spyOn(window.location, 'assign').mockImplementation(() => undefined);
    vi.stubGlobal(
      'fetch',
      vi.fn(async (_url: URL, init?: RequestInit) =>
        init?.method === 'HEAD' ? new Response(null, { status: 404 }) : Response.json(index),
      ),
    );
    await mountVersionSelect(root);

    const select = selectElement();
    select.value = 'main';
    select.dispatchEvent(new Event('change'));

    await vi.waitFor(() => {
      expect(assign).toHaveBeenCalled();
    });
    expect(assign).toHaveBeenCalledWith(new URL('https://example.com/docs/main/'));
  });

  it('keeps the version as text without a version index', async () => {
    const responses = [
      new Response(null, {
        status: 404,
      }),
      new Response('<!doctype html>'),
      Response.json({
        versions: 'main',
      }),
    ];

    for (const response of responses) {
      vi.stubGlobal(
        'fetch',
        vi.fn(async () => response),
      );

      await mountVersionSelect(root);

      expect(selectElement().hidden).toBe(true);
    }
  });

  it('does nothing on a page without a version', async () => {
    document.body.innerHTML = '';
    const fetch = vi.fn<typeof globalThis.fetch>();
    vi.stubGlobal('fetch', fetch);

    await mountVersionSelect(root);

    expect(fetch).not.toHaveBeenCalled();
  });
});

/** Returns the version select of the page header. */
function selectElement(): HTMLSelectElement {
  const select = document.querySelector('select');
  if (!select) {
    throw new Error('The page has no version select.');
  }

  return select;
}
