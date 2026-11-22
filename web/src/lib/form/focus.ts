/** Matches the controls of a form that take a typed or chosen value. */
const FIELD_SELECTOR = [
  'input:not([type="hidden"], [type="checkbox"], [type="radio"], [type="file"])',
  'textarea',
  'select',
  '[role="combobox"]',
].join(', ');

/**
 * Focuses the first enabled field inside root and reports whether it did.
 * It leaves the focus alone without a mouse, since a touch screen would open its keyboard over the form.
 */
export function focusFirstField(root: Element | null | undefined): boolean {
  if (!root || !window.matchMedia('(pointer: fine)').matches) return false;

  const field = Array.from(root.querySelectorAll<HTMLElement>(FIELD_SELECTOR)).find(isEditable);
  if (!field) return false;

  field.focus();
  return true;
}

/** Focuses the first field of a dialog that opens, in place of its first button. */
export function focusFieldOnOpen(event: Event): void {
  if (!(event.target instanceof Element)) return;
  if (focusFirstField(event.target)) event.preventDefault();
}

/** Reports whether a field takes input. */
function isEditable(field: HTMLElement): boolean {
  return !field.matches(':disabled, [readonly], [aria-disabled="true"]');
}
