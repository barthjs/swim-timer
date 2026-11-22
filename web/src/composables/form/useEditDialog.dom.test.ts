import { nextTick, ref } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { useEditDialog, type UseEditDialogOptions } from '@/composables/form/useEditDialog';
import { fakeDistinct } from '@/test/fake';
import { withSetup } from '@/test/withSetup';

const { confirm, toast } = vi.hoisted(() => ({
  confirm: vi.fn<() => Promise<boolean>>(),
  toast: {
    add: vi.fn<(toast: object) => void>(),
  },
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    te: () => false,
  }),
}));

vi.mock('@nuxt/ui/composables', () => ({
  useToast: () => toast,
}));

vi.mock('@/composables/ui/useConfirm', () => ({
  useConfirm: () => ({
    confirm,
  }),
}));

/** A record of the dialog under test. */
type Row = { id: string; name: string };
/** The form of the dialog under test. */
type State = { name: string };

const [name, changedName] = fakeDistinct(() => faker.lorem.words(2), 2);

const row: Row = {
  id: faker.string.uuid(),
  name,
};

const createdId = faker.string.uuid();

let unmount = () => {};

/** Builds the dialog in a mounted component. */
function setup(options: Partial<UseEditDialogOptions<Row, State, Row>> = {}) {
  const mounted = withSetup(() =>
    useEditDialog<Row, State, Row>({
      initial: (current) => ({
        name: current?.name ?? '',
      }),
      create: async (state) =>
        Promise.resolve({
          id: createdId,
          name: state.name,
        }),
      update: async (current, state) =>
        Promise.resolve({
          ...current,
          name: state.name,
        }),
      ...options,
    }),
  );
  unmount = mounted.unmount;
  return mounted.result;
}

describe('useEditDialog', () => {
  beforeEach(() => {
    confirm.mockReset();
    toast.add.mockClear();
  });

  afterEach(() => {
    unmount();
  });

  describe('openCreate', () => {
    it('opens the dialog with the defaults of a new record', () => {
      const dialog = setup();

      dialog.openCreate();

      expect(dialog.open.value).toBe(true);
      expect(dialog.editing.value).toBeNull();
      expect(dialog.state.name).toBe('');
    });
  });

  describe('openEdit', () => {
    it('opens the dialog with the values of the record as the baseline', () => {
      const dialog = setup();

      dialog.openEdit(row);

      expect(dialog.open.value).toBe(true);
      expect(dialog.editing.value).toEqual(row);
      expect(dialog.state.name).toBe(row.name);
      expect(dialog.isDirty.value).toBe(false);
    });
  });

  describe('save', () => {
    it('creates a new record, closes the dialog, and runs onSaved', async () => {
      const [updated, added] = fakeDistinct(() => faker.lorem.sentence(), 2);
      const onSaved = vi.fn<(result: Row, isEdit: boolean) => void>();
      const dialog = setup({
        onSaved,
        success: (isEdit) => (isEdit ? updated : added),
      });
      dialog.openCreate();
      dialog.state.name = changedName;

      await dialog.save();

      expect(dialog.open.value).toBe(false);
      expect(onSaved).toHaveBeenCalledWith(
        {
          id: createdId,
          name: changedName,
        },
        false,
      );
      expect(toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: added,
        }),
      );
    });

    it('updates the edited record with the changed fields', async () => {
      const [changed, unchanged] = fakeDistinct(() => faker.lorem.word(), 2);
      const update = vi.fn<UseEditDialogOptions<Row, State, Row>['update'] & object>(
        async (current, _state, isFieldDirty) =>
          Promise.resolve({
            ...current,
            name: isFieldDirty('name') ? changed : unchanged,
          }),
      );
      const onSaved = vi.fn<(result: Row, isEdit: boolean) => void>();
      const dialog = setup({
        update,
        onSaved,
      });
      dialog.openEdit(row);
      dialog.state.name = changedName;

      await dialog.save();

      expect(onSaved).toHaveBeenCalledWith(
        {
          id: row.id,
          name: changed,
        },
        true,
      );
    });

    it('edits a record without create', async () => {
      const onSaved = vi.fn<(result: Row, isEdit: boolean) => void>();
      const dialog = setup({
        create: undefined,
        onSaved,
      });
      dialog.openEdit(row);
      dialog.state.name = changedName;

      await dialog.save();

      expect(onSaved).toHaveBeenCalledWith(
        {
          id: row.id,
          name: changedName,
        },
        true,
      );
    });

    it('rejects a new record without create and keeps the dialog open', async () => {
      const onSaved = vi.fn<(result: Row, isEdit: boolean) => void>();
      const dialog = setup({
        create: undefined,
        onSaved,
      });
      dialog.openCreate();

      await dialog.save();

      expect(dialog.open.value).toBe(true);
      expect(onSaved).not.toHaveBeenCalled();
    });
  });

  describe('attemptClose', () => {
    it('keeps the dialog open with unsaved changes until the discard is confirmed', async () => {
      confirm.mockResolvedValue(false);
      const dialog = setup();
      dialog.openEdit(row);
      dialog.state.name = changedName;

      await dialog.attemptClose();

      expect(dialog.open.value).toBe(true);
    });
  });

  it('starts from its record whenever the parent of a dialog component opens it', async () => {
    const open = ref(false);
    const dialog = setup({
      open,
      row: () => row,
    });

    open.value = true;
    await nextTick();

    expect(dialog.editing.value).toEqual(row);
    expect(dialog.state.name).toBe(row.name);
  });
});
