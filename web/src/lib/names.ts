/** The start of a part of a name: the beginning, or a space, hyphen, or apostrophe before a letter. */
const NAME_PART_START = /(^|[\s'’-])(\p{L})/gu;

/**
 * formatName writes a competitor as the initial of the first name and the whole last name, with whitespace collapsed.
 * A last name written entirely in capitals, as many timing exports write it, is capitalized per part,
 * split at spaces, hyphens, and apostrophes, without recognizing particles. Any other last name keeps its case.
 */
export function formatName(firstName: string, lastName: string): string {
  const first = collapseWhitespace(firstName);
  const firstInitial = first.length > 0 ? first.charAt(0).toUpperCase() + '.' : '';

  let last = collapseWhitespace(lastName);
  if (isAllUpperCase(last)) {
    last = capitalizeParts(last);
  }

  return [firstInitial, last].filter(Boolean).join(' ');
}

/** fullName joins first and last name as they are, leaving out an empty part. */
export function fullName(firstName: string, lastName: string): string {
  return [collapseWhitespace(firstName), collapseWhitespace(lastName)].filter(Boolean).join(' ');
}

/** collapseWhitespace trims a text and reduces every run of whitespace to a single space. */
function collapseWhitespace(text: string): string {
  return text.trim().split(/\s+/).filter(Boolean).join(' ');
}

/** isAllUpperCase reports whether a text has letters and none of them is lowercase. */
function isAllUpperCase(text: string): boolean {
  return text !== text.toLowerCase() && text === text.toUpperCase();
}

/** capitalizeParts lowercases a text and capitalizes the first letter of every part of it. */
function capitalizeParts(text: string): string {
  return text
    .toLowerCase()
    .replace(NAME_PART_START, (_match, separator: string, letter: string) => {
      return separator + letter.toUpperCase();
    });
}
