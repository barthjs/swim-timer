<script setup lang="ts">
import { Tag } from 'lucide-vue-next';

const version = import.meta.env.VITE_APP_VERSION || 'dev';
</script>

<template>
  <a
    href="https://github.com/barthjs/swim-timer/releases"
    target="_blank"
    rel="noopener noreferrer"
    class="text-muted hover:text-fg flex items-center justify-center gap-1.5 transition-colors"
  >
    <Tag :size="12" />
    <span class="font-mono text-xs">{{ version }}</span>
  </a>
</template>
