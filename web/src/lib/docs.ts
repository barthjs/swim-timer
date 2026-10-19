/** DOCS_ROOT is the path under which the instance serves the documentation of its own version. */
export const DOCS_ROOT = '/docs/';

/**
 * DOCS_PAGES maps the places of the interface to the page and section of the documentation that explains them.
 * The documentation build checks that every target exists, so a renamed page or heading fails there.
 */
export const DOCS_PAGES = {
  dashboard: 'guides/overlays/',
  meet: 'guides/meet-data/#meet-overview',
  meetData: 'guides/meet-data/',
  editor: 'guides/ar-presets/',
  configApp: 'guides/configuration/',
  configTiming: 'guides/timing-device/',
  configServer: 'guides/access/',
  configWebhook: 'guides/webhooks/',
  configStorage: 'reference/settings/#storage',
  configTheme: 'guides/configuration/#colors',
} as const;

/** DocsPage names a place of the interface with a page of the documentation. */
export type DocsPage = keyof typeof DOCS_PAGES;

/** docsUrl returns the URL of the documentation page of a place in the interface, or the start page without one. */
export function docsUrl(page?: DocsPage): string {
  if (!page) {
    return DOCS_ROOT;
  }

  return `${DOCS_ROOT}${DOCS_PAGES[page]}`;
}
