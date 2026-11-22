import { describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { useRecordTexts } from '@/composables/record/useRecordTexts';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, number>) =>
      values ? `${key}:${Object.values(values).join('-')}` : key,
    te: (key: string) =>
      ['pages.records.columnHints.time', 'pages.records.strokes.FR'].includes(key),
  }),
}));

describe('useRecordTexts', () => {
  describe('columnLabel', () => {
    it('names a column by the label of its field', () => {
      const { columnLabel } = useRecordTexts();

      const label = columnLabel('time');

      expect(label).toBe('fields.time');
    });
  });

  describe('columnHint', () => {
    it('describes a column with a hint', () => {
      const { columnHint } = useRecordTexts();

      const hint = columnHint('time');

      expect(hint).toBe('pages.records.columnHints.time');
    });

    it('returns nothing for a column without a hint', () => {
      const { columnHint } = useRecordTexts();

      const hint = columnHint('name');

      expect(hint).toBe('');
    });
  });

  describe('valueLabel', () => {
    it('names a known value of a column', () => {
      const { valueLabel } = useRecordTexts();

      const label = valueLabel('stroke', 'FR');

      expect(label).toBe('pages.records.strokes.FR');
    });

    it('returns nothing for an unknown value or a column without names', () => {
      const { valueLabel } = useRecordTexts();

      expect(valueLabel('stroke', faker.string.alpha(4))).toBe('');
      expect(valueLabel('name', faker.person.lastName())).toBe('');
    });
  });

  describe('ageLabel', () => {
    it('describes a range, a single age, and open bounds', () => {
      const { ageLabel } = useRecordTexts();
      const from = faker.number.int({
        min: 1,
        max: 50,
      });
      const to =
        from +
        faker.number.int({
          min: 1,
          max: 50,
        });

      expect(
        ageLabel({
          ageFrom: from,
          ageTo: to,
        }),
      ).toBe(`pages.records.ageRange:${from}-${to}`);
      expect(
        ageLabel({
          ageFrom: from,
          ageTo: from,
        }),
      ).toBe(`pages.records.age:${from}`);
      expect(
        ageLabel({
          ageFrom: from,
          ageTo: 0,
        }),
      ).toBe(`pages.records.ageFrom:${from}`);
      expect(
        ageLabel({
          ageFrom: 0,
          ageTo: to,
        }),
      ).toBe(`pages.records.ageTo:${to}`);
    });

    it('returns nothing for a record of every age', () => {
      const { ageLabel } = useRecordTexts();

      const label = ageLabel({
        ageFrom: 0,
        ageTo: 0,
      });

      expect(label).toBe('');
    });
  });

  describe('disciplineLabel', () => {
    it('describes an individual record by its distance and stroke', () => {
      const { disciplineLabel } = useRecordTexts();
      const distance = faker.helpers.arrayElement([50, 100, 200]);
      const stroke = faker.string.alpha(2).toUpperCase();

      const label = disciplineLabel({
        relayCount: 1,
        distance,
        stroke,
      });

      expect(label).toBe(`${distance} m ${stroke}`);
    });

    it('describes a relay by its legs', () => {
      const { disciplineLabel } = useRecordTexts();
      const relayCount = faker.number.int({
        min: 2,
        max: 8,
      });
      const distance = faker.helpers.arrayElement([50, 100, 200]);
      const stroke = faker.string.alpha(2).toUpperCase();

      const label = disciplineLabel({
        relayCount,
        distance,
        stroke,
      });

      expect(label).toBe(`${relayCount}×${distance} m ${stroke}`);
    });
  });
});
