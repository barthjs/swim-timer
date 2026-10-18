import { describe, expect, it } from 'vitest';
import { reactive, ref } from 'vue';

import type { ConfigSource } from '@/api/types';
import { useConfigMeta } from '@/composables/useConfigMeta';
import { useDirtyForm } from '@/composables/form/useDirtyForm';

type Settings = {
  app: {
    name: string;
    secret: string;
  };
  server: {
    port: number;
    origins: string[];
  };
};

/** defaultSettings returns the defaults of the sample configuration. */
function defaultSettings(): Settings {
  return {
    app: {
      name: 'Default',
      secret: '',
    },
    server: {
      port: 8080,
      origins: [],
    },
  };
}

/** setup loads a changed sample configuration with a saved secret and the given sources. */
function setup(sources: Record<string, ConfigSource> = {}) {
  const state = reactive<Settings>({
    app: {
      name: 'Changed',
      secret: '',
    },
    server: {
      port: 9000,
      origins: ['https://example.com'],
    },
  });
  const dirtyForm = useDirtyForm(state);
  dirtyForm.capture();
  const savedSecrets = ref<Record<string, boolean>>({
    'app.secret': true,
  });
  const clearedSecrets = reactive<Record<string, boolean>>({});

  const meta = useConfigMeta(state, {
    savedSecrets,
    clearedSecrets,
    isFieldDirty: dirtyForm.isFieldDirty,
  });
  meta.applyMeta({
    defaults: defaultSettings(),
    sources,
    file: {
      path: '/etc/app/config.yaml',
      exists: true,
    },
  });

  return {
    state,
    clearedSecrets,
    meta,
  };
}

describe('useConfigMeta', () => {
  describe('isOverridden', () => {
    const sources: Record<string, ConfigSource> = {
      'app.name': {
        kind: 'env',
        name: 'APP_NAME',
      },
      'server.port': {
        kind: 'flag',
        name: '--server-port',
      },
      'server.origins': {
        kind: 'file',
        name: '/etc/app/config.yaml',
      },
    };

    it.each([
      {
        name: 'an environment variable',
        path: 'app.name',
      },
      {
        name: 'a flag',
        path: 'server.port',
      },
    ] as const)('reports $name as an override of the file', ({ path }) => {
      const { meta } = setup(sources);

      const overridden = meta.isOverridden(path);

      expect(overridden).toBe(true);
    });

    it.each([
      {
        name: 'the file',
        path: 'server.origins',
      },
      {
        name: 'the defaults',
        path: 'app.secret',
      },
    ] as const)('does not report a setting from $name as an override', ({ path }) => {
      const { meta } = setup(sources);

      const overridden = meta.isOverridden(path);

      expect(overridden).toBe(false);
    });
  });

  describe('defaultOf', () => {
    it('returns the default of a setting', () => {
      const { meta } = setup();

      const value = meta.defaultOf('app.name');

      expect(value).toBe('Default');
    });
  });

  describe('isDefault', () => {
    it('reports a setting that equals its default', () => {
      const { state, meta } = setup();
      state.app.name = 'Default';

      const isDefault = meta.isDefault('app.name');

      expect(isDefault).toBe(true);
    });

    it('reports a changed setting as different from its default', () => {
      const { meta } = setup();

      const isDefault = meta.isDefault('app.name');

      expect(isDefault).toBe(false);
    });
  });

  describe('resetField', () => {
    it('resets a setting to a copy of its default', () => {
      const { state, meta } = setup();

      meta.resetField('server.origins');
      state.server.origins.push('https://other.example.com');

      expect(meta.defaultOf('server.origins')).toEqual([]);
    });

    it('marks a saved secret to be removed on reset', () => {
      const { state, clearedSecrets, meta } = setup();
      expect(meta.isSecret('app.secret')).toBe(true);
      expect(meta.isDefault('app.secret')).toBe(false);

      meta.resetField('app.secret');

      expect(state.app.secret).toBe('');
      expect(clearedSecrets['app.secret']).toBe(true);
      expect(meta.isDefault('app.secret')).toBe(true);
    });
  });

  describe('resetSections', () => {
    it('resets every setting without sections', () => {
      const { meta } = setup();

      meta.resetSections();

      expect(meta.isSectionDefault()).toBe(true);
    });

    it('resets only the settings of the given sections', () => {
      const { state, meta } = setup();

      meta.resetSections(['server']);

      expect(meta.isSectionDefault(['server'])).toBe(true);
      expect(state.app.name).toBe('Changed');
      expect(meta.isSectionDefault()).toBe(false);
    });
  });

  describe('overriddenChanges', () => {
    it('lists the changed settings that an override keeps from taking effect', () => {
      const env: ConfigSource = {
        kind: 'env',
        name: 'APP_NAME',
      };
      const { state, meta } = setup({
        'app.name': env,
        'server.port': {
          kind: 'flag',
          name: '--server-port',
        },
      });
      expect(meta.overriddenChanges()).toEqual([]);
      state.app.name = 'Edited';

      const changes = meta.overriddenChanges();

      expect(changes).toEqual([env]);
    });
  });
});
