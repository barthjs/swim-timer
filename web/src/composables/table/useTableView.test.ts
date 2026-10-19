import { nextTick, ref } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { fakeRouter } from '@/test/router';
import { useTableView } from '@/composables/table/useTableView';

let fake = fakeRouter();

vi.mock('vue-router', () => ({
  useRoute: () => fake.route,
  useRouter: () => fake.router,
}));

type Row = { name: string; number: number };

const rows: Row[] = Array.from(
  {
    length: 30,
  },
  (_, index) => ({
    name: `row ${String(index).padStart(2, '0')}`,
    number: 30 - index,
  }),
);

/** view creates a table view on the rows, sortable by every field. */
function view(source: Row[] = rows, resetOn?: () => unknown) {
  return useTableView(source, {
    sortable: ['name', 'number'],
    value: (row, id) => row[id as keyof Row],
    resetOn,
  });
}

describe('useTableView', () => {
  beforeEach(() => {
    fake = fakeRouter();
  });

  describe('sort', () => {
    it('reads the sort from the URL', () => {
      fake = fakeRouter({
        sort: 'number',
      });

      const table = view();

      expect(table.sort.value).toEqual({
        id: 'number',
        desc: false,
      });
    });

    it('writes the sort to the URL and returns to the first page', () => {
      fake = fakeRouter({
        page: '2',
        tab: 'items',
      });
      const table = view();

      table.sort.value = {
        id: 'name',
        desc: true,
      };

      expect(fake.route.query).toEqual({
        tab: 'items',
        sort: '-name',
      });
    });

    it('keeps the sort in memory for a view outside the URL', () => {
      fake = fakeRouter({
        tab: 'items',
        sort: 'number',
      });
      const table = useTableView(rows, {
        sortable: ['name', 'number'],
        value: (row, id) => row[id as keyof Row],
        inUrl: false,
      });

      table.sort.value = {
        id: 'name',
        desc: true,
      };

      expect(table.sort.value).toEqual({
        id: 'name',
        desc: true,
      });
      expect(fake.route.query).toEqual({
        tab: 'items',
        sort: 'number',
      });
    });

    it('removes the sort from the URL', () => {
      fake = fakeRouter({
        tab: 'items',
        sort: '-name',
      });
      const table = view();

      table.sort.value = null;

      expect(fake.route.query).toEqual({
        tab: 'items',
      });
    });
  });

  describe('sorting', () => {
    it('returns the sort as the sorting model of a table', () => {
      fake = fakeRouter({
        sort: 'name',
      });

      const table = view();

      expect(table.sorting.value).toEqual([
        {
          id: 'name',
          desc: false,
        },
      ]);
    });

    it('changes the sort from the sorting model of a table', () => {
      fake = fakeRouter({
        sort: 'name',
      });
      const table = view();

      table.sorting.value = [];

      expect(table.sort.value).toBeNull();
      expect(fake.route.query).toEqual({});
    });
  });

  describe('pageSize', () => {
    it('writes the page size to the URL and returns to the first page', () => {
      fake = fakeRouter({
        tab: 'items',
        sort: '-name',
      });
      const table = view();
      table.page.value = 2;

      table.pageSize.value = 50;

      expect(fake.route.query).toEqual({
        tab: 'items',
        sort: '-name',
        perPage: '50',
      });
    });

    it('removes the default page size from the URL', () => {
      fake = fakeRouter({
        tab: 'items',
        perPage: '50',
      });
      const table = view();

      table.pageSize.value = 25;

      expect(fake.route.query).toEqual({
        tab: 'items',
      });
    });
  });

  describe('page', () => {
    it('clamps a page beyond the last one', () => {
      fake = fakeRouter({
        page: '99',
      });

      const table = view();

      expect(table.page.value).toBe(2);
    });

    it('returns to the first page when the reset source changes', async () => {
      fake = fakeRouter({
        page: '2',
      });
      const filter = ref('');
      const table = view(rows, () => filter.value);

      filter.value = 'changed';
      await nextTick();

      expect(table.page.value).toBe(1);
    });
  });

  describe('visible', () => {
    it('returns the first page in the given order by default', () => {
      const table = view();

      expect(table.sort.value).toBeNull();
      expect(table.page.value).toBe(1);
      expect(table.pageSize.value).toBe(25);
      expect(table.total.value).toBe(30);
      expect(table.visible.value).toHaveLength(25);
      expect(table.visible.value[0]?.name).toBe('row 00');
    });

    it('returns the page of the URL in the sort and page size of the URL', () => {
      fake = fakeRouter({
        sort: 'number',
        page: '2',
        perPage: '10',
      });

      const table = view();

      expect(table.visible.value.map((row) => row.number)).toEqual([
        11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
      ]);
    });
  });

  describe('goTo', () => {
    it('jumps to the page of a row', () => {
      fake = fakeRouter({
        perPage: '10',
      });
      const table = view();

      const found = table.goTo((row) => row.name === 'row 23');

      expect(found).toBe(true);
      expect(table.page.value).toBe(3);
    });

    it('keeps the page for an unknown row', () => {
      fake = fakeRouter({
        perPage: '10',
      });
      const table = view();
      table.goTo((row) => row.name === 'row 23');

      const found = table.goTo((row) => row.name === 'missing');

      expect(found).toBe(false);
      expect(table.page.value).toBe(3);
    });
  });
});
