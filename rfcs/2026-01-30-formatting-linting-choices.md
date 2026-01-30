+++
title = "Formatting and Linting Choices"
author = "ReadB5@cardiff.ac.uk"
date = "2026-01-30"
status = "Draft"
+++

## Summary

This RFC discusses various formatting choices for documents, primarily focusing on the tools used

Choices made:
- Use `go tool` and `bunx`, so we don't have to deal with installing multiple formatters.
- We replaced `lint` and `fmt` with just `fix` so that we don't have to type multiple commands. These commands are designed to fix everything rather than reading anyway.
- `just` is used to manage commands, as it is more flexible than `make` and easier to use than NPM scripts.
- `just` does not handle pre-commit hooks so that is set up separately.
- Use `biome` for formatting and linting the web ecosystem, as it is more modern and faster than alternatives like `eslint` and `prettier`.
- Use `prettier` just for markdown files, as `biome` does not support formatting Markdown well.

- `set -e` is used for the check script to ensure that any failure in the commands causes the script to exit immediately, preventing further execution and potential errors. This is unnecessary for the `fix` script as it is designed to fix everything, and therefore we prefer it to fix as much as possible rather than stopping at the first error.

- A precommit script is used to ensure that code is always formatted before committing, we use the same commands as in `just` to avoid duplication of effort.


