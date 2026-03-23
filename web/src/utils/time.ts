/**
 * Formats the raw milliseconds time into a readable string
 *
 * @param {number} ms - The swimming time expressed as milliseconds.
 * @return {string} The formatted swimming time as a string in the format `mm:ss,cc` or `hh:mm:ss,cc`. Returns an empty string when the time is 0.
 */
export function formatSwimtime(ms: number): string {
  if (ms <= 0) return '';

  const totalSeconds = Math.floor(ms / 1000);
  const hundredths = Math.floor((ms % 1000) / 10);
  const hh = Math.floor(totalSeconds / 3600);
  const mm = Math.floor((totalSeconds % 3600) / 60);
  const ss = totalSeconds % 60;

  const mmStr = mm.toString().padStart(2, '0');
  const ssStr = ss.toString().padStart(2, '0');
  const ccStr = hundredths.toString().padStart(2, '0');

  if (hh === 0) {
    return `${mmStr}:${ssStr},${ccStr}`;
  }
  const hhStr = hh.toString().padStart(2, '0');
  return `${hhStr}:${mmStr}:${ssStr},${ccStr}`;
}
