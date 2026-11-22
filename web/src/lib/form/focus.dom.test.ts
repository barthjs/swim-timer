import { afterEach, describe, expect, it, vi } from 'vitest';

import { focusFieldOnOpen, focusFirstField } from '@/lib/form/focus';

/** Adds a container with the given markup to the document. */
function render(html: string): HTMLElement {
  const root = document.createElement('div');
  root.innerHTML = html;
  document.body.append(root);
  return root;
}

/** Makes the media query of a fine pointer report the given result. */
function usePointer(fine: boolean): void {
  vi.spyOn(window, 'matchMedia').mockImplementation(
    (query) =>
      ({
        matches: fine && query === '(pointer: fine)',
        media: query,
      }) as MediaQueryList,
  );
}

describe('focusFirstField', () => {
  afterEach(() => {
    document.body.innerHTML = '';
    vi.restoreAllMocks();
  });

  it('focuses the first field', () => {
    usePointer(true);
    const root = render('<button>Close</button><input id="first"><input id="second">');

    const focused = focusFirstField(root);

    expect(focused).toBe(true);
    expect(document.activeElement?.id).toBe('first');
  });

  it('focuses a combobox as a field', () => {
    usePointer(true);
    const root = render('<button role="combobox" id="combobox"></button><input>');

    const focused = focusFirstField(root);

    expect(focused).toBe(true);
    expect(document.activeElement?.id).toBe('combobox');
  });

  it('skips hidden inputs, checkboxes, radios, and file inputs', () => {
    usePointer(true);
    const root = render(
      '<input type="hidden"><input type="checkbox"><input type="radio"><input type="file"><input id="text">',
    );

    focusFirstField(root);

    expect(document.activeElement?.id).toBe('text');
  });

  it('skips disabled and read-only fields', () => {
    usePointer(true);
    const root = render(
      '<input disabled><input readonly><button role="combobox" aria-disabled="true"></button><input id="text">',
    );

    focusFirstField(root);

    expect(document.activeElement?.id).toBe('text');
  });

  it('reports false without a field', () => {
    usePointer(true);
    const root = render('<button>Close</button>');

    const focused = focusFirstField(root);

    expect(focused).toBe(false);
    expect(document.activeElement).toBe(document.body);
  });

  it('reports false without a root', () => {
    usePointer(true);

    const focused = focusFirstField(null);

    expect(focused).toBe(false);
  });

  it('leaves the focus alone without a fine pointer', () => {
    usePointer(false);
    const root = render('<input>');

    const focused = focusFirstField(root);

    expect(focused).toBe(false);
    expect(document.activeElement).toBe(document.body);
  });
});

describe('focusFieldOnOpen', () => {
  afterEach(() => {
    document.body.innerHTML = '';
    vi.restoreAllMocks();
  });

  it('focuses the first field and prevents the default focus', () => {
    usePointer(true);
    const root = render('<button>Close</button><input id="first">');
    const event = new Event('focus.autoFocusOnMount', {
      cancelable: true,
    });
    root.addEventListener(event.type, focusFieldOnOpen);

    root.dispatchEvent(event);

    expect(event.defaultPrevented).toBe(true);
    expect(document.activeElement?.id).toBe('first');
  });

  it('keeps the default focus without a field', () => {
    usePointer(true);
    const root = render('<button>Close</button>');
    const event = new Event('focus.autoFocusOnMount', {
      cancelable: true,
    });
    root.addEventListener(event.type, focusFieldOnOpen);

    root.dispatchEvent(event);

    expect(event.defaultPrevented).toBe(false);
  });
});
