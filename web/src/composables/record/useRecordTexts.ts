import { useI18n } from 'vue-i18n';

import type { RecordResponse } from '@/api/types/record';
import { formatRecordDistance } from '@/lib/record/record';

/** Names the columns of a record and describes the age range and the discipline of a record. */
export function useRecordTexts() {
  const { t, te } = useI18n();

  /** Returns the name of a column of a record by its key. */
  function columnLabel(key: string): string {
    return t(`fields.${key}`);
  }

  /** Describes what a column of a record holds, empty for a column that needs no description. */
  function columnHint(key: string): string {
    const hintKey = `pages.records.columnHints.${key}`;
    return te(hintKey) ? t(hintKey) : '';
  }

  /** Names a fixed or common value of a column of a record, empty for a value without a name. */
  function valueLabel(key: string, value: string): string {
    const groups: Record<string, string> = {
      course: 'courseNames',
      stroke: 'strokes',
      approved: 'approvedNames',
    };
    const group = groups[key];
    const labelKey = `pages.records.${group}.${value}`;
    return group && te(labelKey) ? t(labelKey) : '';
  }

  /** Describes the ages a record applies to, empty for a record of every age. */
  function ageLabel(record: Pick<RecordResponse, 'ageFrom' | 'ageTo'>): string {
    const { ageFrom, ageTo } = record;
    if (ageFrom && ageTo) {
      return ageFrom === ageTo
        ? t('pages.records.age', {
            age: ageFrom,
          })
        : t('pages.records.ageRange', {
            from: ageFrom,
            to: ageTo,
          });
    }
    if (ageFrom) {
      return t('pages.records.ageFrom', {
        age: ageFrom,
      });
    }
    if (ageTo) {
      return t('pages.records.ageTo', {
        age: ageTo,
      });
    }
    return '';
  }

  return {
    columnLabel,
    columnHint,
    valueLabel,
    ageLabel,
    disciplineLabel,
  };
}

/** Describes the distance and the stroke of a record, with the number of legs of a relay. */
function disciplineLabel(
  record: Pick<RecordResponse, 'relayCount' | 'distance' | 'stroke'>,
): string {
  return `${formatRecordDistance(record)} ${record.stroke}`;
}
