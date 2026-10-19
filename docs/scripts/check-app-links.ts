import { readFile } from 'node:fs/promises';
import path from 'node:path';

/**
 * missingTarget returns why a link of the interface into the documentation leads nowhere in the built site,
 * or undefined if its page exists and holds its anchor.
 */
export async function missingTarget(dist: string, target: string): Promise<string | undefined> {
  const [page = '', anchor] = target.split('#');
  const file = path.join(dist, page, 'index.html');

  let html: string;
  try {
    html = await readFile(file, 'utf8');
  } catch {
    return `${target}: no page at ${page}`;
  }

  if (anchor && !html.includes(`id="${anchor}"`)) {
    return `${target}: no section #${anchor}`;
  }

  return undefined;
}

/** checkAppLinks fails if a page or section that the interface links to is missing from the built site. */
export async function checkAppLinks(dist: string, targets: string[]): Promise<void> {
  const problems: string[] = [];
  for (const target of targets) {
    const problem = await missingTarget(dist, target);
    if (problem) {
      problems.push(problem);
    }
  }

  if (problems.length > 0) {
    throw new Error(`links of the interface lead nowhere:\n${problems.join('\n')}`);
  }
}

// The targets are imported only here, so the tests do not load the web interface, whose packages may be missing
if (import.meta.main) {
  const { DOCS_PAGES } = await import('../../web/src/lib/docs.ts');
  await checkAppLinks(path.resolve(process.argv[2] ?? 'dist'), Object.values(DOCS_PAGES));
}
