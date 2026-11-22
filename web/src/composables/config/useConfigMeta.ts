import { shallowRef, type Ref } from 'vue';

import type { ConfigFile, ConfigSource } from '@/api/types/config';
import { getAtPath, setAtPath, type Paths } from '@/lib/form/path';

/** What a configuration response reports about its settings besides their values. */
export type ConfigMeta<T> = {
  defaults: T;
  sources: Record<string, ConfigSource>;
  file: ConfigFile;
};

/** Connect the explanations to the secrets and the changes of a configuration form. */
export type ConfigMetaOptions<T> = {
  // savedSecrets reports by path whether each secret is set on the server, so its keys are the secret paths
  savedSecrets: Ref<Record<string, boolean>>;
  // clearedSecrets marks the secrets a save removes
  clearedSecrets: Record<string, boolean>;
  // isFieldDirty reports whether a setting differs from the loaded configuration
  isFieldDirty: (path: Paths<T>) => boolean;
};

/** Explains the settings of a configuration form by their defaults and sources and resets them in the form. */
export function useConfigMeta<T extends Record<string, unknown>>(
  state: T,
  options: ConfigMetaOptions<T>,
) {
  const defaults = shallowRef<T>();
  const sources = shallowRef<Record<string, ConfigSource>>({});
  const file = shallowRef<ConfigFile>({
    path: '',
    exists: false,
  });

  /** Shows the defaults, sources, and config file of a configuration response. */
  function applyMeta(meta: ConfigMeta<T>): void {
    defaults.value = meta.defaults;
    sources.value = meta.sources;
    file.value = meta.file;
  }

  /** Reports whether a setting is a secret, whose value the server never sends. */
  function isSecret(path: Paths<T>): boolean {
    return path in options.savedSecrets.value;
  }

  /** Returns where a setting comes from, undefined if the server does not report it. */
  function sourceOf(path: Paths<T>): ConfigSource | undefined {
    return sources.value[path];
  }

  /** Reports whether a flag or an environment variable sets a setting, which overrides the file. */
  function isOverridden(path: Paths<T>): boolean {
    const kind = sourceOf(path)?.kind;
    return kind === 'env' || kind === 'flag';
  }

  /** Returns the default of a setting, undefined until the configuration is loaded. */
  function defaultOf(path: Paths<T>): unknown {
    return getAtPath(defaults.value, path);
  }

  /** Reports whether a setting holds its default, which for a secret means it is empty after a save. */
  function isDefault(path: Paths<T>): boolean {
    if (!defaults.value) return true;

    if (isSecret(path)) {
      const keepsSaved = !!options.savedSecrets.value[path] && !options.clearedSecrets[path];
      const value = getAtPath(state, path);
      return (value === undefined || value === '') && !keepsSaved;
    }
    return JSON.stringify(getAtPath(state, path)) === JSON.stringify(defaultOf(path));
  }

  /** Sets a setting to its default, or marks a saved secret to be removed. */
  function resetField(path: Paths<T>): void {
    if (isSecret(path)) {
      setAtPath(state, path, '');
      if (options.savedSecrets.value[path]) options.clearedSecrets[path] = true;
      return;
    }
    setAtPath(state, path, structuredClone(defaultOf(path)));
  }

  /** Returns the paths of the settings in the given dotted sections, or of every setting without any. */
  function settingPaths(sections?: string[]): Paths<T>[] {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- the walk follows the defaults, which have the shape of T
    const paths = leafPaths(defaults.value, '') as Paths<T>[];
    if (!sections) return paths;
    return paths.filter((path) =>
      sections.some((section) => path === section || path.startsWith(`${section}.`)),
    );
  }

  /**
   * Reports whether every setting of the given sections,
   * or of all settings without sections, holds its default.
   */
  function isSectionDefault(sections?: string[]): boolean {
    return settingPaths(sections).every(isDefault);
  }

  /** Sets every setting of the given sections, or every setting without sections, to its default. */
  function resetSections(sections?: string[]): void {
    for (const path of settingPaths(sections)) {
      if (!isDefault(path)) resetField(path);
    }
  }

  /**
   * Returns the sources of the changed settings that a flag or an environment variable sets,
   * since saving them to the file has no effect.
   */
  function overriddenChanges(): ConfigSource[] {
    return settingPaths()
      .filter((path) => isOverridden(path))
      .filter((path) => options.isFieldDirty(path) || !!options.clearedSecrets[path])
      .flatMap((path) => sourceOf(path) ?? []);
  }

  return {
    defaults,
    sources,
    file,
    applyMeta,
    isSecret,
    sourceOf,
    isOverridden,
    defaultOf,
    isDefault,
    resetField,
    settingPaths,
    isSectionDefault,
    resetSections,
    overriddenChanges,
  };
}

/** Returns the dot-separated paths of every field of value that is not an object, lists included. */
function leafPaths(value: unknown, prefix: string): string[] {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) {
    return prefix ? [prefix] : [];
  }
  return Object.entries(value).flatMap(([key, child]) =>
    leafPaths(child, prefix ? `${prefix}.${key}` : key),
  );
}
