import { parseVersionIndex, versionLabel, versionUrl } from './versions.ts';

/**
 * Fills the version select of every page header from the versions.json next to the folder of the version,
 * which only the public documentation has, so an instance keeps showing its version as text.
 */
export async function mountVersionSelect(root: URL): Promise<void> {
  const containers = document.querySelectorAll<HTMLElement>('[data-docs-version]');
  if (containers.length === 0) {
    return;
  }

  let data: unknown;
  try {
    const response = await fetch(new URL('../versions.json', root));
    if (!response.ok) {
      return;
    }
    data = await response.json();
  } catch {
    // An instance answers with its web interface instead of a JSON file
    return;
  }

  const index = parseVersionIndex(data);
  if (!index) {
    return;
  }

  for (const container of containers) {
    const current = container.dataset.docsVersion ?? '';
    const select = container.querySelector('select');
    if (!select) {
      continue;
    }

    const labels = {
      development: container.dataset.labelDevelopment ?? '',
      latest: container.dataset.labelLatest ?? '',
    };
    for (const version of index.versions) {
      const option = document.createElement('option');
      option.value = version;
      option.text = versionLabel(version, index, labels);
      option.selected = version === current;
      select.add(option);
    }
    select.addEventListener('change', () => {
      void openVersion(root, select.value);
    });
    select.hidden = false;
  }
}

/** Opens the current page in another version, or the start page of that version if it lacks the page. */
async function openVersion(root: URL, version: string): Promise<void> {
  const target = versionUrl(new URL(window.location.href), root, version);

  try {
    const response = await fetch(target, {
      method: 'HEAD',
    });
    window.location.assign(response.ok ? target : new URL(`../${version}/`, root));
  } catch {
    window.location.assign(new URL(`../${version}/`, root));
  }
}
