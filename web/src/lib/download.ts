/** downloadJSON lets the browser save data as a pretty-printed JSON file with the given name. */
export function downloadJSON(fileName: string, data: unknown): void {
  const blob = new Blob([JSON.stringify(data, null, 2)], {
    type: 'application/json',
  });
  const url = URL.createObjectURL(blob);

  const link = document.createElement('a');
  link.href = url;
  link.download = fileName;
  link.click();

  // The click starts the download synchronously, so the URL is no longer needed afterward
  URL.revokeObjectURL(url);
}
