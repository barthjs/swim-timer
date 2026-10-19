<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import { type DocsPage, docsUrl } from '@/lib/docs';

// A button next to the title of a page that opens its section of the documentation in a new tab.
defineProps<{
  page: DocsPage;
}>();

const { t } = useI18n();
</script>

<template>
  <UTooltip :text="t('admin.pageDocs')">
    <UButton
      icon="i-lucide-book-open"
      color="neutral"
      variant="ghost"
      :aria-label="t('admin.pageDocs')"
      :href="docsUrl(page)"
      target="_blank"
    />
  </UTooltip>
</template>
