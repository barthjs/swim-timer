import { describe, expect, it } from 'vitest';

import { formatDuration, formatSwimtime } from '@/lib/time';

describe('formatSwimtime', () => {
  it.each([
    {
      ms: 65432,
      expected: '01:05,43',
    },
    {
      ms: 59999,
      expected: '00:59,99',
    },
  ])('returns minutes, seconds, and truncated hundredths for $ms ms', ({ ms, expected }) => {
    const formatted = formatSwimtime(ms);

    expect(formatted).toBe(expected);
  });

  it.each([
    {
      ms: 10,
      expected: '00:00,01',
    },
    {
      ms: 9,
      expected: '00:00,00',
    },
  ])('returns a time below a second with leading zeros for $ms ms', ({ ms, expected }) => {
    const formatted = formatSwimtime(ms);

    expect(formatted).toBe(expected);
  });

  it.each([
    {
      ms: 3723456,
      expected: '01:02:03,45',
    },
    {
      ms: 3600000,
      expected: '01:00:00,00',
    },
  ])('adds hours from one hour on for $ms ms', ({ ms, expected }) => {
    const formatted = formatSwimtime(ms);

    expect(formatted).toBe(expected);
  });

  it('keeps minutes just below one hour', () => {
    const formatted = formatSwimtime(3599990);

    expect(formatted).toBe('59:59,99');
  });

  it.each([
    {
      ms: 0,
    },
    {
      ms: -10,
    },
  ])('returns an empty string for $ms ms', ({ ms }) => {
    const formatted = formatSwimtime(ms);

    expect(formatted).toBe('');
  });
});

describe('formatDuration', () => {
  it.each([
    {
      seconds: 309,
      expected: '0:05:09',
    },
    {
      seconds: 97200,
      expected: '27:00:00',
    },
  ])('returns hours, minutes, and seconds for $seconds s', ({ seconds, expected }) => {
    const formatted = formatDuration(seconds);

    expect(formatted).toBe(expected);
  });

  it.each([
    {
      seconds: -5,
      expected: '0:00:00',
    },
    {
      seconds: 61.9,
      expected: '0:01:01',
    },
  ])('treats $seconds s as whole non-negative seconds', ({ seconds, expected }) => {
    const formatted = formatDuration(seconds);

    expect(formatted).toBe(expected);
  });
});
