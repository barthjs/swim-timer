<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { useCopy } from '@/composables/ui/useCopy';

export type OverlayLink = {
  label: string;
  path: string;
  // Spans both columns, for a link that stands apart from the others
  wide?: boolean;
};

// Lists overlay pages with their absolute URL for the browser source of the streaming software.
// The overlays are a separate app (see overlay.ts), so they are linked by path instead of through the admin router.
const props = defineProps<{
  title: string;
  icon: string;
  links: OverlayLink[];
}>();

const { t } = useI18n();
const { copy } = useCopy();

const rows = computed(() =>
  props.links.map((link) => ({
    label: link.label,
    url: new URL(link.path, window.location.origin).href,
    wide: link.wide ?? false,
  })),
);

/** Copies the URL of an overlay to the clipboard. */
async function copyUrl(url: string): Promise<void> {
  await copy(url, t('pages.dashboard.urlCopied'));
}
</script>

<template>
  <UCard variant="subtle">
    <template #header>
      <div class="flex items-center gap-2">
        <UIcon :name="icon" class="text-primary size-5" />
        <h2 class="text-highlighted heading-section">{{ title }}</h2>
      </div>
    </template>

    <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
      <div
        v-for="row in rows"
        :key="row.url"
        class="border-default bg-default hover:bg-elevated flex items-center gap-2 rounded-md border px-3 py-2 transition-colors"
        :class="{ 'sm:col-span-2': row.wide }"
      >
        <div class="min-w-0 flex-1">
          <p class="text-highlighted truncate text-sm font-medium">{{ row.label }}</p>
          <p class="text-muted truncate font-mono text-xs" :title="row.url">{{ row.url }}</p>
        </div>
        <UTooltip :text="t('pages.dashboard.copyUrl')">
          <UButton
            icon="i-lucide-copy"
            color="neutral"
            variant="ghost"
            size="sm"
            :aria-label="t('pages.dashboard.copyUrl')"
            @click="copyUrl(row.url)"
          />
        </UTooltip>
        <UTooltip :text="t('pages.dashboard.openUrl')">
          <UButton
            icon="i-lucide-external-link"
            color="neutral"
            variant="ghost"
            size="sm"
            :href="row.url"
            target="_blank"
            :aria-label="t('pages.dashboard.openUrl')"
          />
        </UTooltip>
      </div>
    </div>
  </UCard>
</template>
