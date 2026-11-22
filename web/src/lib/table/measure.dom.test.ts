import { afterEach, describe, expect, it, vi } from 'vitest';

import { measureColumnWidth, measureTableColumns, rootFontSize } from '@/lib/table/measure';

/**
 * Renders a table with a header row and the given body rows.
 * happy-dom does no layout, so every cell reports a width of ten pixels per character of its text.
 */
function renderTable(header: string[], rows: string[][]): HTMLTableElement {
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (
    this: HTMLElement,
  ) {
    return new DOMRect(0, 0, this.textContent.length * 10, 20);
  });

  const table = document.createElement('table');
  table.innerHTML = `
    <thead><tr>${header.map((label) => `<th>${label}</th>`).join('')}</tr></thead>
    <tbody>${rows.map((row) => `<tr>${row.map((cell) => `<td>${cell}</td>`).join('')}</tr>`).join('')}</tbody>
  `;
  document.body.append(table);
  return table;
}

describe('rootFontSize', () => {
  afterEach(() => {
    document.documentElement.style.fontSize = '';
  });

  it('returns the size of 1rem in pixels', () => {
    document.documentElement.style.fontSize = '20px';

    const size = rootFontSize();

    expect(size).toBe(20);
  });
});

describe('measureTableColumns', () => {
  afterEach(() => {
    document.body.innerHTML = '';
    vi.restoreAllMocks();
  });

  it('returns the widest cell of every column, header included', () => {
    const table = renderTable(
      ['Name', 'Email'],
      [
        ['Ann', 'a@example.com'],
        ['Christopher', 'c@example.com'],
      ],
    );

    const widths = measureTableColumns(table);

    expect(widths).toEqual([110, 130]);
  });

  it('ignores a row whose cell spans the whole table', () => {
    const table = renderTable(['Id', 'Name'], []);
    table.querySelector('tbody')!.innerHTML =
      '<tr><td colspan="2">Nothing found in this list</td></tr>';

    const widths = measureTableColumns(table);

    expect(widths).toEqual([20, 40]);
  });

  it('returns nothing for a table without a header', () => {
    const table = document.createElement('table');

    const widths = measureTableColumns(table);

    expect(widths).toEqual([]);
  });

  it('removes its helper table again', () => {
    const table = renderTable(['Name'], [['Ann']]);

    measureTableColumns(table);

    expect(document.querySelectorAll('table')).toHaveLength(1);
  });
});

describe('measureColumnWidth', () => {
  afterEach(() => {
    document.body.innerHTML = '';
    vi.restoreAllMocks();
  });

  it('returns the widest cell of the column, header included', () => {
    const table = renderTable(
      ['Name', 'Email'],
      [
        ['Ann', 'a@example.com'],
        ['Christopher', 'c@example.com'],
      ],
    );
    const th = table.querySelectorAll('th');

    const widths = [measureColumnWidth(th[0]!), measureColumnWidth(th[1]!)];

    expect(widths).toEqual([110, 130]);
  });

  it('ignores a row whose cell spans the whole table', () => {
    const table = renderTable(['Id'], []);
    table.querySelector('tbody')!.innerHTML =
      '<tr><td colspan="3">Nothing found in this list</td></tr>';

    const width = measureColumnWidth(table.querySelector('th')!);

    expect(width).toBe(20);
  });

  it('removes its helper table again', () => {
    const table = renderTable(['Name'], [['Ann']]);

    measureColumnWidth(table.querySelector('th')!);

    expect(document.querySelectorAll('table')).toHaveLength(1);
  });
});
