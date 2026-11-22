/** The start of a part of a name: the beginning, or a space, hyphen, or apostrophe before a letter. */
const NAME_PART_START = /(^|[\s'’-])(\p{L})/gu;

/**
 * Writes a competitor as the initial of the first name and the whole last name, capitalizing a last name in capitals.
 */
export function formatName(firstName: string, lastName: string): string {
  // A last name in capitals, as many timing exports write it, is capitalized per part,
  // split at spaces, hyphens, and apostrophes, without recognizing particles.
  const first = collapseWhitespace(firstName);
  const firstInitial = first.length > 0 ? `${first.charAt(0).toUpperCase()}.` : '';

  let last = collapseWhitespace(lastName);
  if (isAllUpperCase(last)) {
    last = capitalizeParts(last);
  }

  return [firstInitial, last].filter(Boolean).join(' ');
}

/** Joins first and last name as they are, leaving out an empty part. */
export function fullName(firstName: string, lastName: string): string {
  return [collapseWhitespace(firstName), collapseWhitespace(lastName)].filter(Boolean).join(' ');
}

/** Trims a text and reduces every run of whitespace to a single space. */
function collapseWhitespace(text: string): string {
  return text.trim().split(/\s+/).filter(Boolean).join(' ');
}

/** Reports whether a text has letters and none of them is lowercase. */
function isAllUpperCase(text: string): boolean {
  return text !== text.toLowerCase() && text === text.toUpperCase();
}

/** Lowercases a text and capitalizes the first letter of every part of it. */
function capitalizeParts(text: string): string {
  return text
    .toLowerCase()
    .replace(NAME_PART_START, (_match, separator: string, letter: string) => {
      return separator + letter.toUpperCase();
    });
}
