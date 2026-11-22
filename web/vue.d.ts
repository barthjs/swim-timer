import type { AriaAttributes, HTMLAttributes } from 'vue';

// Attributes fall through to the root element of a component, so every component accepts ARIA and data attributes
// and the global attributes and listeners templates pass on, while strictTemplates still rejects unknown props.
// Templates name data attributes of components in camel case, and native elements accept data attributes as well.
declare module 'vue' {
  interface AllowedComponentProps
    extends
      AriaAttributes,
      Pick<
        HTMLAttributes,
        'id' | 'tabindex' | 'inputmode' | 'onClick' | 'onFocus' | 'onInput' | 'onKeydown'
      > {
    [dataAttribute: `data${Capitalize<string>}`]: unknown;
  }
}

declare module '@vue/runtime-dom' {
  interface HTMLAttributes {
    [dataAttribute: `data-${string}`]: unknown;
  }
}
