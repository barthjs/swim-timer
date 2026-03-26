<script setup lang="ts">
import { Check, X } from 'lucide-vue-next';

import BaseButton from '@/components/BaseButton.vue';

defineProps<{
  isOpen: boolean;
  message: string;
}>();

const emit = defineEmits<{
  (e: 'confirm'): void;
  (e: 'cancel'): void;
}>();
</script>

<template>
  <div
    v-if="isOpen"
    class="bg-base/80 fixed inset-0 z-9999 flex items-center justify-center"
    @mousedown.stop
  >
    <div
      class="border-subtle bg-surface flex flex-col items-center justify-center space-y-4 rounded-lg border p-8 shadow-lg"
    >
      <p>{{ message }}</p>
      <div class="flex justify-center space-x-4">
        <BaseButton variant="danger" :icon="X" @click="emit('cancel')">
          {{ $t('common.cancel') }}
        </BaseButton>
        <BaseButton variant="primary" :icon="Check" @click="emit('confirm')">
          {{ $t('common.confirm') }}
        </BaseButton>
      </div>
    </div>
  </div>
</template>
