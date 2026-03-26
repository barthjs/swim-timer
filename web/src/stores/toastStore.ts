import { defineStore } from 'pinia';
import { ref } from 'vue';

import { type Toast, ToastPosition, ToastType } from '@/types/toasts';

export const useToastStore = defineStore('toast', () => {
  const toasts = ref<Toast[]>([]);

  const add = (
    message: string,
    type: ToastType = ToastType.Info,
    timeout: number = 3000,
    position: ToastPosition = ToastPosition.TopCenter,
    permanent: boolean = false,
  ) => {
    const id = Math.random().toString(36).substring(2, 9);
    const toast: Toast = { id, message, type, timeout, position, permanent };
    toasts.value.push(toast);

    if (!permanent && timeout > 0) {
      setTimeout(() => {
        remove(id);
      }, timeout);
    }
  };

  const remove = (id: string) => {
    toasts.value = toasts.value.filter((t) => t.id !== id);
  };

  const success = (
    message: string,
    timeout?: number,
    position?: ToastPosition,
    permanent?: boolean,
  ) => {
    add(message, ToastType.Success, timeout, position, permanent);
  };

  const error = (
    message: string,
    timeout?: number,
    position?: ToastPosition,
    permanent?: boolean,
  ) => {
    add(message, ToastType.Error, timeout, position, permanent);
  };

  const warning = (
    message: string,
    timeout?: number,
    position?: ToastPosition,
    permanent?: boolean,
  ) => {
    add(message, ToastType.Warning, timeout, position, permanent);
  };

  const info = (
    message: string,
    timeout?: number,
    position?: ToastPosition,
    permanent?: boolean,
  ) => {
    add(message, ToastType.Info, timeout, position, permanent);
  };

  return {
    toasts,
    add,
    remove,
    success,
    error,
    warning,
    info,
  };
});
