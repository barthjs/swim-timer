import { faker } from '@faker-js/faker';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useUrlSection } from '@/composables/ui/useUrlSection';
import { fakeRouter, settleRouter } from '@/test/router';

let fake = fakeRouter();

vi.mock('vue-router', () => ({
  useRoute: () => fake.route,
  useRouter: () => fake.router,
}));

const SECTIONS = ['first', 'second', 'third'] as const;

describe('useUrlSection', () => {
  beforeEach(() => {
    fake = fakeRouter();
  });

  it('reads the section the URL names', () => {
    fake = fakeRouter({
      section: 'second',
    });

    const section = useUrlSection(SECTIONS, 'first');

    expect(section.value).toBe('second');
  });

  it('falls back to the default without a section in the URL', () => {
    const section = useUrlSection(SECTIONS, 'first');

    expect(section.value).toBe('first');
  });

  it('falls back to the default for an unknown section', () => {
    fake = fakeRouter({
      section: faker.lorem.word(),
    });

    const section = useUrlSection(SECTIONS, 'first');

    expect(section.value).toBe('first');
  });

  it('writes a section to the URL and drops the rest of the query', async () => {
    fake = fakeRouter({
      page: faker.string.numeric(2),
    });
    const section = useUrlSection(SECTIONS, 'first');

    section.value = 'third';
    await settleRouter();

    expect(fake.route.query).toEqual({
      section: 'third',
    });
  });

  it('removes the section from the URL for the default', async () => {
    fake = fakeRouter({
      section: 'second',
    });
    const section = useUrlSection(SECTIONS, 'first');

    section.value = 'first';
    await settleRouter();

    expect(fake.route.query).toEqual({});
  });

  it('keeps the shared keys of the query on a switch', async () => {
    const search = faker.lorem.word();
    fake = fakeRouter({
      q: search,
      page: faker.string.numeric(2),
    });
    const section = useUrlSection(SECTIONS, 'first', ['q']);

    section.value = 'third';
    await settleRouter();

    expect(fake.route.query).toEqual({
      q: search,
      section: 'third',
    });
  });

  it('keeps the shared keys of the query on a switch to the default', async () => {
    const search = faker.lorem.word();
    fake = fakeRouter({
      q: search,
      section: 'second',
      page: faker.string.numeric(2),
    });
    const section = useUrlSection(SECTIONS, 'first', ['q']);

    section.value = 'first';
    await settleRouter();

    expect(fake.route.query).toEqual({
      q: search,
    });
  });

  it('keeps the URL when the open section is chosen again', () => {
    fake = fakeRouter({
      section: 'second',
      page: faker.string.numeric(2),
    });
    const section = useUrlSection(SECTIONS, 'first');

    section.value = 'second';

    expect(fake.router.replace).not.toHaveBeenCalled();
  });
});
