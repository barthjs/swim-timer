import './zod-config';

import { createApiReference } from '@scalar/api-reference';
import '@scalar/api-reference/style.css';

/**
 * The reference loads the OpenAPI document next to this page and resolves its relative server against the origin
 * of the page, since behind a reverse proxy only the browser knows the public origin. Everything that would reach
 * a service of Scalar is switched off, so the page works without internet access and sends nothing elsewhere.
 */
createApiReference('#api-reference', {
  url: 'doc.json',
  baseServerURL: window.location.origin,
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
