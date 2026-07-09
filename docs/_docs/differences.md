---
title: Differences from Jekyll
permalink: /docs/differences/
description: Where Jigyll's behavior diverges from Ruby Jekyll.
---

Jigyll targets Jekyll compatibility, but it is a separate implementation and
some behavior differs. The sections below are generated from the same
compatibility metadata (`_data/compat.yml`) that powers the callout at the top
of each affected page and the badges in the sidebar — one source of truth,
three surfaces.

{% assign statuses = "replaced|modified|unsupported|jigyll-only" | split: "|" -%}
{% for status in statuses -%}
{% capture rows -%}
{% for section in site.data.docs_nav -%}
{% for item in section.docs -%}
{% assign compat_key = item.link | remove_first: "/docs/" | replace: "/", "" -%}
{% if compat_key == "" %}{% assign compat_key = "index" %}{% endif -%}
{% assign compat = site.data.compat[compat_key] -%}
{% if compat.status == status -%}
- **[{{ item.title }}]({{ item.link | relative_url }})** — {{ compat.summary }}
{% endif -%}
{% endfor -%}
{% endfor -%}
{% endcapture -%}
{% assign trimmed = rows | strip -%}
{% unless trimmed == "" %}
{% case status %}
{% when "replaced" %}## Replaced

These topics work fundamentally differently in Jigyll.
{% when "modified" %}## Modified

Mostly the same as Jekyll, with specific differences.
{% when "unsupported" %}## Not supported

Jekyll has these; Jigyll doesn't.
{% when "jigyll-only" %}## Jigyll-only

Jigyll additions with no Jekyll counterpart.
{% endcase %}
{{ trimmed }}
{% endunless %}
{% endfor %}

## Not yet tied to a docs page

These differences are real today but their reference pages haven't been ported
yet; each will get its own compatibility entry as the docs grow.

- **Strict Liquid.** Undefined variables and filters are errors, not silent
  blanks.
- **Strict configuration.** A wrong type in `_config.yml` (a list where a
  string is expected, etc.) is generally an error.
- **Markdown** is rendered by goldmark, not kramdown. Raw `<` / `>` is treated
  as HTML, auto-generated header IDs replace punctuation with hyphens, and
  kramdown extensions — <code>{&#58;toc}</code>, attribute lists, math — are
  not expanded.
- **Plugins** are listed in `_config.yml`, not a `Gemfile`, and there is no
  extensible plugin system — only plugins compiled into the binary are
  available. [Some plugins](https://github.com/reidransom/jigyll/blob/main/docs/plugins.md)
  are emulated.
- **Pagination** (`jekyll-paginate`) is not implemented.
- The `sassify` Liquid filter (indented Sass) is unimplemented; `scssify`
  works.
- Caches live in `/tmp/jigyll-$USER`, not `./.sass-cache`.
- **Shopify-style money filters** — `money`, `money_with_currency`,
  `money_without_currency`, and `money_without_trailing_zeros` — are a
  Jigyll-only addition.
