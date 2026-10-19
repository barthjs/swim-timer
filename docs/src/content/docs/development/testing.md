---
title: Testing
description: Run the tests, linters, and formatters of the backend, the web interface, and the documentation.
sidebar:
  order: 3
---

The Makefile covers the Go backend, and the package scripts cover the web interface and the documentation.
The GitHub workflow of the tests runs the same targets and scripts for every pull request against `main`
and before every release.

## Backend

The linters and formatters run through `golangci-lint`, configured in `.golangci.yml`:

| Command             | What it does                                                        |
| ------------------- | ------------------------------------------------------------------- |
| `make fmt`          | Formats the Go code, sorts the imports, and formats the annotations |
| `make tidy`         | Formats the Go code and tidies and verifies the modules             |
| `make test-lint`    | Checks formatting and modules without changing files                |
| `make test-types`   | Runs the linters, including `go vet` and `staticcheck`              |
| `make test-unit`    | Runs the unit tests with coverage                                   |
| `make test-feature` | Runs the feature tests of the API with coverage                     |
| `make test`         | Runs all checks and tests and prints the total coverage             |

The backend embeds the built web interface and documentation, so build both before the Go tests:

```sh
npm run build-only --prefix web
npm run build-only --prefix docs
make test
```

- The tests write their coverage profiles to `coverage/`.
- A finding that is intended stays in the code with a `//nolint:<linter>` directive and a comment with the reason.

## Feature tests

- Feature tests in `internal/api` start the whole application per test through `internal/app`.
- Every test gets storage in a temporary directory and a fake timing system.
- Requests go through the helpers in `internal/apitest`.
- Test data comes from factories that use a faker, so every test runs with different values. A failed test prints
  the faker seed, and setting `APITEST_SEED` to it repeats the run with the same data.

## Web interface and documentation

`web/` and `docs/` have the same scripts:

| Script                   | What it does                                                                                                                |
| ------------------------ | --------------------------------------------------------------------------------------------------------------------------- |
| `npm run dev`            | Starts the development server                                                                                               |
| `npm run build`          | Builds and checks the types                                                                                                 |
| `npm run build-only`     | Builds without checking the types                                                                                           |
| `npm run type-check`     | Checks the types                                                                                                            |
| `npm run lint`           | Fixes the findings of the linters                                                                                           |
| `npm run format`         | Formats the files with Prettier                                                                                             |
| `npm run test`           | Runs `test:lint`, `test:format`, and `test:unit`, in `docs/` also `test:app-links`                                          |
| `npm run test:lint`      | Checks the code with the linters                                                                                            |
| `npm run test:format`    | Checks the formatting                                                                                                       |
| `npm run test:unit`      | Runs the unit tests with Vitest                                                                                             |
| `npm run test:app-links` | Only in `docs/`, checks the [links from the web interface](./documentation.md#links-from-the-web-interface) against a build |
