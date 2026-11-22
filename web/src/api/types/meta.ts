import type { ThemeSettings } from '@/api/types/config';

export type MetaResponse = {
  name: string;
  version: string;
  language: string;
  theme: ThemeSettings;
};
