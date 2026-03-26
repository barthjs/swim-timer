<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { ChevronLeft } from 'lucide-vue-next';

import { configApi } from '@/api/configApi.ts';

import BaseButton from '@/components/BaseButton.vue';
import ConfirmModal from '@/components/ConfirmModal.vue';
import FormInput from '@/components/FormInput.vue';
import SelectDropdown from '@/components/SelectDropdown.vue';

import { useFormValidation } from '@/composables/useFormValidation.ts';
import { useUnsavedChanges } from '@/composables/useUnsavedChanges.ts';

import { useToastStore } from '@/stores/toastStore.ts';

import type { Config } from '@/types';
import { getErrorMessageKey, isApiError, isValidationError } from '@/types/api.ts';

const loading = ref(false);
const saving = ref(false);
const hasChanges = ref(false);
const showConfirmModal = ref(false);

const { t } = useI18n();

const toastStore = useToastStore();
const { errors, setErrors, clearErrors } = useFormValidation();

useUnsavedChanges(hasChanges);

const config = reactive<Config>({
  config_file_path: '',
  log_level: '',
  pool_length: 0,
  presets_dir: '',
  race_data_dir: '',
  serial_port: '',
  simulate_event: 0,
  server: {
    host: '',
    port: 8080,
  },
  secret: '',
});

const initialConfig = reactive<Config>({
  config_file_path: '',
  log_level: '',
  pool_length: 0,
  presets_dir: '',
  race_data_dir: '',
  serial_port: '',
  simulate_event: 0,
  server: {
    host: '',
    port: 8080,
  },
  secret: '',
});

const logLevelOptions = ['debug', 'info', 'warn', 'error'];

const loadConfig = async () => {
  loading.value = true;

  try {
    const data = await configApi.getConfig();
    config.config_file_path = data.config_file_path;
    config.log_level = data.log_level;
    config.pool_length = data.pool_length;
    config.presets_dir = data.presets_dir;
    config.race_data_dir = data.race_data_dir;
    config.serial_port = data.serial_port;
    config.simulate_event = data.simulate_event;
    config.server.host = data.server.host;
    config.server.port = data.server.port;
    config.secret = data.secret;

    initialConfig.config_file_path = data.config_file_path;
    initialConfig.log_level = data.log_level;
    initialConfig.pool_length = data.pool_length;
    initialConfig.presets_dir = data.presets_dir;
    initialConfig.race_data_dir = data.race_data_dir;
    initialConfig.serial_port = data.serial_port;
    initialConfig.simulate_event = data.simulate_event;
    initialConfig.server.host = data.server.host;
    initialConfig.server.port = data.server.port;
    initialConfig.secret = data.secret;

    hasChanges.value = false;
  } catch (error) {
    console.error('Failed to load config:', error);

    if (isApiError(error)) {
      toastStore.error(t(getErrorMessageKey(error)));
    } else {
      toastStore.error(t('config.loadError'));
    }
  } finally {
    loading.value = false;
  }
};

const handleConfirmSave = async () => {
  showConfirmModal.value = false;
  saving.value = true;

  try {
    await configApi.updateConfig(config);

    initialConfig.config_file_path = config.config_file_path;
    initialConfig.log_level = config.log_level;
    initialConfig.pool_length = config.pool_length;
    initialConfig.presets_dir = config.presets_dir;
    initialConfig.race_data_dir = config.race_data_dir;
    initialConfig.serial_port = config.serial_port;
    initialConfig.simulate_event = config.simulate_event;
    initialConfig.server.host = config.server.host;
    initialConfig.server.port = config.server.port;
    initialConfig.secret = config.secret;

    hasChanges.value = false;
    clearErrors();
    toastStore.warning(t('config.restartRequired'));
  } catch (error) {
    console.error('Failed to save config:', error);

    if (isValidationError(error)) {
      setErrors(error.errors);
    } else if (isApiError(error)) {
      toastStore.error(t(getErrorMessageKey(error)));
    } else {
      toastStore.error(t('config.saveError'));
    }
  } finally {
    saving.value = false;
  }
};

const openSaveConfirm = () => {
  showConfirmModal.value = true;
};

const discardChanges = () => {
  Object.assign(config, initialConfig);
  hasChanges.value = false;
  clearErrors();
};

const updateField = (field: string, value: string | number): void => {
  const keys = field.split('.');
  let obj: Record<string, unknown> = config;
  for (let i = 0; i < keys.length - 1; i++) {
    const key = keys[i];
    if (!key) return;
    obj = obj[key] as Record<string, unknown>;
  }
  const lastKey = keys[keys.length - 1];
  if (lastKey) {
    obj[lastKey] = value;
  }

  hasChanges.value = JSON.stringify(config) !== JSON.stringify(initialConfig);
};

