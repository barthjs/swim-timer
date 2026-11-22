import { afterEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { copyText } from '@/lib/browser/clipboard';

/** Replaces the Clipboard API with one that writes or fails. */
function stubClipboard(writes: boolean) {
  const writeText = vi.fn<(text: string) => Promise<void>>(async () =>
    writes ? Promise.resolve() : Promise.reject(new Error('not allowed')),
  );
  vi.stubGlobal('navigator', {
    clipboard: {
      writeText,
    },
  });
  return writeText;
}

/** Replaces the copy command of the document with one that returns the result. */
function stubCommand(result: boolean) {
  const execCommand = vi.fn<(command: string) => boolean>(() => result);
  Object.defineProperty(document, 'execCommand', {
    value: execCommand,
    configurable: true,
  });
  return execCommand;
}

describe('copyText', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('copies through the Clipboard API', async () => {
    const writeText = stubClipboard(true);
    const execCommand = stubCommand(true);
    const text = faker.lorem.sentence();

    const copied = await copyText(text);

    expect(copied).toBe(true);
    expect(writeText).toHaveBeenCalledWith(text);
    expect(execCommand).not.toHaveBeenCalled();
  });

  it('falls back to the copy command when the Clipboard API fails', async () => {
    stubClipboard(false);
    const execCommand = stubCommand(true);

    const copied = await copyText(faker.lorem.sentence());

    expect(copied).toBe(true);
    expect(execCommand).toHaveBeenCalledWith('copy');
  });

  it('reports a copy command that copies nothing', async () => {
    stubClipboard(false);
    stubCommand(false);

    const copied = await copyText(faker.lorem.sentence());

    expect(copied).toBe(false);
  });

  it('removes the hidden text field again', async () => {
    stubClipboard(false);
    stubCommand(true);

    await copyText(faker.lorem.sentence());

    expect(document.querySelector('textarea')).toBeNull();
  });
});
