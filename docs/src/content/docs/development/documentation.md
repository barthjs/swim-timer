---
title: Documentation
description: How this documentation is written, translated, and built.
sidebar:
  order: 4
---

This documentation is an [Astro](https://astro.build) site with [Starlight](https://starlight.astro.build) in `docs/`.
Its pages are Markdown files, so they can be read in the repository as well.

| Path                                | Content                                                                 |
| ----------------------------------- | ----------------------------------------------------------------------- |
| `docs/src/content/docs/`            | The English pages.                                                      |
| `docs/src/content/docs/<language>/` | The translations, with the same paths as the English pages.             |
| `docs/src/content/i18n/`            | The texts of the own components, such as the breadcrumbs, per language. |
| `docs/src/components/`              | Components that replace parts of Starlight.                             |
| `docs/public/api.json`              | The OpenAPI document of the API reference.                              |
| `docs/astro.config.ts`              | The languages, the sidebar, and the plugins.                            |

## Writing a page

Every page starts with a frontmatter instead of a top-level heading, since Starlight shows the title as the heading:

```md
---
title: Page title
description: One sentence about the page.
sidebar:
  order: 1
---
```

- The sidebar lists the directories in `astro.config.ts` and orders the pages of a directory by `sidebar.order`.
- Links between pages point to the Markdown file, such as `../installation/linux.md#files`, so they work
  in the repository and on the site.
- A link to a path of the site that is not a page, such as the API reference, starts with `/`, such as `/api/`.
- Downloads link to the latest release, such as `releases/latest/download/swim-timer_linux_amd64.tar.gz` on GitHub.
  The build of a release points them at that release, so every version offers its own downloads.
- The build checks every link and anchor and fails on a broken one.
- Pages for users and admins describe what Swim-Timer does and how to use it, without implementation details such as
  endpoints. Those only appear on the development pages and in the API reference.

## Languages

English is the main language of the documentation. Every page is written in English first, at the root of
`docs/src/content/docs/`. A translation has the same path in the folder of its language, such as `de/`.

- A change to an English page is carried over to every translation of that page in the same change.
- New texts of the own components go into the files of every language in `docs/src/content/i18n/`.
- New sidebar groups get a label in every language in `astro.config.ts`.
- A page without a translation falls back to the English page, which Starlight marks as not translated yet.

## API reference

The API reference is a page with [Scalar](https://scalar.com) that reads `docs/public/api.json`. `make docs-api` generates
this file from the annotations of the API handlers, so run it after changing a handler, a request, or a response.

## Building

`npm run build` in `docs/` builds the site into `docs/dist` and checks the types. A step after the Astro build:

- makes every link relative, so the build works under any path,
- moves inline scripts into files of their own, since the instance only allows scripts from its own origin,
- writes the version from `DOCS_VERSION` into the API reference.

The release embeds the documentation into the binary, and the instance serves it under `/docs/`. It also publishes
every version on GitHub Pages, see [Public documentation](./releases.md#public-documentation).

The header shows the version next to the title. On GitHub Pages, it is a select that lists every version
from `versions.json` and opens the current page in the chosen version, or the start page of that version if the page
does not exist there. An instance has no `versions.json` and shows only its own version.

## Links from the web interface

The web interface links to the documentation of its own version: the footer to the start page, and every page
to the section that explains it. `web/src/lib/docs.ts` lists these targets, and `npm run test` in `docs/` checks
them against the built site, so a renamed page or heading fails until its target is updated.