onMounted(() => {
  loadConfig();
});
</script>

<template>
  <div class="mx-auto flex w-full max-w-2xl flex-col space-y-6">
    <div class="flex items-center gap-2">
      <RouterLink :to="{ name: 'home' }">
        <BaseButton
          variant="ghost"
          :icon="ChevronLeft"
          size="icon"
          :title="$t('common.backToHome')"
        />
      </RouterLink>
      <h1 class="text-3xl font-bold">{{ $t('config.title') }}</h1>
    </div>

    <div v-if="loading" class="py-8 text-center">
      <p class="text-muted">{{ $t('config.loading') }}</p>
    </div>

    <div v-else>
      <div class="bg-warning/10 border-warning mb-6 rounded-lg border p-4">
        <p class="text-warning text-sm">{{ $t('config.configMismatch') }}</p>
      </div>

      <div
        v-if="config.config_file_path"
        class="bg-surface border-subtle mb-6 rounded-lg border p-4"
      >
        <p class="text-fg-secondary text-xs font-medium">{{ $t('config.configFilePath') }}</p>
        <p class="text-fg mt-1 font-mono text-sm break-all">{{ config.config_file_path }}</p>
      </div>

      <form
        @submit.prevent="openSaveConfirm"
        class="bg-surface border-subtle space-y-6 rounded-lg border p-6"
      >
        <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
          <div class="flex flex-col gap-2">
            <label class="text-fg text-sm font-medium">{{ $t('config.logLevel') }}</label>
            <SelectDropdown
              :model-value="config.log_level"
              :options="logLevelOptions"
              :placeholder="$t('config.logLevel')"
              @update:model-value="updateField('log_level', $event)"
            />
            <p v-if="errors.LogLevel" class="text-danger text-sm">
              {{ errors.LogLevel }}
            </p>
          </div>

          <FormInput
            :label="$t('config.poolLength')"
            type="number"
            :model-value="config.pool_length"
            :error="errors.PoolLength"
            :min="1"
            :max="1000000"
            @update:model-value="updateField('pool_length', $event)"
          />
        </div>

        <FormInput
          :label="$t('config.presetsDir')"
          type="text"
          :model-value="config.presets_dir"
          :error="errors.PresetDir"
          required
          @update:model-value="updateField('presets_dir', $event)"
        />

        <FormInput
          :label="$t('config.raceDataDir')"
          type="text"
          :model-value="config.race_data_dir"
          :error="errors.RaceDataDir"
          required
          @update:model-value="updateField('race_data_dir', $event)"
        />

        <FormInput
          :label="$t('config.serialPort')"
          type="text"
          :model-value="config.serial_port"
          :error="errors.SerialPort"
          @update:model-value="updateField('serial_port', $event)"
        />

        <FormInput
          :label="$t('config.simulateEvent')"
          type="number"
          :model-value="config.simulate_event"
          :error="errors.SimulateEvent"
          :min="0"
          :max="1000000"
          @update:model-value="updateField('simulate_event', $event)"
        />

        <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
          <FormInput
            :label="$t('config.serverHost')"
            type="text"
            :model-value="config.server.host"
            :error="errors['Server.Host']"
            @update:model-value="updateField('server.host', $event)"
          />

          <FormInput
            :label="$t('config.serverPort')"
            type="number"
            :model-value="config.server.port"
            :error="errors['Server.Port']"
            :min="1"
            :max="65535"
            @update:model-value="updateField('server.port', $event)"
          />
        </div>

        <FormInput
          :label="$t('config.secret')"
          type="password"
          :model-value="config.secret"
          :error="errors.Secret"
          @update:model-value="updateField('secret', $event)"
        />

        <div class="border-subtle flex justify-end gap-3 border-t pt-4">
          <BaseButton v-if="hasChanges" variant="danger" @click="discardChanges">
            {{ $t('common.discardChanges') }}
          </BaseButton>
          <BaseButton variant="primary" type="submit" :disabled="!hasChanges || saving">
            {{ saving ? $t('config.loading') : $t('common.save') }}
          </BaseButton>
        </div>
      </form>
    </div>

    <ConfirmModal
      :is-open="showConfirmModal"
      :message="$t('config.saveConfirm')"
      @confirm="handleConfirmSave"
      @cancel="showConfirmModal = false"
    />
  </div>
</template>
