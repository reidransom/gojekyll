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


## Remaining discrepancies

| Severity | Gap | Evidence |
|---|---|---|
| Medium | Liquid highlight output differs | Production emits `<figure class="highlight">`; Jigyll emits bare `<pre class="chroma">`. Three code blocks on Line Numbers lose the production borders and spacing. |
| Medium | SEO metadata differs | Jigyll omits `generator` and Twitter metadata, emits `og:type=article` instead of `website`, and adds a build-time `article:published_time`. Canonical URL input now uses the corrected directory path. |
| Medium | GitHub metadata content is absent | Production home contains 97 contributor avatars; Jigyll contains none. [INFERENCE] `site.github.contributors` is not populated. |
| Low | Typography transformations differ | Production smartifies some quotation marks; Jigyll leaves straight quotes. |



### 1. Highlight markup differs

On Line Numbers:

```text
Production: 3 <figure class="highlight"> blocks
Jigyll:     0 <figure> blocks
```

The highlighted tokens are present, but Jigyll’s Chroma structure does not receive the same theme styling. This accounts for much of the 60–76px page-height difference observed on desktop and mobile.

### 2. Metadata differs

Representative production metadata:

```text
generator: Jekyll v4.4.1
og:type: website
twitter:card: summary
canonical: .../line-numbers/
```

Jigyll:

```text
generator: missing
og:type: article
twitter metadata: missing
article:published_time: current build timestamp
canonical: .../line-numbers/
```

The canonical path now matches the configured directory route; the remaining
metadata differences are unchanged.

## Recommended next compatibility order

1. Match Jekyll’s `{% highlight %}` wrapper contract.
2. Align `jekyll-seo-tag` metadata.
3. Populate `site.github` metadata or explicitly document that plugin gap.

The array-removal/navigation fix is verified, but full visual and functional parity should not yet be declared.