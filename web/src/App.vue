<script setup lang="ts">
import { computed, watchEffect } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import { de, en } from '@nuxt/ui/locale';

import AppConfirmModal from '@/components/ui/AppConfirmModal.vue';
import { useMetaStore } from '@/stores/meta';

const { t, locale } = useI18n();
const route = useRoute();
const metaStore = useMetaStore();

const uiLocale = computed(() => (locale.value === 'de' ? de : en));

watchEffect(() => {
  const baseTitle = metaStore.meta?.name || 'Swim-Timer';
  const titleKey = route.meta.titleKey;
  const titleParams = route.meta.titleParams?.(route) ?? {};

  document.title = titleKey ? `${t(titleKey, titleParams)} - ${baseTitle}` : baseTitle;
});
</script>

<template>
  <UApp :locale="uiLocale">
    <RouterView />
    <AppConfirmModal />
  </UApp>
</template>
