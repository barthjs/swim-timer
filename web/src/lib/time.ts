/**
 * formatSwimtime formats a swimming time in milliseconds as `mm:ss,cc`, or as `hh:mm:ss,cc` from one hour on.
 * A time of 0 or less returns an empty string.
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

/**
 * formatDuration formats a duration in whole seconds as `h:mm:ss`, with hours not wrapping at a day.
 * A negative duration counts as 0.
 */
export function formatDuration(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  const hh = Math.floor(total / 3600);
  const mm = Math.floor((total % 3600) / 60);
  const ss = total % 60;

  return `${hh}:${mm.toString().padStart(2, '0')}:${ss.toString().padStart(2, '0')}`;
}
