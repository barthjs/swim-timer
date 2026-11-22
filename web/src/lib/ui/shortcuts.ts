/** Joins the keys of a shortcut into the name defineShortcuts of Nuxt UI expects. */
export function shortcutDefinition(keys: readonly string[]): string {
  return keys.join('_');
}
