<script setup lang="ts">
import { reactive, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { ApiError } from '@/api/errors';
import { authRepository } from '@/api/repositories/auth';
import type { AuthLoginRequest } from '@/api/types';
import PasswordField from '@/components/form/PasswordField.vue';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import { useMutation } from '@/composables/form/useMutation';
import { redirectTarget } from '@/lib/auth/redirect';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';
import { useAuthStore } from '@/stores/auth';

const { t } = useI18n();
const router = useRouter();
const route = useRoute();

const authStore = useAuthStore();

// The server accepts an empty password while no admin password is set, so the form has no client validation
const state = reactive<AuthLoginRequest>({
  password: '',
});
const form = useTemplateRef<FormErrorTarget>('form');

const {
  mutate: login,
  pending,
  errorMessage,
} = useMutation(
  async () => {
    try {
      await authRepository.login({
        password: state.password,
      });
    } catch (err) {
      // The server rejects a login with a session that another tab already started, which counts as logged in
      if (err instanceof ApiError && err.status === 409) return;
      throw err;
    }
  },
  {
    form,
    values: state,
    toast: false,
    onSuccess: async () => {
      await authStore.refresh();
      await router.push(
        redirectTarget(route.query, {
          name: 'dashboard',
        }),
      );
    },
  },
);
</script>

<template>
  <div class="relative flex flex-1 items-center justify-center">
    <UButton
      :label="t('common.back')"
      icon="i-lucide-arrow-left"
      color="neutral"
      variant="ghost"
      :to="{ name: 'dashboard' }"
      class="absolute top-4 left-4"
    />

    <UCard class="w-full max-w-sm">
      <template #header>
        <div class="flex flex-col items-center gap-3">
          <img src="@/assets/images/logo/favicon.svg" alt="" class="size-12" />
          <h1 class="text-highlighted text-center text-xl font-semibold">
            {{ t('pages.login.title') }}
          </h1>
        </div>
      </template>

      <UForm
        :validate-on="FORM_VALIDATE_ON"
        ref="form"
        :state="state"
        class="space-y-4"
        @submit="login()"
      >
        <UFormField :label="t('fields.password')" name="password">
          <PasswordField v-model="state.password" autocomplete="current-password" autofocus />
        </UFormField>

        <p v-if="errorMessage" class="text-error text-sm">{{ errorMessage }}</p>

        <UButton type="submit" :label="t('pages.login.submit')" :loading="pending" block />
      </UForm>
    </UCard>
  </div>
</template>
