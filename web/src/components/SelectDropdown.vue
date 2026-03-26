<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { ChevronDown } from 'lucide-vue-next';

defineProps<{
  modelValue: string | null;
  options: string[];
  placeholder?: string;
  emptyText?: string;
  error?: string;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | number): void;
}>();

const isOpen = ref(false);
const containerRef = ref<HTMLDivElement | null>(null);

onMounted(() => {
  const handleClickOutside = (event: MouseEvent) => {
    if (containerRef.value && !containerRef.value.contains(event.target as Node)) {
      isOpen.value = false;
    }
  };

  document.addEventListener('click', handleClickOutside);

  return () => {
    document.removeEventListener('click', handleClickOutside);
  };
});

const selectOption = (option: string) => {
  emit('update:modelValue', option);
  isOpen.value = false;
};
</script>

<template>
  <div ref="containerRef" class="flex flex-col gap-2">
    <div class="relative w-full">
      <div
        class="border-subtle bg-surface flex cursor-pointer items-center justify-between rounded-lg border px-2 py-1.5 hover:border"
        :class="{ 'border-danger': error }"
        @click="isOpen = !isOpen"
      >
        <span>{{ modelValue || placeholder || 'Select...' }}</span>
        <ChevronDown :size="16" />
      </div>

      <ul
        v-if="isOpen"
        class="border-subtle bg-surface absolute top-full left-0 z-1001 max-h-80 w-full list-none overflow-y-auto rounded-b-lg border p-0"
      >
        <li
          v-for="option in options"
          :key="option"
          @click="selectOption(option)"
          class="hover:bg-subtle cursor-pointer px-2 py-1.5"
          :class="{ 'bg-primary': modelValue === option }"
        >
          {{ option }}
        </li>
        <li v-if="options.length === 0" class="text-muted cursor-default px-2 py-1.5 italic">
          {{ emptyText || 'No options' }}
        </li>
      </ul>
    </div>
    <p v-if="error" class="text-danger text-sm">
      {{ error }}
    </p>
  </div>
</template>
