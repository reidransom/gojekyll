---
title: Themes
permalink: /docs/themes/
description: Using Jekyll themes with Jigyll — the _theme folder, Bundler fallback, and what's not supported.
---

Themes package up layouts, includes, and stylesheets in a way that can be
overridden by your site's content. Jigyll supports Jekyll's theme layout:
a theme contributes its `_layouts`, `_includes`, `_sass`, and `assets`
directories to your site's build.

Activate a theme with the `theme` key in `_config.yml`:

```yaml
theme: minima
```

## How Jigyll finds a theme

> **Differs from Jekyll.** There is no gem installation step. Jigyll
> resolves the theme directory in two ways, in order:
>
> 1. **A local `_theme` folder** — if `<source>/_theme/<theme-name>/`
>    exists, it is used. This is the Jigyll-native way to vendor a theme:
>    copy (or submodule) the theme's files there.
> 2. **Bundler fallback** — otherwise, if `bundle` is on your `PATH`,
>    Jigyll runs `bundle show <theme-name>` and uses the gem's directory.
>    This lets an existing Jekyll project with a `Gemfile` keep working,
>    but it does require a Ruby toolchain.
>
> If neither works, the build fails with an error. The
> `jekyll-remote-theme` plugin (`remote_theme:` key) is **not supported** —
> vendor the theme into `_theme/` instead.

## What a theme provides

Jigyll reads these directories from the theme, with your site's own files
always taking precedence:

- `_layouts` — a page's `layout` is looked up in your site's `_layouts`
  first, then the theme's.
- `_includes` — `{% raw %}{% include %}{% endraw %}` searches your site's
  `_includes` first, then the theme's.
- `_sass` — the theme's partials are on the Sass load path; a same-named
  partial in your site's `_sass` wins.
- `assets` — theme assets are output unless your site has a file at the
  same path.

> **Differs from Jekyll.** A theme's `_data` directory (Jekyll 4.3+) and a
> theme-bundled `_config.yml` (Jekyll 4.0+) are ignored, the theme's
> gemspec `runtime_dependencies` are not auto-loaded as plugins (list the
> [emulated plugins](/docs/plugins/) you need explicitly), and there are no
> `theme.*` Liquid variables.

## Overriding theme defaults

To replace layouts or includes in your theme, make a copy in your
`_layouts` or `_includes` directory of the specific file you wish to
modify, or create the file from scratch giving it the same name as the file
you wish to override.

For example, if your selected theme has a `page` layout, you can override
the theme's layout by creating your own `page` layout in the `_layouts`
directory (that is, `_layouts/page.html`).

To modify a stylesheet, also copy the theme's main Sass file into the
`_sass` directory in your site's source. Your theme's styles can be
included in your stylesheet using the `@import` directive:

{% raw %}
```css
@import "{{ site.theme }}";
```
{% endraw %}

## Creating themes

Jekyll's `jekyll new-theme` scaffolding and RubyGems publishing workflow
don't apply to Jigyll. A Jigyll theme is just a directory with `_layouts`,
`_includes`, `_sass`, and `assets` folders — anything that follows that
shape (including any existing Jekyll theme's source) can be dropped into
`_theme/<name>/` and used directly.
