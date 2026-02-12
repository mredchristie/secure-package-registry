+++
title = "Replacing biome with prettier for Svelte formatting and linting"
author = "cheongyx@cardiff.ac.uk"
date = "2026-02-12"
supersedes = ["2026-01-30-formatting-linting-choices.md"]
+++

## Summary

Biome does not properly support Svelte files, leading to false positives in linting. For example, any bindings of
variables from JavaScript to HTML were treated as unused, and `--fix --unsafe` broke builds.

As such, we are forced to use Prettier instead with `sveltejs/prettier-plugin-svelte`.

There were a couple bugs encountered along the way, the main one being that Prettier automatically attempts to discover
plugins, leading to errors when loading a nested `package.json`. **Solution**: Add `--config ./.prettierrc` to
`bunx prettier --write "**/*.md"` to ensure it doesn't read nested configurations.
