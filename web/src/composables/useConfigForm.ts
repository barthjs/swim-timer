import { computed, inject, provide, reactive, ref, type InjectionKey } from 'vue';

import type { ConfigResponse, ConfigSaveRequest, ConfigSettings } from '@/api/types';
import { useConfigMeta } from '@/composables/useConfigMeta';
import { useDirtyForm } from '@/composables/form/useDirtyForm';
import { useFormValidation } from '@/composables/form/useFormValidation';
import type { FormRules } from '@/lib/form/validation';

/** CONFIG_LIMITS are the maximum lengths the server accepts for the text fields of the configuration. */
export const CONFIG_LIMITS = {
  appName: 100,
  logPath: 1024,
  serverCorsAllowedOrigin: 255,
  serverSslCertFile: 1024,
  serverSslKeyFile: 1024,
  serverTrustedProxies: 1024,
  storagePath: 1024,
  timingReplayPath: 1024,
  timingSerialPort: 255,
} as const;

/** emptyConfig is the state shown until the configuration is loaded. */
function emptyConfig(): ConfigSettings {
  return {
    app: {
      name: '',
      env: 'production',
      language: 'en',
      secret: '',
    },
    log: {
      level: 'warn',
      channel: 'stdout',
      path: '',
      format: 'json',
      maxSizeMb: 100,
      maxAgeDays: 30,
      compress: true,
    },
    server: {
      host: '',
      port: 8080,
      trustedProxies: '',
      corsAllowedOrigins: [],
      corsAllowCredentials: false,
      sslEnabled: false,
      sslCertFile: '',
      sslKeyFile: '',
    },
    storage: {
      driver: 'local',
      path: '',
      s3: {
        endpoint: '',
        region: '',
        bucket: '',
        accessKey: '',
        secretKey: '',
        usePathStyle: true,
      },
    },
    theme: {
      primary: '',
      secondary: '',
      success: '',
      info: '',
      warning: '',
      error: '',
      neutral: '',
    },
    timing: {
      serialPort: '',
      baudRate: 9600,
      poolLength: 50,
      simulateEvent: 0,
      replayPath: '',
      recording: true,
    },
  };
}

/**
 * createConfigForm holds the edited configuration, its changes, the secrets to keep or clear,
 * and the defaults and sources of its settings.
 */
