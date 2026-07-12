---
title: Plugins
permalink: /docs/plugins/
description: Jigyll has no extensible plugin system, but emulates a fixed set of popular Jekyll plugins.
---

Jekyll has an extensible Ruby plugin system — generators, converters,
commands, tags, filters, and hooks. **Jigyll does not.** Plugins are Go code
compiled into the binary, and only a fixed set of popular Jekyll plugins is
emulated.¹

Enable them in `_config.yml` (never a Gemfile):

```yaml
plugins:
  - jekyll-feed
  - jekyll-seo-tag
  - jekyll-sitemap
```

The legacy `gems:` key works as an alias. Naming a plugin Jigyll doesn't know
prints a warning. To see the plugin names your installed binary recognizes,
run `jigyll plugins` — note it lists *recognized* names; the table below is
the source of truth for how completely each is implemented.

## Emulated plugins

| Plugin | Motivation | Status | Missing features |
| --- | --- | --- | --- |
| [jekyll-avatar][jekyll-avatar] | GitHub Pages² | ✓ | |
| [jekyll-coffeescript][jekyll-coffeescript] | GitHub Pages | ✗ | |
| [jekyll-default-layout][jekyll-default-layout] | GitHub Pages | ✓ | |
| [jekyll-feed][jekyll-feed] | GitHub Pages | ✓ | |
| [jekyll-gist][jekyll-gist] | core³ | ✓ | |
| [jekyll-github-metadata][jekyll-github-metadata] | GitHub Pages | partial | `contributors`, `public_repositories`, `show_downloads`, `releases`, `versions`, `wiki_url`; Octokit configuration; GitHub Enterprise |
| [jekyll-live-reload][jekyll-live-reload] | core | ✓ | always enabled (by design); no way to disable |
| [jekyll-mentions][jekyll-mentions] | GitHub Pages | ✓ | |
| [jekyll-optional-front-matter][jekyll-optional-front-matter] | GitHub Pages | ✓ | |
| [jekyll-paginate][jekyll-paginate] | core | ✓ | jekyll-paginate-v2 features; page URLs are directory-normalized (`/page2/`) — see [Pagination](/docs/pagination/) |
| [jekyll-readme-index][jekyll-readme-index] | GitHub Pages | ✓ | |
| [jekyll-redirect-from][jekyll-redirect-from] | GitHub Pages | ✓ | user template |
| [jekyll-relative-links][jekyll-relative-links] | GitHub Pages | ✓ | |
| [jekyll-sass-converter][jekyll-sass-converter] | core | ✓ | always enabled (by design); no way to disable; only `sass_dir` is configurable |
| [jekyll-seo-tag][jekyll-seo-tag] | GitHub Pages | partial | `dateModified`, `datePublished`, `publisher`, `mainEntityOfPage`, `@type` |
| [jekyll-sitemap][jekyll-sitemap] | GitHub Pages | ✓ | file modified dates⁴ |
| [jekyll-titles-from-headings][jekyll-titles-from-headings] | GitHub Pages | ✗ | |
| [jemoji][jemoji] | GitHub Pages | ✓ | image tag fallback |
| [github-pages][github-pages] | GitHub Pages | ✓ | enables the plugins github-pages includes, each in the state listed above |

¹ The internal APIs are too immature for a stable plugin interface, and Go's
[native plugin mechanism](https://golang.org/pkg/plugin/) only works on
Linux.

² Listed in the [GitHub Pages dependency
versions](https://pages.github.com/versions/).

³ "Core" plugins are referenced in the main [Jekyll
documentation](https://jekyllrb.com/docs/).

⁴ Modified dates aren't that useful with source control and CI. (Post dates
are included.)

[jekyll-avatar]: https://github.com/benbalter/jekyll-avatar
[jekyll-coffeescript]: https://github.com/jekyll/jekyll-coffeescript
[jekyll-default-layout]: https://github.com/benbalter/jekyll-default-layout
[jekyll-feed]: https://github.com/jekyll/jekyll-feed
[jekyll-gist]: https://github.com/jekyll/jekyll-gist
[jekyll-github-metadata]: https://github.com/parkr/github-metadata
[jekyll-live-reload]: https://github.com/RobertDeRose/jekyll-livereload
[jekyll-mentions]: https://github.com/jekyll/jekyll-mentions
[jekyll-optional-front-matter]: https://github.com/benbalter/jekyll-optional-front-matter
[jekyll-paginate]: https://github.com/jekyll/jekyll-paginate
[jekyll-readme-index]: https://github.com/benbalter/jekyll-readme-index
[jekyll-redirect-from]: https://github.com/jekyll/jekyll-redirect-from
[jekyll-relative-links]: https://github.com/benbalter/jekyll-relative-links
[jekyll-sass-converter]: https://github.com/jekyll/jekyll-sass-converter
[jekyll-seo-tag]: https://github.com/jekyll/jekyll-seo-tag
[jekyll-sitemap]: https://github.com/jekyll/jekyll-sitemap
[jekyll-titles-from-headings]: https://github.com/benbalter/jekyll-titles-from-headings
[jemoji]: https://github.com/jekyll/jemoji
[github-pages]: https://github.com/github/pages-gem
