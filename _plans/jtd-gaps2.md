## Conclusion

The Jigyll build is **not yet Ruby-Jekyll compatible** with the live [just-the-docs.com](https://just-the-docs.com/) site.

The recently fixed navigation and Kramdown attribute-list rendering are correct. Sidebar hierarchy, breadcrumbs, responsive geometry, children navigation, callouts, buttons, labels, spacing utilities, and heading classes match. Broader rendering and URL-generation gaps remain.

Comparison baseline:

- Jigyll: pinned source `394d6c0ec33852f8e593145d21344a955e908acb`
- Production: live site, reporting `Jekyll v4.4.1`
- Pages checked:
  - `/`
  - `/docs/ui-components/`
  - `/docs/ui-components/code/line-numbers/`
- Viewports: desktop `1280×800`, mobile `375×812`
- Also compared sitemaps, CSS, JavaScript, search, metadata, and generated links

The production site does not expose its source commit, so exact source-revision equivalence is not guaranteed.

## What matches

- Sidebar navigation hierarchy renders successfully.
- Both Line Numbers pages contain 46 sidebar links.
- Breadcrumbs match exactly:
  - `UI Components`
  - `Code`
  - `Line Numbers`
- Desktop geometry matches:
  - sidebar width: `372px`
  - main width: `800px`
  - content width: `736px`
  - matching typography, colors, and positioning
- Mobile geometry matches:
  - `375px` page width
  - `52.5px` mobile header
  - hidden navigation and visible menu button
  - no horizontal overflow
- Both sitemaps contain 47 page entries.
- `just-the-docs-head-nav.css` is byte-for-byte identical.
- The main stylesheet has 2,031 rules in both builds. Production is expanded; Jigyll emits a compressed and color-normalized form.
- The generated pages contain the expected sidebar, breadcrumbs, children list, footer, and main content.

## Resolved discrepancies

### Kramdown attribute lists

Jigyll now applies Kramdown block and inline attribute lists to paragraphs,
headings, blockquotes, lists, links, images, and other supported inline nodes.
Attribute syntax remains literal inside fenced and inline code.

Verified against pinned source `394d6c0ec33852f8e593145d21344a955e908acb`:

```text
47 HTML files
0 raw attribute paragraphs
```

The Line Numbers warning renders as `<p class="warning">`, and the home page
renders its `fs-9`, `fs-6 fw-300`, and button classes without visible IAL
syntax.

### `relative_url` root paths

`relative_url` now roots non-empty relative inputs and prefixes `baseurl`
exactly once, matching Jekyll’s string URL behavior. Scheme-absolute inputs
remain unchanged.

Verified against pinned source `394d6c0ec33852f8e593145d21344a955e908acb`:

```text
59 files written
search-data request: /assets/js/search-data.json (200)
callout search: populated results
dark stylesheet: /assets/css/just-the-docs-dark.css (200)
dark body background: rgb(39, 38, 43)
nested /docs/ui-components/code/assets/ requests: none
```

A focused `baseurl: /docs` render emits `/docs/assets/js/search-data.json`
and `/docs/assets/css/just-the-docs-`.

Covers [`just-the-docs-relative-url-filter.md`](just-the-docs-relative-url-filter.md).

### Global `permalink: pretty`

Ordinary HTML pages now use trailing-slash directory routes when the site sets
`permalink: pretty`; explicit page permalinks still override that default.

Verified against pinned source `394d6c0ec33852f8e593145d21344a955e908acb`:

```text
59 files written
47 Jigyll sitemap entries
47 production sitemap entries
47/47 normalized route paths match
/docs/configuration/ -> 200
/docs/ui-components/code/line-numbers/ -> 200
/docs/configuration.html -> 404
```

The corrected directory routes reach Liquid links, the sitemap, and canonical
URL input without a consumer-specific rewrite. Representative output files are
`docs/configuration/index.html` and
`docs/ui-components/code/line-numbers/index.html`; their former `.html`
destinations are absent.

### Liquid highlight wrapper

Liquid `{% raw %}{% highlight %}{% endraw %}` blocks now use one
`figure.highlight > pre > code` shell with normalized language metadata.
Chroma continues to supply token spans and line-number table internals.

Verified against pinned source `394d6c0ec33852f8e593145d21344a955e908acb`:

```text
59 files written
Liquid source blocks:                         3
generated figure.highlight blocks:            3
code.language-yaml[data-lang="yaml"] shells:  3
bare Chroma roots inside those figures:        0
fenced div.highlighter-rouge blocks:           2
```

The three code bodies begin with `compress_html:`, `kramdown:`, and the
literal nested highlight example, without delimiter-created blank lines.
The production and Jigyll shells have the same `figure.highlight > pre >
code.language-yaml[data-lang="yaml"]` hierarchy. At `1280×800`, both pages
measured `2389px` high and their three figure bottoms were `568px`, `745px`,
and `904px`. At `375×812`, both pages applied figure borders, `12px` padding
and margins, `overflow-x: auto`, copy buttons, and no horizontal page
overflow; the pinned Jigyll page measured `2605px` and the live production
page `2953px`. The latter total remains source-version-sensitive because the
live site does not expose its source commit.

Covers [`just-the-docs-highlight-wrapper.md`](just-the-docs-highlight-wrapper.md).

### SEO metadata

The pinned Just the Docs source now builds 59 files with 47 HTML routes. All
47 routes emit the Jigyll generator identity, `twitter:card=summary`,
`twitter:title`, and `og:type=website`; none emit article publication or
modification times. The home page emits `WebSite` JSON-LD with
`name=Just the Docs`, while Line Numbers emits `WebPage` JSON-LD with
`https://schema.org`, the configured description, and its corrected directory
canonical URL.

The SEO data builder derives article and `BlogPosting` metadata only from a
real document date, so explicitly dated documents retain stable publication
and modification timestamps. Canonical, Open Graph, and JSON-LD URLs share the
same baseurl-aware resolver.

Covers [`just-the-docs-seo-metadata.md`](just-the-docs-seo-metadata.md).



## Remaining discrepancies

| Severity | Gap | Evidence |
|---|---|---|
| Medium | GitHub metadata content is absent | Production home contains 97 contributor avatars; Jigyll contains none. [INFERENCE] `site.github.contributors` is not populated. |
| Low | Typography transformations differ | Production smartifies some quotation marks; Jigyll leaves straight quotes. |






## Recommended next compatibility order

1. Populate `site.github` metadata or explicitly document that plugin gap.
2. Align typography transformations.

The array-removal/navigation fix is verified, but full visual and functional parity should not yet be declared.