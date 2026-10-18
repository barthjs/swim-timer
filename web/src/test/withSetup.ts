import { createApp } from 'vue';

/**
 * withSetup runs a composable inside a mounted component, so lifecycle hooks such as onMounted and onUnmounted
 * fire as in the app. It returns the composable's result and a function that unmounts the component.
 * It needs a DOM and is meant for .dom.test.ts files.
 */
export function withSetup<T>(composable: () => T): { result: T; unmount: () => void } {
  let result: T | undefined;

  const app = createApp({
    setup() {
      result = composable();
      return () => null;
    },
  });
  app.mount(document.createElement('div'));

  return {
    result: result as T,
    unmount: () => app.unmount(),
  };
}
