---
title: Assets
parent: Content
nav_order: 6
permalink: /docs/assets/
description: Sass and SCSS conversion in Jigyll — Liquid first, compressed CSS, limited options.
---

Jigyll provides built-in support for [Sass](https://sass-lang.com/). To have
a file converted, give it a `.sass` or `.scss` extension and ***start the
file with two lines of triple dashes***, like this:

```sass
---
---

// start content
.my-definition
  font-size: 1.2em
```

Jigyll treats these files the same as a regular page: the output file is
placed in the same directory it came from. For instance, a file named
`css/styles.scss` is processed and written to your site's destination folder
as `css/styles.css`.

Like Jekyll, Jigyll processes the body after front matter through Liquid before
sending it to the Sass compiler.

> **Differs from Jekyll.** CoffeeScript is not supported. There is no equivalent
> of the `jekyll-coffeescript` plugin; `.coffee` files without front matter are
> copied verbatim as static files.

## Sass/SCSS

Place all your partials in your `sass_dir`, which defaults to
`<source>/_sass`. Place your main SCSS or Sass files where you want them to
be in the output, such as `<source>/css`. Files in the `sass_dir` are the
load path for `@import` — they should *not* have front matter, and they are
not written to the site themselves:

```yaml
sass:
  sass_dir: _sass   # the default
```

> **Differs from Jekyll.** `sass_dir` is the only `sass:` option Jigyll
> honors. Output is **always compressed** — `style` is ignored — and
> `load_paths`, `sourcemap`, Sass warning controls, and deprecation controls are
> not supported. Conversion requires the
> [Dart Sass](https://sass-lang.com/dart-sass/) `sass` executable on your
> `PATH` (the [install script](/docs/installation/) sets this up for you),
> and compiled CSS is cached in `/tmp/jigyll-$USER`.

If your site uses a [theme](/docs/themes/), the theme's `_sass` partials are
on the load path too, and partials in your site's `_sass` override
same-named ones from the theme.
