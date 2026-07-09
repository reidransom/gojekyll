---
title: Differences from Jekyll
permalink: /docs/differences/
description: Where Jigyll's behavior diverges from Ruby Jekyll.
---

Jigyll targets Jekyll compatibility, but it is a separate implementation and
some behavior differs. This page summarizes the notable differences. (A future
release will generate this automatically from per-page compatibility metadata.)

## Not supported

- **Pagination** (`jekyll-paginate`).
- **Math** — `$…$` / `$$…$$` pass through as literal text.
- **Table of contents** — kramdown's `{:toc}` is not expanded.
- **An extensible plugin system** — only plugins compiled into the binary are
  available. [Some plugins](https://github.com/reidransom/jigyll/blob/main/docs/plugins.md)
  are emulated.
- The `sassify` Liquid filter (indented Sass). `scssify` works.

## Behaves differently

- **Strict Liquid.** Undefined variables and filters are errors, not silent
  blanks.
- **Strict configuration.** A wrong type in `_config.yml` (a list where a
  string is expected, etc.) is generally an error.
- **Markdown** is rendered by goldmark, not kramdown. Raw `<` / `>` is treated
  as HTML, and auto-generated header IDs replace punctuation with hyphens.
- **Plugins** are listed in `_config.yml`, not a `Gemfile`.
- **`serve`** renders in memory (no files written) and **live reload is always
  on**.
- Caches live in `/tmp/jigyll-$USER`, not `./.sass-cache`.

## Jigyll-only

- **Shopify-style money filters** — `money`, `money_with_currency`,
  `money_without_currency`, and `money_without_trailing_zeros`.

## Not in Jigyll's CLI

The `new`, `doctor`, and `import` commands are not implemented.
