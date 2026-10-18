<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import SecretField from '@/components/form/SecretField.vue';
import { CONFIG_LIMITS, useConfigForm } from '@/composables/useConfigForm';
import { countLabel } from '@/lib/form/validation';

// The general settings of the application, its logging, and the admin access.
const { t } = useI18n();
const { state, fieldProps, savedSecrets, clearedSecrets } = useConfigForm();

// The language names stay in their own language, so everyone finds theirs
const languageItems = [
  {
    label: 'English',
    value: 'en',
  },
  {
    label: 'Deutsch',
    value: 'de',
  },
];

const envItems = computed(() => [
  {
    label: t('pages.config.app.envDevelopment'),
    value: 'development',
  },
  {
    label: t('pages.config.app.envStaging'),
    value: 'staging',
  },
  {
    label: t('pages.config.app.envProduction'),
    value: 'production',
  },
]);

const logLevelItems = computed(() => [
  {
    label: t('pages.config.log.levelDebug'),
    value: 'debug',
  },
  {
    label: t('pages.config.log.levelInfo'),
    value: 'info',
  },
  {
    label: t('pages.config.log.levelWarn'),
    value: 'warn',
  },
  {
    label: t('pages.config.log.levelError'),
    value: 'error',
  },
]);

const logChannelItems = computed(() => [
  {
    label: t('pages.config.log.channelStdout'),
    value: 'stdout',
  },
  {
    label: t('pages.config.log.channelFile'),
    value: 'file',
  },
  {
    label: t('pages.config.log.channelBoth'),
    value: 'both',
  },
]);

const logFormatItems = computed(() => [
  {
    label: 'JSON',
    value: 'json',
  },
  {
    label: t('pages.config.log.formatConsole'),
    value: 'console',
  },
]);
</script>

<template>
  <UCard variant="subtle" class="mt-4">
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <UFormField :label="t('fields.app.name')" name="app.name">
        <template #hint>
          <ConfigFieldHint
            path="app.name"
            :help="t('pages.config.app.nameHint')"
            :count="countLabel(state.app.name, CONFIG_LIMITS.appName)"
          />
        </template>
        <UInput
          v-model="state.app.name"
          v-bind="fieldProps('app.name')"
          :maxlength="CONFIG_LIMITS.appName"
          class="w-full"
        />
      </UFormField>
      <UFormField :label="t('fields.app.language')" name="app.language">
        <template #hint>
          <ConfigFieldHint
            path="app.language"
            :help="t('pages.config.app.languageHint')"
            :options="languageItems"
          />
        </template>
        <USelect
          v-model="state.app.language"
          v-bind="fieldProps('app.language')"
          :items="languageItems"
          class="w-full"
        />
      </UFormField>
      <UFormField :label="t('fields.app.env')" name="app.env">
        <template #hint>
          <ConfigFieldHint
            path="app.env"
            :help="t('pages.config.app.envHint')"
            :options="envItems"
          />
        </template>
        <USelect
          v-model="state.app.env"
          v-bind="fieldProps('app.env')"
          :items="envItems"
          class="w-full"
        />
      </UFormField>
      <UFormField :label="t('fields.log.level')" name="log.level">
        <template #hint>
          <ConfigFieldHint path="log.level" :options="logLevelItems" />
        </template>
        <USelect
          v-model="state.log.level"
          v-bind="fieldProps('log.level')"
          :items="logLevelItems"
          class="w-full"
        />
      </UFormField>
      <UFormField :label="t('fields.log.channel')" name="log.channel" class="sm:col-start-1">
        <template #hint>
          <ConfigFieldHint path="log.channel" :options="logChannelItems" />
        </template>
        <USelect
          v-model="state.log.channel"
          v-bind="fieldProps('log.channel')"
          :items="logChannelItems"
          class="w-full"
        />
      </UFormField>
      <UFormField :label="t('fields.log.format')" name="log.format" class="sm:col-start-1">
        <template #hint>
          <ConfigFieldHint path="log.format" :options="logFormatItems" />
        </template>
        <USelect
          v-model="state.log.format"
          v-bind="fieldProps('log.format')"
          :items="logFormatItems"
          class="w-full"
        />
      </UFormField>
      <UFormField
        v-if="state.log.channel !== 'stdout'"
        :label="t('fields.log.path')"
        name="log.path"
      >
        <template #hint>
          <ConfigFieldHint
            path="log.path"
            :help="t('pages.config.log.pathHint')"
            :count="countLabel(state.log.path, CONFIG_LIMITS.logPath)"
          />
        </template>
        <UInput
          v-model="state.log.path"
          v-bind="fieldProps('log.path')"
          :placeholder="t('pages.config.log.pathPlaceholder')"
          :maxlength="CONFIG_LIMITS.logPath"
          class="w-full"
        />
      </UFormField>
      <template v-if="state.log.channel !== 'stdout'">
        <UFormField :label="t('fields.log.maxSizeMb')" name="log.maxSizeMb">
          <template #hint>
            <ConfigFieldHint path="log.maxSizeMb" :help="t('pages.config.log.maxSizeMbHint')" />
          </template>
          <UInputNumber
            v-model="state.log.maxSizeMb"
            v-bind="fieldProps('log.maxSizeMb')"
            :min="1"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.log.maxAgeDays')" name="log.maxAgeDays">
          <template #hint>
            <ConfigFieldHint path="log.maxAgeDays" :help="t('pages.config.log.maxAgeDaysHint')" />
          </template>
          <UInputNumber
            v-model="state.log.maxAgeDays"
            v-bind="fieldProps('log.maxAgeDays')"
            :min="-1"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.log.compress')" name="log.compress">
          <template #hint>
            <ConfigFieldHint path="log.compress" />
          </template>
          <USwitch v-model="state.log.compress" v-bind="fieldProps('log.compress')" />
        </UFormField>
      </template>
      <UFormField
        :label="t('fields.app.secret')"
        name="app.secret"
        class="sm:col-span-2 sm:col-start-1"
      >
        <template #hint>
          <ConfigFieldHint path="app.secret" :help="t('pages.config.app.secretHint')" />
        </template>
        <SecretField
          v-model="state.app.secret"
          v-model:cleared="clearedSecrets['app.secret']"
          :saved="!!savedSecrets['app.secret']"
          v-bind="fieldProps('app.secret')"
        />
      </UFormField>
    </div>
  </UCard>
</template>
