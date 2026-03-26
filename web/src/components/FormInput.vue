<script setup lang="ts">
interface Props {
  label?: string;
  type?: 'text' | 'number' | 'password' | 'email';
  modelValue: string | number;
  error?: string;
  disabled?: boolean;
  required?: boolean;
  placeholder?: string;
  min?: number;
  max?: number;
  class?: string;
  inputClass?: string;
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  disabled: false,
  required: false,
  class: '',
  inputClass: '',
});

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | number): void;
}>();

const handleInput = (event: Event) => {
  const target = event.target as HTMLInputElement;
  let value: string | number;

  if (props.type === 'number') {
    value = target.value ? Number(target.value) : 0;
  } else {
    value = target.value;
  }

  emit('update:modelValue', value);
};
</script>

<template>
  <div class="flex min-w-0 flex-col gap-2" :class="props.class">
    <label v-if="label" class="text-fg text-sm font-medium">
      {{ label }}
      <span v-if="required" class="text-danger">*</span>
    </label>
    <input
      :type="type"
      :value="modelValue"
      :disabled="disabled"
      :placeholder="placeholder"
      :min="min"
      :max="max"
      :required="required"
      @input="handleInput"
      class="border-subtle bg-surface text-fg placeholder-muted focus:border-primary focus:ring-primary disabled:bg-subtle min-w-0 rounded-md border px-3 py-2 focus:ring-1 focus:outline-none disabled:cursor-not-allowed"
      :class="[{ 'border-danger': error }, props.inputClass]"
    />
    <p v-if="error" class="text-danger text-sm">
      {{ error }}
    </p>
  </div>
</template>
