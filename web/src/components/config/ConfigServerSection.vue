<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import { CONFIG_LIMITS, useConfigForm } from '@/composables/useConfigForm';
import { countLabel } from '@/lib/form/validation';

// How the server is reached on the network.
const { t } = useI18n();
const { state, fieldProps, showsSslFiles } = useConfigForm();

// One origin per line in the form, a list in the configuration.
const corsOrigins = computed({
  get: () => state.server.corsAllowedOrigins.join('\n'),
  set: (value: string) => {
    state.server.corsAllowedOrigins = value
      .split(/[\n,]/)
      .map((origin) => origin.trim())
      .filter(Boolean);
  },
});
</script>

<template>
  <UCard variant="subtle" class="mt-4">
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <UFormField :label="t('fields.server.host')" name="server.host">
        <template #hint>
          <ConfigFieldHint path="server.host" />
        </template>
        <UInput
          v-model="state.server.host"
          v-bind="fieldProps('server.host')"
          :placeholder="t('pages.config.server.hostPlaceholder')"
          class="w-full"
        />
      </UFormField>
      <UFormField :label="t('fields.server.port')" name="server.port">
        <template #hint>
          <ConfigFieldHint path="server.port" />
        </template>
        <UInputNumber
          v-model="state.server.port"
          v-bind="fieldProps('server.port')"
          :min="1"
          :max="65535"
          class="w-full"
        />
      </UFormField>
      <UFormField
        :label="t('fields.server.trustedProxies')"
        name="server.trustedProxies"
        class="sm:col-span-2"
      >
        <template #hint>
          <ConfigFieldHint
            path="server.trustedProxies"
            :help="t('pages.config.server.trustedProxiesHint')"
          />
        </template>
        <UTextarea
          v-model="state.server.trustedProxies"
          v-bind="fieldProps('server.trustedProxies')"
          :maxlength="CONFIG_LIMITS.serverTrustedProxies"
          :rows="2"
          class="w-full"
        />
      </UFormField>
      <UFormField
        :label="t('fields.server.sslEnabled')"
        name="server.sslEnabled"
        class="sm:col-span-2"
      >
        <template #hint>
          <ConfigFieldHint
            path="server.sslEnabled"
            :help="t('pages.config.server.sslEnabledHint')"
          />
        </template>
        <USwitch v-model="state.server.sslEnabled" v-bind="fieldProps('server.sslEnabled')" />
      </UFormField>
      <template v-if="showsSslFiles">
        <UFormField
          :label="t('fields.server.sslCertFile')"
          name="server.sslCertFile"
          :required="!!state.server.sslKeyFile"
          class="self-end"
        >
          <template #hint>
            <ConfigFieldHint
              path="server.sslCertFile"
              :help="t('pages.config.server.sslCertFileHint')"
              :count="countLabel(state.server.sslCertFile, CONFIG_LIMITS.serverSslCertFile)"
            />
          </template>
          <UInput
            v-model="state.server.sslCertFile"
            v-bind="fieldProps('server.sslCertFile')"
            :maxlength="CONFIG_LIMITS.serverSslCertFile"
            class="w-full"
          />
        </UFormField>
        <UFormField
          :label="t('fields.server.sslKeyFile')"
          name="server.sslKeyFile"
          :required="!!state.server.sslCertFile"
          class="self-end"
        >
          <template #hint>
            <ConfigFieldHint
              path="server.sslKeyFile"
              :count="countLabel(state.server.sslKeyFile, CONFIG_LIMITS.serverSslKeyFile)"
            />
          </template>
          <UInput
            v-model="state.server.sslKeyFile"
            v-bind="fieldProps('server.sslKeyFile')"
            :maxlength="CONFIG_LIMITS.serverSslKeyFile"
            class="w-full"
          />
        </UFormField>
      </template>
      <UFormField
        :label="t('fields.server.corsAllowedOrigins')"
        name="server.corsAllowedOrigins"
        class="sm:col-span-2"
      >
        <template #hint>
          <ConfigFieldHint
            path="server.corsAllowedOrigins"
            :help="t('pages.config.server.corsAllowedOriginsHint')"
          />
        </template>
        <UTextarea
          v-model="corsOrigins"
          v-bind="fieldProps('server.corsAllowedOrigins')"
          :rows="2"
          :placeholder="t('pages.config.server.corsAllowedOriginsPlaceholder')"
          class="w-full"
        />
      </UFormField>
      <UFormField
        :label="t('fields.server.corsAllowCredentials')"
        name="server.corsAllowCredentials"
        class="sm:col-span-2"
      >
        <template #hint>
          <ConfigFieldHint
            path="server.corsAllowCredentials"
            :help="t('pages.config.server.corsAllowCredentialsHint')"
          />
        </template>
        <USwitch
          v-model="state.server.corsAllowCredentials"
          v-bind="fieldProps('server.corsAllowCredentials')"
        />
      </UFormField>
    </div>
  </UCard>
</template>
