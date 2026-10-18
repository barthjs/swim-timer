import type { NuxtUIOptions } from '@nuxt/ui/vite';

/**
 * Solid buttons, badges, and alerts in a semantic color get white text in both color modes,
 * see --color-on-accent in theme.css. The neutral color keeps Nuxt UI's inverted text,
 * since its solid background is light in dark mode.
 */
const onAccentVariants = (
  ['primary', 'secondary', 'success', 'info', 'warning', 'error'] as const
).map((color) => ({
  color,
  variant: 'solid' as const,
  class: 'text-on-accent',
}));

// The options of the Nuxt UI plugin, kept apart from the build setup in vite.config.ts.
export default {
  // Only Nuxt UI components resolve without an import. The components of the app
  // are imported explicitly, so every file shows where its components come from.
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
      compoundVariants: onAccentVariants,
    },
    badge: {
      compoundVariants: onAccentVariants,
    },
    alert: {
      compoundVariants: onAccentVariants,
    },
  },
} satisfies NuxtUIOptions;
