<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink, useRouter } from 'vue-router';
import { Loader2 } from 'lucide-vue-next';

import { authApi } from '@/api/authApi.ts';

import BaseButton from '@/components/BaseButton.vue';

import { isApiError, getErrorMessageKey } from '@/types/api.ts';

const key = ref('');
const error = ref('');
const isLoading = ref(false);

const { t } = useI18n();
const router = useRouter();

async function handleLogin() {
  if (isLoading.value) return;
  isLoading.value = true;
  error.value = '';

  try {
    await authApi.login(key.value);
    sessionStorage.setItem('secretKey', key.value);

    const redirect = router.currentRoute.value.query.redirect as string | undefined;
    if (redirect) {
      await router.push(redirect);
    } else {
      await router.push({ name: 'home' });
    }
  } catch (err) {
    if (isApiError(err)) {
      error.value = t(getErrorMessageKey(err));
    } else {
      error.value = t('login.invalidSecret');
    }
    console.error(err);
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <div
    class="bg-surface shadow-1xl m-auto flex w-full max-w-md flex-col items-center space-y-4 rounded-xl p-8"
  >
    <RouterLink :to="{ name: 'home' }">
      <img src="@/assets/images/logo/favicon.svg" alt="Swim-Timer" class="w-24" />
    </RouterLink>

    <h2 class="text-2xl font-bold">{{ $t('login.title') }}</h2>

    <form @submit.prevent="handleLogin" class="flex w-full flex-col space-y-4">
      <input
        v-model="key"
        type="password"
        :placeholder="$t('login.secret')"
        autofocus
        class="border-subtle bg-base focus:ring-primary rounded-lg border px-4 py-3 focus:ring-2 focus:outline-none"
      />
      <p v-if="error" class="text-danger text-center font-medium">
        {{ error }}
      </p>

      <BaseButton type="submit" :disabled="isLoading" class="py-3">
        <Loader2 v-if="isLoading" class="animate-spin" :size="20" />
        <span v-else>{{ $t('login.title') }}</span>
      </BaseButton>
    </form>
  </div>
</template>
