import { describe, expect, it } from 'vitest';

import { formatName, fullName } from '@/lib/names';

describe('formatName', () => {
  it.each([
    {
      name: 'keeps a mixed-case last name with lowercase particles',
      first: 'first second',
      last: 'van der Last',
      expected: 'F. van der Last',
    },
    {
      name: 'capitalizes every part of an uppercase last name',
      first: 'First',
      last: 'VAN DER LAST',
      expected: 'F. Van Der Last',
    },
    {
      name: 'capitalizes after hyphens and keeps accented letters',
      first: 'First',
      last: 'ÉLAN-ÑAME',
      expected: 'F. Élan-Ñame',
    },
    {
      name: 'capitalizes after apostrophes',
      first: 'First',
      last: "O'LAST",
      expected: "F. O'Last",
    },
    {
      name: 'keeps a mixed-case hyphenated last name',
      first: 'First',
      last: 'Last-Name',
      expected: 'F. Last-Name',
    },
    {
      name: 'keeps inner capitals of a mixed-case last name',
      first: 'First',
      last: 'McLast',
      expected: 'F. McLast',
    },
    {
      name: 'keeps a lowercase last name',
      first: 'First',
      last: 'de last',
      expected: 'F. de last',
    },
  ])('$name', ({ first, last, expected }) => {
    const name = formatName(first, last);

    expect(name).toBe(expected);
  });

  it('collapses surrounding and repeated whitespace', () => {
    const name = formatName('  first ', '  DE   LAST  ');

    expect(name).toBe('F. De Last');
  });

  it.each([
    {
      part: 'first name',
      first: '',
      last: 'ÉLAN',
      expected: 'Élan',
    },
    {
      part: 'last name',
      first: 'First',
      last: '',
      expected: 'F.',
    },
  ])('skips an empty $part', ({ first, last, expected }) => {
    const name = formatName(first, last);

    expect(name).toBe(expected);
  });

  it('keeps a last name without letters', () => {
    const name = formatName('', '123');

    expect(name).toBe('123');
  });

  it('returns an empty string without any name', () => {
    const name = formatName(' ', ' ');

    expect(name).toBe('');
  });
});

describe('fullName', () => {
  it('joins first and last name', () => {
    const name = fullName(' First ', 'van der Last');

    expect(name).toBe('First van der Last');
  });

  it.each([
    {
      case: 'an empty first name',
      first: '',
      last: 'Last',
      expected: 'Last',
    },
    {
      case: 'two empty names',
      first: '',
      last: ' ',
      expected: '',
    },
  ])('leaves out $case', ({ first, last, expected }) => {
    const name = fullName(first, last);

    expect(name).toBe(expected);
  });
});
