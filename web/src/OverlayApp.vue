<script setup lang="ts">
import { watchEffect } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import { useMetaStore } from '@/stores/meta';

const { t } = useI18n();
const route = useRoute();
const metaStore = useMetaStore();

watchEffect(() => {
  const baseTitle = metaStore.meta?.name || 'Swim-Timer';
  const titleKey = route.meta.titleKey;
  const titleParams = route.meta.titleParams?.(route) ?? {};

  document.title = titleKey ? `${t(titleKey, titleParams)} - ${baseTitle}` : baseTitle;
});
</script>

<template>
  <RouterView />
</template>