function createConfigForm() {
  const state = reactive<ConfigSettings>(emptyConfig());
  const dirtyForm = useDirtyForm(state);

  // The server never sends secrets. savedSecrets reports which are set, and a secret marked
  // in clearedSecrets is removed on save, while an empty one keeps its saved value.
  const savedSecrets = ref<Record<string, boolean>>({});
  const clearedSecrets = reactive<Record<string, boolean>>({});

  const clearSecrets = computed(() =>
    Object.entries(clearedSecrets)
      .filter(([, cleared]) => cleared)
      .map(([path]) => path),
  );

  const isDirty = computed(() => dirtyForm.isDirty.value || clearSecrets.value.length > 0);

  const meta = useConfigMeta(state, {
    savedSecrets,
    clearedSecrets,
    isFieldDirty: dirtyForm.isFieldDirty,
  });

  /** keepsSecret reports whether saving keeps a stored secret the field leaves empty. */
  function keepsSecret(path: string): boolean {
    return !!savedSecrets.value[path] && !clearedSecrets[path];
  }

  /** hasSecret reports whether a secret is available for a save or a connection test. */
  function hasSecret(path: string, value: string): boolean {
    return !!value.trim() || keepsSecret(path);
  }

  /** forgetClearedSecrets unmarks every secret that was marked to be cleared. */
  function forgetClearedSecrets(): void {
    for (const path of Object.keys(clearedSecrets)) {
      delete clearedSecrets[path];
    }
  }

  /** applyResponse shows a configuration from the server and starts tracking changes from it. */
  function applyResponse(response: ConfigResponse): void {
    const { secrets, defaults, sources, file, ...settings } = response;
    Object.assign(state, structuredClone(settings));
    savedSecrets.value = secrets;
    meta.applyMeta({
      defaults,
      sources,
      file,
    });
    forgetClearedSecrets();
    dirtyForm.capture();
  }

  /** resetChanges restores the configuration as it was loaded or saved. */
  function resetChanges(): void {
    dirtyForm.reset();
    forgetClearedSecrets();
  }

  // The certificate and key paths stay visible while either has a value even with HTTPS off,
  // because their rules apply regardless of the switch and their errors must stay visible.
  const showsSslFiles = computed(
    () => state.server.sslEnabled || !!state.server.sslCertFile || !!state.server.sslKeyFile,
  );

  /** saveRequest builds the request that saves the edited configuration. */
  function saveRequest(): ConfigSaveRequest {
    return {
      ...state,
      clearSecrets: clearSecrets.value,
    };
  }

  // The rules repeat the required fields and maximum lengths the server checks, with its tags
  const { validate } = useFormValidation<ConfigSettings>((settings): FormRules<ConfigSettings> => {
    const s3Required = settings.storage.driver === 's3' ? 'required_if' : undefined;
    return {
      'app.name': {
        max: CONFIG_LIMITS.appName,
      },
      'log.path': {
        max: CONFIG_LIMITS.logPath,
      },
      'server.corsAllowedOrigins': {
        entryMax: CONFIG_LIMITS.serverCorsAllowedOrigin,
      },
      'server.trustedProxies': {
        max: CONFIG_LIMITS.serverTrustedProxies,
      },
      'server.sslCertFile': {
        required: settings.server.sslKeyFile ? 'required_with' : undefined,
        max: CONFIG_LIMITS.serverSslCertFile,
      },
      'server.sslKeyFile': {
        required: settings.server.sslCertFile ? 'required_with' : undefined,
        max: CONFIG_LIMITS.serverSslKeyFile,
      },
      'storage.path': {
        max: CONFIG_LIMITS.storagePath,
      },
      'storage.s3.endpoint': {
        required: s3Required,
      },
      'storage.s3.bucket': {
        required: s3Required,
      },
      'storage.s3.accessKey': {
        required: s3Required,
      },
      'storage.s3.secretKey': {
        required: keepsSecret('storage.s3.secretKey') ? undefined : s3Required,
      },
      'timing.serialPort': {
        max: CONFIG_LIMITS.timingSerialPort,
      },
      'timing.replayPath': {
        max: CONFIG_LIMITS.timingReplayPath,
      },
    };
  });

  return {
    state,
    fieldProps: dirtyForm.fieldProps,
    savedSecrets,
    clearedSecrets,
    isDirty,
    keepsSecret,
    hasSecret,
    file: meta.file,
    sourceOf: meta.sourceOf,
    isOverridden: meta.isOverridden,
    defaultOf: meta.defaultOf,
    isSecret: meta.isSecret,
    isDefault: meta.isDefault,
    resetField: meta.resetField,
    isSectionDefault: meta.isSectionDefault,
    resetSections: meta.resetSections,
    overriddenChanges: meta.overriddenChanges,
    applyResponse,
    resetChanges,
    saveRequest,
    validate,
    showsSslFiles,
  };
}

export type ConfigForm = ReturnType<typeof createConfigForm>;

const configFormKey: InjectionKey<ConfigForm> = Symbol('configForm');

/** provideConfigForm creates the configuration form for a page and makes it available to its sections. */
export function provideConfigForm(): ConfigForm {
  const form = createConfigForm();
  provide(configFormKey, form);
  return form;
}

/** useConfigForm returns the configuration form provided by the configuration page. */
export function useConfigForm(): ConfigForm {
  const form = inject(configFormKey);
  if (!form) {
    throw new Error('useConfigForm must be used below provideConfigForm');
  }
  return form;
}
