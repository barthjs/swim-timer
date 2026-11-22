/** A section with the distance of its top from the top of the view in pixels. */
export type SectionTop<T> = {
  section: T;
  top: number;
};

/**
 * Returns the section a reader is at: the last one whose top passed the line,
 * the first one before any did, and the last one at the end of the page, which may never reach the line.
 */
export function activeSection<T>(
  tops: SectionTop<T>[],
  line: number,
  atEnd: boolean,
): T | undefined {
  if (atEnd) return tops.at(-1)?.section;
  let active = tops[0]?.section;
  for (const entry of tops) {
    if (entry.top <= line) active = entry.section;
  }
  return active;
}
