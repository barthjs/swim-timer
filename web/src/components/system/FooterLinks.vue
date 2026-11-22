<script setup lang="ts">
/** A link of the footer with its icon and its label, which the tooltip shows. */
export type FooterLink = {
  label: string;
  icon: string;
  to: string;
  // A link outside the app opens in a new tab
  external?: boolean;
};

// The icon links of the footer of the sidebar in their order. Without links it renders nothing.
defineProps<{
  links: FooterLink[];
}>();
</script>

<template>
  <div v-if="links.length > 0" class="flex items-center gap-0.5">
    <UTooltip v-for="link in links" :key="link.to" :text="link.label">
      <UButton
        :to="link.to"
        :external="link.external"
        :target="link.external ? '_blank' : undefined"
        :icon="link.icon"
        size="xs"
        color="neutral"
        variant="ghost"
        :aria-label="link.label"
      />
    </UTooltip>
  </div>
</template>
