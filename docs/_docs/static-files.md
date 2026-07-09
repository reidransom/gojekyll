---
title: Static Files
permalink: /docs/static-files/
description: Files without front matter — copied verbatim and listed in site.static_files.
---

A static file is a file that does not contain any front matter. These
include images, PDFs, and other un-rendered content. They are copied to the
destination verbatim.

Static files are accessible in Liquid via `site.static_files` and contain
the following metadata:

| Variable | Description |
| --- | --- |
| `file.path` | The site-relative path to the file, e.g. `/assets/img/image.jpg` |
| `file.modified_time` | The time the file was last modified |
| `file.name` | The string name of the file, e.g. `image.jpg` |
| `file.basename` | The string basename of the file, e.g. `image` |
| `file.extname` | The extension name of the file, e.g. `.jpg` |

Note that in the above table, `file` can be anything — it's an arbitrary
variable used in your own logic, such as a `for` loop, not a global site or
page variable:

{% raw %}
```liquid
{% for file in site.static_files %}
  {{ file.path }}
{% endfor %}
```
{% endraw %}

Which files count as static is affected by the `include:` and `exclude:`
configuration keys, and files or directories whose names start with `.`,
`#`, `~`, or `_` are skipped unless listed under `include:` — the same rules
as Jekyll.

## Front matter values for static files

> **Differs from Jekyll.** Jekyll lets you attach front matter values to
> static files through the `defaults` configuration (e.g. tag every file
> under `assets/img` with `image: true`, then filter `site.static_files`
> with `where`). Jigyll does not — static files expose only the five
> metadata fields above, and `defaults` values never reach them. To select
> a group of static files, filter on `path` instead:

{% raw %}
```liquid
{% for file in site.static_files %}
  {% if file.path contains "/assets/img/" %}
    {{ file.path }}
  {% endif %}
{% endfor %}
```
{% endraw %}
