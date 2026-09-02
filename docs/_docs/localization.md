---
title: Localization
parent: Build
nav_order: 3
permalink: /docs/localization/
description: Build one Jigyll project into coherent locale-specific sites.
---

Localization is opt-in. Add a `localization` block to `_config.yml`; without it,
Jigyll uses its ordinary single-site build.

```yaml
url: https://example.test
baseurl: /guide
localization:
  default_language: en
  locales:
    en:
      tag: en
      label: English
    de:
      tag: de-DE
      label: Deutsch
      fallbacks: [en]
```

Locale keys are lowercase URL-safe project identifiers. Tags are explicit BCP 47
tags. The default locale owns root routes; other locales are prefixed by their
key. Set `default_language_in_subdir: true` to prefix the default locale too.

## Content editions

Use `lang` to assign an edition and `translation_key` to relate editions. A
missing `lang` uses the default locale. `translation_key` is optional, but must
be a non-empty string when present.

```yaml
---
lang: de
translation_key: getting-started
permalink: /erste-schritte/
---
```

Translation keys are scoped to pages or to one collection. Jigyll never
substitutes default-language page content for a missing edition: unavailable
editions are omitted from routes, lists, feeds, and language selectors.

## Data and messages

Shared `_data` remains visible in every locale. Put locale modules below
`_data/locales/<locale>/`; their filenames become data keys and overlay shared
data through the locale fallback chain.

```text
_data/
  settings.yml
  locales/
    en/messages.yml
    de/messages.yml
    de/settings.yml
```

`messages.yml` supplies the `translate` filter. Message keys are dotted paths;
missing messages fail the build unless `missing_messages: key` is configured.

```liquid
{{ "nav.home" | translate }}
{{ site.data.settings.title }}
```

## Liquid contract

`site.language` is the active locale, `site.languages` is the configured,
deterministically ordered locale list, and `site.default_language` is the
default locale. Locale values expose `key`, `tag`, `label`, `direction`,
`weight`, and `default`.

Pages expose `page.language`, `page.translation_key`, `page.translations`,
`page.all_translations`, and canonical `page.alternates`. `translations`
excludes the current edition; `all_translations` includes it.

Use `translation` for an edition lookup and `localized_url` for a page or known
content route. `localized_url` preserves query strings and fragments. It leaves
external URLs, fragments, and shared assets unchanged, and rejects unknown
internal routes rather than guessing a locale prefix.

```liquid
{% assign german = page | translation: "de" %}
{% if german %}<a href="{{ german.url }}">Deutsch</a>{% endif %}
<a href="{{ "/about/" | localized_url: "de" }}">Über uns</a>
```

Localized builds validate every route before publishing. Output is rendered to
a sibling staging directory and promoted only after the complete generation
succeeds; a failed localized build leaves the previous destination intact.
