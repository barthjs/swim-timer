<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { configRepository } from '@/api/repositories/config';
import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import HelpHint from '@/components/form/HelpHint.vue';
import SecretField from '@/components/form/SecretField.vue';
import { CONFIG_LIMITS, useConfigForm } from '@/composables/useConfigForm';
import { useConnectionTestReport } from '@/composables/useConnectionTestReport';
import { useMutation } from '@/composables/form/useMutation';
import { countLabel } from '@/lib/form/validation';

// Where the files of the application are stored, with a test of the connection.
const { t } = useI18n();
const { state, fieldProps, savedSecrets, clearedSecrets, keepsSecret, hasSecret } = useConfigForm();
const { reportTestConnection } = useConnectionTestReport();

const driverItems = computed(() => [
  {
    label: t('pages.config.storage.local'),
    value: 'local',
  },
  {
    label: 'S3',
    value: 's3',
  },
]);

const { mutate: testStorage, pending: testingStorage } = useMutation(
  () => configRepository.testStorage(state.storage),
  {
    onSuccess: reportTestConnection,
  },
);

const storageTestDisabled = computed(() => {
  if (state.storage.driver !== 's3') return false;
  const s3 = state.storage.s3;
  // A saved secret key is used for the test when the field is left empty
  return !(
    s3.endpoint.trim() &&
    s3.bucket.trim() &&
    s3.accessKey.trim() &&
    hasSecret('storage.s3.secretKey', s3.secretKey)
  );
});
</script>

<template>
  <UCard variant="subtle" class="mt-4">
    <div class="flex flex-col gap-4">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <UFormField :label="t('fields.storage.driver')" name="storage.driver">
          <template #hint>
            <ConfigFieldHint
              path="storage.driver"
              :help="t('pages.config.storage.driverHint')"
              :options="driverItems"
            />
          </template>
          <USelect
            v-model="state.storage.driver"
            v-bind="fieldProps('storage.driver')"
            :items="driverItems"
            class="w-full"
          />
        </UFormField>

        <UFormField
          v-if="state.storage.driver === 'local'"
          :label="t('fields.storage.path')"
          name="storage.path"
        >
          <template #hint>
            <ConfigFieldHint
              path="storage.path"
              :help="t('pages.config.storage.pathHint')"
              :count="countLabel(state.storage.path, CONFIG_LIMITS.storagePath)"
            />
          </template>
          <UInput
            v-model="state.storage.path"
            v-bind="fieldProps('storage.path')"
            :placeholder="t('pages.config.storage.pathPlaceholder')"
            :maxlength="CONFIG_LIMITS.storagePath"
            class="w-full"
          />
        </UFormField>
      </div>

      <div v-if="state.storage.driver === 's3'" class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <UFormField :label="t('fields.storage.s3.endpoint')" name="storage.s3.endpoint" required>
          <template #hint>
            <ConfigFieldHint path="storage.s3.endpoint" />
          </template>
          <UInput
            v-model="state.storage.s3.endpoint"
            v-bind="fieldProps('storage.s3.endpoint')"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.storage.s3.region')" name="storage.s3.region">
          <template #hint>
            <ConfigFieldHint path="storage.s3.region" />
          </template>
          <UInput
            v-model="state.storage.s3.region"
            v-bind="fieldProps('storage.s3.region')"
            class="w-full"
          />
        </UFormField>
        <UFormField
          :label="t('fields.storage.s3.bucket')"
          name="storage.s3.bucket"
          required
          class="sm:col-span-2"
        >
          <template #hint>
            <ConfigFieldHint path="storage.s3.bucket" />
          </template>
          <UInput
            v-model="state.storage.s3.bucket"
            v-bind="fieldProps('storage.s3.bucket')"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.storage.s3.accessKey')" name="storage.s3.accessKey" required>
          <template #hint>
            <ConfigFieldHint path="storage.s3.accessKey" />
          </template>
          <UInput
            v-model="state.storage.s3.accessKey"
            v-bind="fieldProps('storage.s3.accessKey')"
            class="w-full"
          />
        </UFormField>
        <UFormField
          :label="t('fields.storage.s3.secretKey')"
          name="storage.s3.secretKey"
          :required="!keepsSecret('storage.s3.secretKey')"
        >
          <template #hint>
            <ConfigFieldHint path="storage.s3.secretKey" />
          </template>
          <SecretField
            v-model="state.storage.s3.secretKey"
            v-model:cleared="clearedSecrets['storage.s3.secretKey']"
            :saved="!!savedSecrets['storage.s3.secretKey']"
            v-bind="fieldProps('storage.s3.secretKey')"
          />
        </UFormField>
        <UFormField :label="t('fields.storage.s3.usePathStyle')" name="storage.s3.usePathStyle">
          <template #hint>
            <ConfigFieldHint
              path="storage.s3.usePathStyle"
              :help="t('pages.config.storage.usePathStyleHint')"
            />
          </template>
          <USwitch
            v-model="state.storage.s3.usePathStyle"
            v-bind="fieldProps('storage.s3.usePathStyle')"
          />
        </UFormField>
      </div>

      <div class="flex items-center gap-1.5">
        <UButton
          :label="t('pages.config.testConnection')"
          icon="i-lucide-plug-zap"
          color="neutral"
          variant="outline"
          :loading="testingStorage"
          :disabled="storageTestDisabled"
          @click="testStorage()"
        />
        <HelpHint :text="t('pages.config.storage.testHint')" />
      </div>
    </div>
  </UCard>
</template>
