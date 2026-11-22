<script setup lang="ts">
// A numbered step of the import dialog with its title and what to do in it, followed by its content
defineProps<{
  number: number;
  title: string;
  description: string;
}>();
</script>

<template>
  <section class="flex flex-col gap-3">
    <div class="flex items-start gap-3">
      <span
        class="bg-primary/10 text-primary flex size-6 shrink-0 items-center justify-center rounded-full text-xs font-semibold"
      >
        {{ number }}
      </span>
      <div class="flex flex-col gap-0.5">
        <h3 class="text-highlighted heading-card">{{ title }}</h3>
        <p class="text-muted text-sm">{{ description }}</p>
      </div>
    </div>
    <slot />
  </section>
</template>
