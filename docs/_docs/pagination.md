---
title: Pagination
permalink: /docs/pagination/
description: Pagination is not implemented in Jigyll.
---

In Jekyll, the `jekyll-paginate` plugin breaks a post listing into multiple
pages, driven by the `paginate` and `paginate_path` configuration keys and a
`paginator` Liquid object. See the [Jekyll pagination
docs](https://jekyllrb.com/docs/pagination/) for how that works.

**Jigyll does not implement pagination.**

- The `paginate` and `paginate_path` configuration keys are ignored.
- No `page2/`, `page3/`, … files are generated.
- Listing `jekyll-paginate` under `plugins:` is recognized (so a site config
  ported from Jekyll doesn't fail), and it exposes a stub `paginator` object —
  first page only, five posts per page — so that themes referencing
  `paginator` don't crash. Do not rely on it.

## What to do instead

For a site small enough for Jigyll's speed to make pagination unnecessary,
list everything:

{% raw %}
```liquid
{% for post in site.posts %}
  <a href="{{ post.url }}">{{ post.title }}</a>
{% endfor %}
```
{% endraw %}

Or group an archive by year:

{% raw %}
```liquid
{% assign by_year = site.posts | group_by_exp: "post", "post.date | date: '%Y'" %}
{% for year in by_year %}
  <h2>{{ year.name }}</h2>
  {% for post in year.items %}
    <a href="{{ post.url }}">{{ post.title }}</a>
  {% endfor %}
{% endfor %}
```
{% endraw %}
