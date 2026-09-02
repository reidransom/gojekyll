package pages

import "github.com/reidransom/jigyll/config"

// LocalizationContext supplies one localized page's metadata and related
// editions. A localized site exposes it through LocalizationContext() while
// ordinary sites need not implement that optional interface.
type LocalizationContext interface {
	PageLocalization(Page) (PageLocalization, bool)
}

// LocalizedSite is the optional extension a localized site provides to page
// drops. Keeping it separate from Site preserves the ordinary site contract.
type LocalizedSite interface {
	LocalizationContext() LocalizationContext
}

// PageLocalization is the structured locale and translation metadata exposed
// by a page drop. Translation slices must already be in configured locale
// order; AllTranslations includes the current page.
type PageLocalization struct {
	Language        config.Locale
	TranslationKey  string
	Translations    []Page
	AllTranslations []Page
	Alternates      []Alternate
}

// Alternate is one canonical alternate-link record for a published edition.
// XDefault is true only for an existing default-locale edition.
type Alternate struct {
	Language config.Locale
	URL      string
	XDefault bool
}

func pageLocalization(site Site, page Page) (PageLocalization, bool) {
	localized, ok := site.(LocalizedSite)
	if !ok {
		return PageLocalization{}, false
	}
	context := localized.LocalizationContext()
	if context == nil {
		return PageLocalization{}, false
	}
	return context.PageLocalization(page)
}

func localeDrop(locale config.Locale) map[string]interface{} {
	return map[string]interface{}{
		"key":       locale.Key,
		"tag":       locale.Tag,
		"label":     locale.Label,
		"direction": locale.Direction,
		"weight":    locale.Weight,
		"default":   locale.Default,
	}
}

func alternateDrop(alternate Alternate) map[string]interface{} {
	return map[string]interface{}{
		"tag":       alternate.Language.Tag,
		"url":       alternate.URL,
		"x_default": alternate.XDefault,
	}
}
