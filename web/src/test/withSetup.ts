import { createApp } from 'vue';

/**
 * Runs a composable inside a mounted component of a .dom.test.ts file, so its lifecycle hooks fire,
 * and returns its result and a function that unmounts the component.
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
