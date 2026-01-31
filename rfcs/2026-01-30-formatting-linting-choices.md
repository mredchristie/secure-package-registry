+++
title = "Formatting and Linting Choices"
author = "ReadB5@cardiff.ac.uk"
reviewer = ["cheongyx@cardiff.ac.uk"]
date = "2026-01-30"
status = "implemented"
+++

## Summary

We choose a standard set of formatters to ensure consistency and prevent merge conflicts, and linters to avoid common
code smells.

The specific choices of linters and formatters:

- `prettier`: Markdown formatting
- `markdownlint-cli2`: Markdown linting (finds issues that can't be automatically fixed)
- `gofumpt`: Go formatting stricter than built-in `go fmt`. It provides better consistency across the code base.
- `golangci-lint`: Go linter. Catches bugs that compiler cannot via static analysis.
- `@biomejs/biome`: Typescript formatter. Currently the modern standard. Includes linting.

We furthermore use `just` as a runner with `just fix` and `just check` scripts to automatically fix issues and detect
errors. `pre-commit` is used to avoid pushing up unformatted commits.

## Specific choices made

- Use `go tool` and `bunx`, so we don't have to deal with installing multiple formatters.
- We replaced `lint` and `fmt` with just `fix` so that we don't have to type multiple commands. These commands are
  designed to fix everything rather than reading anyway.
- `just` is used to manage commands, as it is more flexible than `make` and easier to use than NPM scripts.
- Use `biome` for formatting and linting the web ecosystem instead of prettier, as it is more capable and fast.
- Use `prettier` for markdown files as `markdownlint-cli2` fails to fix some whitespace issues.
- `set -e` is used for the check script to ensure that any failure in the commands causes the script to exit
  immediately, preventing further execution and potential errors. This is unnecessary for the `fix` script as it is
  designed to fix everything, and therefore we prefer it to fix as much as possible rather than stopping at the first
  error.
