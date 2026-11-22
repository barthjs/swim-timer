/**
 * Copies text to the clipboard and reports whether it worked. Browsers offer the Clipboard API only in a secure context,
 * so it falls back to the copy command of the document elsewhere.
 */
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return copyWithCommand(text);
  }
}

/** Copies text through a hidden text field and the copy command of the document. */
function copyWithCommand(text: string): boolean {
  const field = document.createElement('textarea');
  field.value = text;
  field.setAttribute('readonly', '');
  field.style.position = 'absolute';
  field.style.opacity = '0';
  document.body.append(field);
  field.select();
  try {
    // eslint-disable-next-line @typescript-eslint/no-deprecated -- pages without a secure context have no Clipboard API
    return document.execCommand('copy');
  } catch {
    return false;
  } finally {
    field.remove();
  }
}
