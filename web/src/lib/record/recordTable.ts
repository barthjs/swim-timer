import type { RecordTableResponse, RecordTableUpdateRequest } from '@/api/types/recordTable';
import type { FormRules } from '@/lib/form/validation';

/** And RECORD_TABLE_ABBREVIATION_MAX are the maximum lengths the server accepts. */
export const RECORD_TABLE_NAME_MAX = 100;
export const RECORD_TABLE_ABBREVIATION_MAX = 10;

/** The checks of the fields of a record table, as the server runs them. */
export const RECORD_TABLE_RULES: FormRules<RecordTableUpdateRequest> = {
  name: {
    required: 'required',
    max: RECORD_TABLE_NAME_MAX,
  },
  abbreviation: {
    required: 'required',
    max: RECORD_TABLE_ABBREVIATION_MAX,
  },
};

/** Returns the id of the chosen record table if the list has it, empty otherwise. */
export function chosenRecordTableId(id: string, tables: Pick<RecordTableResponse, 'id'>[]): string {
  return tables.some((table) => table.id === id) ? id : '';
}
