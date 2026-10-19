import { describe, expect, it } from 'vitest';

import { DOCS_ROOT, docsUrl } from '@/lib/docs';

describe('docsUrl', () => {
  it('returns the page and section of a place in the interface', () => {
    const url = docsUrl('meet');

    expect(url).toBe('/docs/guides/meet-data/#meet-overview');
  });

  it('returns the start page without a place', () => {
    const url = docsUrl();

    expect(url).toBe(DOCS_ROOT);
  });
});
