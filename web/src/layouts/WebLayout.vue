<script setup lang="ts">
import AppVersion from '@/components/AppVersion.vue';
</script>

<template>
  <div class="bg-base text-fg flex min-h-screen flex-col p-6">
    <main class="flex flex-1 flex-col">
      <RouterView />
    </main>
    <footer class="mt-8 pb-4">
      <AppVersion />
    </footer>
  </div>
</template>
