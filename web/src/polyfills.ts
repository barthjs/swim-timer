// Polyfills for the overlays, which run in the embedded Chromium 90 of streaming software such as vMix.
// The build only transpiles syntax down to that version, it does not add missing APIs.

// Object.hasOwn arrived in Chromium 93. Pinia uses it, for example when a store is patched.
// The type of Object declares it, so the check reads it as optional.
if (typeof (Object as { hasOwn?: unknown }).hasOwn !== 'function') {
  Object.defineProperty(Object, 'hasOwn', {
    value: (target: object, key: PropertyKey): boolean =>
      Object.prototype.hasOwnProperty.call(target, key),
    configurable: true,
    writable: true,
  });
}
