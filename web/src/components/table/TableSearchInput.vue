<script setup lang="ts">
import { nextTick, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';

// The search input of a list. A cross on the right and the Escape key clear it,
// and the focus stays in the input, so a new search can be typed right away.
withDefaults(
  defineProps<{
    placeholder?: string;
  }>(),
  {
    placeholder: '',
  },
);

const search = defineModel<string>({
  default: '',
});

const { t } = useI18n();

const input = useTemplateRef<{ inputRef?: HTMLInputElement | null }>('input');

/** focus moves the focus into the input. */
function focus() {
  input.value?.inputRef?.focus();
}

/** clear empties the input and keeps the focus in it. */
function clear() {
  search.value = '';
  void nextTick(focus);
}

/** onEscape clears a filled input. An empty one lets the key through, for example to close a dialog. */
function onEscape(event: KeyboardEvent) {
  if (search.value === '') return;
  event.stopPropagation();
  clear();
}

defineExpose({
  focus,
});
</script>

<template>
  <UInput
    ref="input"
    v-model="search"
    icon="i-lucide-search"
    :placeholder="placeholder"
    :aria-label="placeholder || t('common.search')"
    :ui="{ trailing: 'pe-1' }"
    @keydown.esc="onEscape"
  >
    <template v-if="search !== ''" #trailing>
      <UButton
        icon="i-lucide-x"
        color="neutral"
        variant="link"
        size="sm"
        :aria-label="t('table.clearSearch')"
        @click="clear"
      />
    </template>
  </UInput>
</template>
