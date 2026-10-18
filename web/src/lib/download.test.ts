import { afterEach, describe, expect, it, vi } from 'vitest';

import { downloadJSON } from '@/lib/download';

describe('downloadJSON', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('downloads the data as a pretty-printed JSON file and releases the URL', async () => {
    const link = {
      href: '',
      download: '',
      click: vi.fn<() => void>(),
    };
    vi.stubGlobal('document', {
      createElement: vi.fn<() => typeof link>(() => link),
    });
    let blob: Blob | null = null;
    const createObjectURL = vi.spyOn(URL, 'createObjectURL').mockImplementation((value) => {
      blob = value as Blob;
      return 'blob:file';
    });
    const revokeObjectURL = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});

    downloadJSON('items.json', {
      name: 'items',
      items: {},
    });

    expect(createObjectURL).toHaveBeenCalledOnce();
    expect(link.href).toBe('blob:file');
    expect(link.download).toBe('items.json');
    expect(link.click).toHaveBeenCalledOnce();
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:file');
    expect(blob).not.toBeNull();
    const saved = blob as unknown as Blob;
    expect(saved.type).toBe('application/json');
    expect(await saved.text()).toBe('{\n  "name": "items",\n  "items": {}\n}');
  });
});
