import { afterEach, describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import { syncThemeColor } from '@/lib/browser/themeColor';

/** Returns a random opaque color as browsers compute it. */
function fakeColor(): string {
  const channel = () => faker.number.int({ min: 0, max: 255 });
  return `rgb(${channel()}, ${channel()}, ${channel()})`;
}

/** Adds a theme color to the head of the document, optionally for one color scheme. */
function addThemeColor(color: string, media?: string): void {
  const meta = document.createElement('meta');
  meta.name = 'theme-color';
  meta.content = color;
  if (media) {
    meta.media = media;
  }
  document.head.append(meta);
}

/** Returns the theme colors in the head of the document. */
function themeColors(): HTMLMetaElement[] {
  return [...document.head.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]')];
}

/** Waits until the mutation observers have run. */
async function nextMutations(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve));
}

describe('syncThemeColor', () => {
  let stop: () => void = () => {};

  afterEach(() => {
    stop();
    document.head.innerHTML = '';
    document.documentElement.removeAttribute('class');
    document.body.removeAttribute('style');
  });

  it('sets the theme color to the background of the page', () => {
    const color = fakeColor();
    document.body.style.backgroundColor = color;

    stop = syncThemeColor();

    expect(themeColors().map((meta) => meta.content)).toEqual([color]);
  });

  it('replaces the theme colors per color scheme with one for every scheme', () => {
    addThemeColor(fakeColor(), '(prefers-color-scheme: light)');
    addThemeColor(fakeColor(), '(prefers-color-scheme: dark)');
    const color = fakeColor();
    document.body.style.backgroundColor = color;

    stop = syncThemeColor();

    const metas = themeColors();
    expect(metas).toHaveLength(1);
    expect(metas[0]?.content).toBe(color);
    expect(metas[0]?.hasAttribute('media')).toBe(false);
  });

  it('follows a change of the class on the root element', async () => {
    document.body.style.backgroundColor = fakeColor();
    stop = syncThemeColor();
    const color = fakeColor();
    document.body.style.backgroundColor = color;

    document.documentElement.classList.add('dark');
    await nextMutations();

    expect(themeColors()[0]?.content).toBe(color);
  });

  it('follows a change of the style on the root element', async () => {
    document.body.style.backgroundColor = fakeColor();
    stop = syncThemeColor();
    const color = fakeColor();
    document.body.style.backgroundColor = color;

    document.documentElement.style.setProperty('--ui-bg', color);
    await nextMutations();

    expect(themeColors()[0]?.content).toBe(color);
  });

  it('keeps the theme color while the page has no background', () => {
    const color = fakeColor();
    addThemeColor(color);

    stop = syncThemeColor();

    expect(themeColors().map((meta) => meta.content)).toEqual([color]);
  });

  it('stops following changes once stopped', async () => {
    const color = fakeColor();
    document.body.style.backgroundColor = color;
    syncThemeColor()();
    document.body.style.backgroundColor = fakeColor();

    document.documentElement.classList.add('dark');
    await nextMutations();

    expect(themeColors()[0]?.content).toBe(color);
  });
});
