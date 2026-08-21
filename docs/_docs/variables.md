---
title: Variables
parent: Site Structure
nav_order: 3
permalink: /docs/variables/
description: The site, page, layout, and jekyll variables Jigyll exposes to Liquid.
---

Jigyll traverses your site looking for files to process. Any files with
[front matter](/docs/front-matter/) are subject to processing. For each of
these files, Jigyll makes a variety of data available via
[Liquid](/docs/liquid/). The following is a reference of the available data.

## Global Variables

| Variable | Description |
| --- | --- |
| `site` | Site-wide information + configuration settings from `_config.yml`. See below for details. |
| `page` | Page-specific information + the [front matter](/docs/front-matter/). Custom variables set via the front matter will be available here. See below for details. |
| `layout` | Layout-specific information + the front matter. Custom variables set via front matter in [layouts](/docs/layouts/) will be available here. |
| `jekyll` | Version and environment information. See below for details. |
| `content` | In layout files, the rendered content of the post or page being wrapped. Not defined in post or page files. |
| `paginator` | The [pagination](/docs/pagination/) state, on paginated index pages. |

> **Differs from Jekyll.** The `theme` variable (gemspec metadata of the
> active theme gem, Jekyll 4.3+) does not exist.

## Site Variables

| Variable | Description |
| --- | --- |
| `site.time` | The current time (when you ran the `jigyll` command). |
| `site.pages` | A list of all pages. |
| `site.posts` | A reverse-chronological list of all posts. |
| `site.related_posts` | The ten most recent posts. (Jekyll's `--lsi` similarity ranking is not supported.) |
| `site.static_files` | A list of all [static files](/docs/static-files/). |
| `site.html_pages` | A subset of `site.pages` listing those which end in `.html`. |
| `site.html_files` | A subset of `site.static_files` listing those which end in `.html`. |
| `site.collections` | A list of all the [collections](/docs/collections/) (including posts). |
| `site.data` | The data loaded from the files in the `_data` directory. |
| `site.documents` | A list of all the documents in every collection. |
| `site.categories.CATEGORY` | The list of all posts in category `CATEGORY`. |
| `site.tags.TAG` | The list of all posts with tag `TAG`. |
| `site.url` | The url of your site as configured in `_config.yml` (or the `JEKYLL_URL` environment variable). |
| `site.[CONFIGURATION_DATA]` | All the variables set in your `_config.yml` are available through the `site` variable. For example, if you have `foo: bar` in your configuration file, then it will be accessible in Liquid as `site.foo`. |

## Page Variables

| Variable | Description |
| --- | --- |
| `page.content` | The content of the page. |
| `page.title` | The title of the page, from its front matter. |
| `page.excerpt` | The un-rendered excerpt of a page. Can be overridden with an `excerpt` key in the front matter. |
| `page.url` | The URL of the page without the domain, but with a leading slash, e.g. `/2026/12/14/my-post.html` |
| `page.date` | The date assigned to a post. Can be overridden in a post's front matter with a `date` key. |
| `page.id` | An identifier unique to a document in a collection or a post (useful in RSS feeds). |
| `page.categories` | The list of categories to which this post belongs, from its front matter. |
| `page.collection` | The label of the collection this document belongs to — `posts` for a post. Not set for regular pages. |
| `page.tags` | The list of tags to which this post belongs, from its front matter. |
| `page.name` | The filename of the post or page, e.g. `about.md` |
| `page.path` | The path to the raw post or page, relative to the source directory. |
| `page.relative_path` | Same as `page.path`, relative to the source directory. |
| `page.slug` | The filename of the document without its extension (or date prefix, for a post). |
| `page.ext` | The file extension of the source document, e.g. `.md` |
| `page.next` | The next post relative to the position of the current post in `site.posts`. Defined for posts only. |
| `page.previous` | The previous post relative to the position of the current post in `site.posts`. Defined for posts only. |

> **Differs from Jekyll.** `page.dir` does not exist — derive the directory
> from `page.url` or `page.path` if you need it. Categories always come from
> front matter, [never from the directory path](/docs/posts/). And any
> custom front matter is available under `page`, exactly as in Jekyll.

## Jekyll Variables

| Variable | Description |
| --- | --- |
| `jekyll.version` | The Jigyll version used to build the site, suffixed with `(jigyll)` — e.g. `1.2.3 (jigyll)`. It is **not** a Ruby Jekyll version number. |
| `jekyll.environment` | The value of the `JEKYLL_ENV` environment variable during the build, defaulting to `development`. |
