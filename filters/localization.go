package filters

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/reidransom/liquid"
)

// LocalizationContext supplies locale-specific lookups to Liquid filters. A
// localized site must provide this context while its renderer is initialized,
// before any page drops can be cached.
type LocalizationContext interface {
	// ActiveLocale returns the locale key used when a filter omits its target.
	ActiveLocale() string
	// Translation returns the published edition for page and targetLocale, or
	// nil when that translation set has no such edition.
	Translation(page interface{}, targetLocale string) (interface{}, error)
	// LocalizedPageURL resolves page through its translation identity.
	LocalizedPageURL(page interface{}, targetLocale string) (string, error)
	// LocalizedRouteURL resolves a known, locale-relative content route.
	LocalizedRouteURL(route string, targetLocale string) (string, error)
	// IsSharedAsset reports whether route belongs to the project's shared asset
	// set and must therefore remain unprefixed.
	IsSharedAsset(route string) bool
	// Translate resolves an interface-message key for the active locale.
	Translate(key string) (string, error)
}

// AddLocalizationFilters registers filters whose behavior depends on a
// localized project. It intentionally leaves existing URL filters unchanged.
func AddLocalizationFilters(e *liquid.Engine, context LocalizationContext) {
	if context == nil {
		return
	}
	e.RegisterFilter("translation", func(page interface{}, target ...string) (interface{}, error) {
		locale, err := filterLocale(context, target)
		if err != nil {
			return nil, err
		}
		return context.Translation(page, locale)
	})
	e.RegisterFilter("localized_url", func(value interface{}, target ...string) (string, error) {
		locale, err := filterLocale(context, target)
		if err != nil {
			return "", err
		}
		return localizedURL(context, value, locale)
	})
	e.RegisterFilter("translate", func(key string) (string, error) {
		return context.Translate(key)
	})
}

func filterLocale(context LocalizationContext, target []string) (string, error) {
	switch len(target) {
	case 0:
		return context.ActiveLocale(), nil
	case 1:
		return target[0], nil
	default:
		return "", fmt.Errorf("localized filter accepts at most one target locale")
	}
}

func localizedURL(context LocalizationContext, value interface{}, targetLocale string) (string, error) {
	route, ok := value.(string)
	if !ok {
		return context.LocalizedPageURL(value, targetLocale)
	}
	if strings.HasPrefix(route, "#") || strings.HasPrefix(route, "//") {
		return route, nil
	}
	parsed, err := url.Parse(route)
	if err != nil {
		return "", err
	}
	if parsed.IsAbs() {
		return route, nil
	}
	if context.IsSharedAsset(parsed.Path) {
		return route, nil
	}
	localized, err := context.LocalizedRouteURL(parsed.Path, targetLocale)
	if err != nil {
		return "", err
	}
	return appendURLSuffix(localized, parsed.RawQuery, parsed.Fragment), nil
}

func appendURLSuffix(route, rawQuery, fragment string) string {
	if rawQuery != "" {
		route += "?" + rawQuery
	}
	if fragment != "" {
		route += "#" + fragment
	}
	return route
}
