import { describe, expect, it, vi } from 'vitest';

import type { ConfigSettings } from '@/api/types';
import { CONFIG_LIMITS, provideConfigForm, type ConfigForm } from '@/composables/useConfigForm';
import { withSetup } from '@/test/withSetup';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    te: () => false,
  }),
}));

/** mountForm creates the configuration form in a mounted component and loads the default configuration. */
function mountForm(secrets: Record<string, boolean> = {}): ConfigForm {
  const { result } = withSetup(() => provideConfigForm());
  const settings = JSON.parse(JSON.stringify(result.state)) as ConfigSettings;
  result.applyResponse({
    ...settings,
    secrets,
    defaults: structuredClone(settings),
    sources: {
      'app.name': {
        kind: 'env',
        name: 'APP_NAME',
      },
    },
    file: {
      path: '/etc/config.yaml',
      exists: true,
    },
  });
  return result;
}

/** failedFields returns the names of the fields that fail validation. */
function failedFields(form: ConfigForm): (string | undefined)[] {
  return form.validate(form.state).map((error) => error.name);
}

describe('provideConfigForm', () => {
  describe('isDirty', () => {
    it('counts a secret marked to be cleared as a change', () => {
      const form = mountForm({
        'app.secret': true,
      });

      form.clearedSecrets['app.secret'] = true;

      expect(form.isDirty.value).toBe(true);
    });
  });

  describe('isOverridden', () => {
    it('reports a setting that an environment variable sets', () => {
      const form = mountForm();

      const overridden = form.isOverridden('app.name');

      expect(overridden).toBe(true);
    });
  });

  describe('isDefault', () => {
    it('compares a setting with the defaults of the loaded configuration', () => {
      const form = mountForm();

      const isDefault = form.isDefault('app.name');

      expect(isDefault).toBe(true);
    });
  });

  describe('resetSections', () => {
    it('resets every setting as a change to save, removing saved secrets', () => {
      const form = mountForm({
        'app.secret': true,
      });
      form.state.server.port = 9000;

      form.resetSections();

      expect(form.state.server.port).toBe(8080);
      expect(form.isDirty.value).toBe(true);
      expect(form.saveRequest().clearSecrets).toEqual(['app.secret']);
    });

    it('resets the log directory and the recording to their defaults', () => {
      const form = mountForm();
      form.state.log.path = '/srv/logs';
      form.state.timing.recording = false;
      expect(form.isDefault('log.path')).toBe(false);
      expect(form.isDefault('timing.recording')).toBe(false);

      form.resetSections(['log', 'timing']);

      expect(form.state.log.path).toBe('');
      expect(form.state.timing.recording).toBe(true);
    });
  });

  describe('overriddenChanges', () => {
    it('names a changed setting that an environment variable overrides', () => {
      const form = mountForm();
      form.state.app.name = 'Changed';

      const sources = form.overriddenChanges();

      expect(sources.map((source) => source.name)).toEqual(['APP_NAME']);
    });
  });

  describe('applyResponse', () => {
    it('starts without changes after a configuration is loaded', () => {
      const form = mountForm();

      expect(form.isDirty.value).toBe(false);
      expect(failedFields(form)).toEqual([]);
    });

    it('keeps the config file of a loaded configuration', () => {
      const form = mountForm();

      expect(form.file.value.path).toBe('/etc/config.yaml');
    });
  });

  describe('resetChanges', () => {
    it('restores the loaded configuration', () => {
      const form = mountForm({
        'app.secret': true,
      });
      form.state.app.name = 'Changed';
      form.clearedSecrets['app.secret'] = true;

      form.resetChanges();

      expect(form.state.app.name).toBe('');
      expect(form.isDirty.value).toBe(false);
    });
  });

  describe('showsSslFiles', () => {
    it.each([
      {
        name: 'HTTPS is on',
        enabled: true,
        cert: '',
        key: '',
      },
      {
        name: 'the certificate path has a value',
        enabled: false,
        cert: '/etc/ssl/cert.pem',
        key: '',
      },
      {
        name: 'the key path has a value',
        enabled: false,
        cert: '',
        key: '/etc/ssl/key.pem',
      },
    ])('shows the certificate and key paths while $name', ({ enabled, cert, key }) => {
      const form = mountForm();
      form.state.server.sslEnabled = enabled;
      form.state.server.sslCertFile = cert;
      form.state.server.sslKeyFile = key;

      const shows = form.showsSslFiles.value;

      expect(shows).toBe(true);
    });

    it('hides the certificate and key paths while HTTPS is off and both paths are empty', () => {
      const form = mountForm();

      const shows = form.showsSslFiles.value;

      expect(shows).toBe(false);
    });
  });

  describe('saveRequest', () => {
    it('sends the paths of the secrets to clear', () => {
      const form = mountForm({
        'app.secret': true,
      });
      form.clearedSecrets['app.secret'] = true;

      const request = form.saveRequest();

      expect(request.clearSecrets).toEqual(['app.secret']);
    });
  });

  describe('validate', () => {
    it('leaves the application name optional, since an empty name selects the default', () => {
      const form = mountForm();
      form.state.app.name = '';

      const fields = failedFields(form);

      expect(fields).toEqual([]);
    });

    it('accepts an empty log directory for a log channel that writes a file', () => {
      const form = mountForm();
      form.state.log.channel = 'file';

      const fields = failedFields(form);

      expect(fields).toEqual([]);
    });

    it('keeps a saved S3 secret key without asking for it again', () => {
      const form = mountForm({
        'storage.s3.secretKey': true,
      });
      form.state.storage.driver = 's3';

      const fields = failedFields(form);

      expect(fields).not.toContain('storage.s3.secretKey');
    });

    it('rejects texts over the maximum length', () => {
      const form = mountForm();
      form.state.app.name = 'x'.repeat(101);

      const fields = failedFields(form);

      expect(fields).toEqual(['app.name']);
    });

    it('limits the length of the log directory', () => {
      const form = mountForm();
      form.state.log.path = 'a'.repeat(CONFIG_LIMITS.logPath + 1);

      const fields = failedFields(form);

      expect(fields).toEqual(['log.path']);
    });

    it('shows a CORS origin over the maximum length at the list of origins', () => {
      const form = mountForm();
      form.state.server.corsAllowedOrigins = [
        'https://example.com',
        `https://${'x'.repeat(250)}.com`,
      ];

      const fields = failedFields(form);

      expect(fields).toEqual(['server.corsAllowedOrigins']);
    });

    it('requires the S3 settings for the S3 driver', () => {
      const form = mountForm();
      form.state.storage.driver = 's3';

      const fields = failedFields(form);

      expect(fields).toEqual([
        'storage.s3.endpoint',
        'storage.s3.bucket',
        'storage.s3.accessKey',
        'storage.s3.secretKey',
      ]);
    });

    it.each([
      {
        name: 'key',
        cert: '',
        key: '/etc/ssl/key.pem',
        missing: 'server.sslCertFile',
      },
      {
        name: 'certificate',
        cert: '/etc/ssl/cert.pem',
        key: '',
        missing: 'server.sslKeyFile',
      },
    ])(
      'requires the TLS certificate and key together for a $name alone',
      ({ cert, key, missing }) => {
        const form = mountForm();
        form.state.server.sslCertFile = cert;
        form.state.server.sslKeyFile = key;

        const fields = failedFields(form);

        expect(fields).toEqual([missing]);
      },
    );
  });
});
