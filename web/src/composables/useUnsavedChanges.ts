import { onBeforeUnmount, type Ref } from 'vue';
import { onBeforeRouteLeave } from 'vue-router';
import { useI18n } from 'vue-i18n';

export function useUnsavedChanges(isDirty: Ref<boolean> | (() => boolean)) {
  const { t } = useI18n();

  const getDirty = (): boolean => {
    return typeof isDirty === 'function' ? isDirty() : isDirty.value;
  };

  const beforeUnloadHandler = (e: BeforeUnloadEvent) => {
    if (getDirty()) {
      e.preventDefault();
    }
  };

  window.addEventListener('beforeunload', beforeUnloadHandler);

  onBeforeUnmount(() => {
    window.removeEventListener('beforeunload', beforeUnloadHandler);
  });

  onBeforeRouteLeave(() => {
    if (getDirty()) {
      return window.confirm(t('common.unsavedChangesConfirm'));
    }
    return true;
  });
}
