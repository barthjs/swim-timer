/**
 * Formats a first and last name into a full name
 */
export function formatName(firstName: string, lastName: string): string {
  const first = firstName.trim();

  const firstInitial = first.length > 0 ? first.charAt(0).toUpperCase() + '.' : '';

  const last = lastName.trim().split(/\s+/).filter(Boolean).pop();

  const formattedLast = last ? last.charAt(0).toUpperCase() + last.slice(1).toLowerCase() : '';

  return [firstInitial, formattedLast].filter(Boolean).join(' ');
}
