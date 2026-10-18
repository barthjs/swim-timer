import { shallowRef, type Ref } from 'vue';

import type { ConfigFile, ConfigSource } from '@/api/types';
import { getAtPath, setAtPath, type Paths } from '@/lib/form/path';

/** ConfigMeta is what a configuration response reports about its settings besides their values. */
export type ConfigMeta<T> = {
  defaults: T;
  sources: Record<string, ConfigSource>;
  file: ConfigFile;
};

/** ConfigMetaOptions connect the explanations to the secrets and the changes of a configuration form. */
export type ConfigMetaOptions<T> = {
  // savedSecrets reports by path whether each secret is set on the server, so its keys are the secret paths.
  savedSecrets: Ref<Record<string, boolean>>;
  // clearedSecrets marks the secrets a save removes.
  clearedSecrets: Record<string, boolean>;
  // isFieldDirty reports whether a setting differs from the loaded configuration.
  isFieldDirty: (path: Paths<T>) => boolean;
};

/**
 * useConfigMeta explains the settings of a configuration form by their defaults and sources and resets
 * them to their defaults. A reset only changes the form, the configuration changes once it is saved.
 * A secret has no default, so resetting a saved secret marks it to be removed.
 */
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

  /** applyMeta shows the defaults, sources, and config file of a configuration response. */
  function applyMeta(meta: ConfigMeta<T>): void {
    defaults.value = meta.defaults;
    sources.value = meta.sources;
    file.value = meta.file;
  }

  /** isSecret reports whether a setting is a secret, whose value the server never sends. */
  function isSecret(path: Paths<T>): boolean {
    return path in options.savedSecrets.value;
  }

  /** sourceOf returns where a setting comes from, undefined if the server does not report it. */
  function sourceOf(path: Paths<T>): ConfigSource | undefined {
    return sources.value[path];
  }

  /** isOverridden reports whether a flag or an environment variable sets a setting, which overrides the file. */
  function isOverridden(path: Paths<T>): boolean {
    const kind = sourceOf(path)?.kind;
    return kind === 'env' || kind === 'flag';
  }

  /** defaultOf returns the default of a setting, undefined until the configuration is loaded. */
  function defaultOf(path: Paths<T>): unknown {
    return getAtPath(defaults.value, path);
  }

  /** isDefault reports whether a setting holds its default, which for a secret means it is empty after a save. */
  function isDefault(path: Paths<T>): boolean {
    if (!defaults.value) return true;

    if (isSecret(path)) {
      const keepsSaved = !!options.savedSecrets.value[path] && !options.clearedSecrets[path];
      return !getAtPath(state, path) && !keepsSaved;
    }
    return JSON.stringify(getAtPath(state, path)) === JSON.stringify(defaultOf(path));
  }

  /** resetField sets a setting to its default, or marks a saved secret to be removed. */
  function resetField(path: Paths<T>): void {
    if (isSecret(path)) {
      setAtPath(state, path, '');
      if (options.savedSecrets.value[path]) options.clearedSecrets[path] = true;
      return;
    }
    setAtPath(state, path, structuredClone(defaultOf(path)));
  }

  /**
   * settingPaths returns the paths of the settings in the given sections, such as "log" or "storage.s3",
   * or of every setting without sections. A list is one setting.
   */
  function settingPaths(sections?: string[]): Paths<T>[] {
    // The walk follows the defaults, which have the shape of T, so every path is one of Paths<T>
    const paths = leafPaths(defaults.value, '') as Paths<T>[];
    if (!sections) return paths;
    return paths.filter((path) =>
      sections.some((section) => path === section || path.startsWith(`${section}.`)),
    );
  }

  /**
   * isSectionDefault reports whether every setting of the given sections,
   * or of all settings without sections, holds its default.
   */
  function isSectionDefault(sections?: string[]): boolean {
    return settingPaths(sections).every(isDefault);
  }

  /** resetSections sets every setting of the given sections, or every setting without sections, to its default. */
  function resetSections(sections?: string[]): void {
    for (const path of settingPaths(sections)) {
      if (!isDefault(path)) resetField(path);
    }
  }

  /**
   * overriddenChanges returns the sources of the changed settings that a flag or an environment variable sets,
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

/** leafPaths returns the dot-separated paths of every field of value that is not an object, lists included. */
function leafPaths(value: unknown, prefix: string): string[] {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) {
    return prefix ? [prefix] : [];
  }
  return Object.entries(value).flatMap(([key, child]) =>
    leafPaths(child, prefix ? `${prefix}.${key}` : key),
  );
}
