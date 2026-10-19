import { defineCollection } from 'astro:content';
import { z } from 'astro/zod';
import { docsLoader, i18nLoader } from '@astrojs/starlight/loaders';
import { docsSchema, i18nSchema } from '@astrojs/starlight/schema';

export const collections = {
  docs: defineCollection({
    loader: docsLoader(),
    schema: docsSchema(),
  }),
  // Translations of the own components next to the texts of Starlight
  i18n: defineCollection({
    loader: i18nLoader(),
    schema: i18nSchema({
      extend: z
        .object({
          'breadcrumbs.label': z.string(),
          'breadcrumbs.home': z.string(),
          'versionSelect.label': z.string(),
          'versionSelect.development': z.string(),
          'versionSelect.latest': z.string(),
        })
        .partial(),
    }),
  }),
};
