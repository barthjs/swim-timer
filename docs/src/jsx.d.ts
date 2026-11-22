import 'astro/astro-jsx';

declare global {
  namespace JSX {
    // Astro types an element as any to support the JSX of other frameworks, which this site does not use
    type Element = HTMLElement;
  }
}
