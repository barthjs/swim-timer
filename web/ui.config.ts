import type { NuxtUIOptions } from '@nuxt/ui/vite';

/**
 * Solid buttons, badges, and alerts in a semantic color get the text of --color-on-accent in theme.css,
 * white in light mode and dark in dark mode. The neutral color keeps Nuxt UI's inverted text,
 * since its solid background is light in dark mode.
 */
const onAccentVariants = (
  ['primary', 'secondary', 'success', 'info', 'warning', 'error'] as const
).map((color) => ({
  color,
  variant: 'solid' as const,
  class: 'text-on-accent',
}));

/** The semantic colors with the class that sets the color of a text or an icon. */
const TINTED_COLORS = [
  ['primary', 'text-primary'],
  ['secondary', 'text-secondary'],
  ['success', 'text-success'],
  ['info', 'text-info'],
  ['warning', 'text-warning'],
  ['error', 'text-error'],
] as const;

/**
 * Returns the compound variants of the soft and subtle variants of a component in every semantic color.
 * They keep the color in the tinted background, the ring, and the icon and set the text in the text colors,
 * since text in the color of its tinted background stays below the contrast of WCAG AA.
 */
function tintedVariants<S extends string>(slots: (icon: string) => Record<S, string>) {
  return TINTED_COLORS.flatMap(([color, icon]) =>
    (['soft', 'subtle'] as const).map((variant) => ({
      color,
      variant,
      class: slots(icon),
    })),
  );
}

/**
 * Text fields get the height of a button of the same size. Some browsers keep an input at least as tall
 * as the normal line height of the font, which is taller than the line height of the fields for Poppins.
 */
const fieldHeights = {
  variants: {
    size: {
      xs: {
        base: 'h-6',
      },
      sm: {
        base: 'h-7',
      },
      md: {
        base: 'h-8',
      },
      lg: {
        base: 'h-9',
      },
      xl: {
        base: 'h-10',
      },
    },
  },
};

/**
 * The heights of fieldHeights for the number field. Its sizes are classes of the base slot
 * instead of a class per slot, so an object would end up in the class as text.
 */
const numberFieldHeights = {
  variants: {
    size: {
      xs: 'h-6',
      sm: 'h-7',
      md: 'h-8',
      lg: 'h-9',
      xl: 'h-10',
    },
  },
};

// The options of the Nuxt UI plugin, kept apart from the build setup in vite.config.ts
export default {
  // Only Nuxt UI components resolve without an import. The components of the app are imported explicitly,
  // so every file shows where its components come from.
  components: {
    dirs: [],
  },
  icon: {
    clientBundle: {
      scan: true,
    },
  },
  ui: {
    colors: {
      primary: 'brand',
    },
    button: {
      compoundVariants: [
        ...onAccentVariants,
        ...tintedVariants((icon) => ({
          base: 'text-highlighted',
          leadingIcon: icon,
        })),
      ],
    },
    badge: {
      compoundVariants: [
        ...onAccentVariants,
        ...tintedVariants((icon) => ({
          base: 'text-highlighted',
          leadingIcon: icon,
        })),
      ],
    },
    alert: {
      compoundVariants: [
        ...onAccentVariants,
        ...tintedVariants((icon) => ({
          root: 'text-default',
          icon,
          title: 'text-highlighted',
          description: 'text-toned opacity-100',
        })),
      ],
    },
    // A selected pill takes the inverted surface. A selected link or menu entry stands out by its text,
    // and only its indicator or icon takes the primary color.
    tabs: {
      compoundVariants: [
        {
          color: 'primary',
          variant: 'pill',
          class: {
            indicator: 'bg-inverted',
            trigger: 'data-[state=active]:text-inverted',
          },
        },
        {
          color: 'primary',
          variant: 'link',
          class: {
            trigger: 'data-[state=active]:text-highlighted',
          },
        },
      ],
    },
    navigationMenu: {
      compoundVariants: [
        {
          color: 'primary',
          variant: 'pill',
          active: true,
          class: {
            link: 'text-highlighted',
          },
        },
      ],
    },
    // A field is a named group, so its parts can react to the pointer or the focus on the field
    formField: {
      slots: {
        root: 'group/field',
      },
    },
    input: fieldHeights,
    inputMenu: fieldHeights,
    inputNumber: numberFieldHeights,
    tooltip: {
      slots: {
        content: 'h-auto max-w-80',
        text: 'whitespace-normal',
      },
    },
  },
} satisfies NuxtUIOptions;
