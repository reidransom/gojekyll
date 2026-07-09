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
