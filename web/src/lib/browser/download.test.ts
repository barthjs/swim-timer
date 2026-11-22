import { afterEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { downloadBlob, downloadJSON } from '@/lib/browser/download';

/** Replaces the link and the object URLs the download uses and returns them. */
function stubDownload() {
  const url = `blob:${faker.string.uuid()}`;
  const link = {
    href: '',
    download: '',
    click: vi.fn<() => void>(),
  };
  vi.stubGlobal('document', {
    createElement: vi.fn<() => typeof link>(() => link),
  });
  const saved: Blob[] = [];
  const createObjectURL = vi.spyOn(URL, 'createObjectURL').mockImplementation((value) => {
    saved.push(value as Blob);
    return url;
  });
  const revokeObjectURL = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
  return {
    url,
    link,
    saved,
    createObjectURL,
    revokeObjectURL,
  };
}

describe('downloadBlob', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('downloads the blob as a file and releases the URL', () => {
    const fileName = faker.system.commonFileName('csv');
    const blob = new Blob([faker.lorem.sentence()]);
    const download = stubDownload();

    downloadBlob(fileName, blob);

    expect(download.createObjectURL).toHaveBeenCalledWith(blob);
    expect(download.link.href).toBe(download.url);
    expect(download.link.download).toBe(fileName);
    expect(download.link.click).toHaveBeenCalledOnce();
    expect(download.revokeObjectURL).toHaveBeenCalledWith(download.url);
  });
});

describe('downloadJSON', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('downloads the data as a pretty-printed JSON file', async () => {
    const fileName = faker.system.commonFileName('json');
    const name = faker.lorem.word();
    const download = stubDownload();

    downloadJSON(fileName, {
      name,
      items: {},
    });

    expect(download.link.download).toBe(fileName);
    expect(download.saved).toHaveLength(1);
    const saved = download.saved[0]!;
    expect(saved.type).toBe('application/json');
    expect(await saved.text()).toBe(`{\n  "name": "${name}",\n  "items": {}\n}`);
  });
});
