import './zod-config.ts';

import { createApiReference } from '@scalar/api-reference';
import '@scalar/api-reference/style.css';

/**
 * The reference loads the OpenAPI document next to this page and resolves its relative server against the origin
 * of the page, since behind a reverse proxy only the browser knows the public origin. Everything that would reach
 * a service of Scalar is switched off, so the page works without internet access and sends nothing elsewhere.
 * Requests can only be tried where an instance serves the page, not on the public documentation.
 */
createApiReference('#api-reference', {
  url: '../api.json',
  baseServerURL: window.location.origin,
  hideTestRequestButton: !(await isInstance()),
  // Without a value, requests from the page would go through the proxy of Scalar
  proxyUrl: '',
  withDefaultFonts: false,
  telemetry: false,
  agent: {
    disabled: true,
  },
  mcp: {
    disabled: true,
  },
  showDeveloperTools: 'never',
  documentDownloadType: 'json',
});

/** isInstance reports whether the API of an instance answers at the origin of the page. */
async function isInstance(): Promise<boolean> {
  try {
    const response = await fetch(new URL('/api/ping', window.location.origin));
    return response.ok && (await response.text()) === 'pong';
  } catch {
    return false;
  }
}
