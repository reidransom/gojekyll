// Package localization prepares localized document inputs before locale sites
// bind documents to routes, renderers, or cached Liquid drops.
package localization

import (
	"fmt"
	"sort"
	"strings"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/frontmatter"
)

// PagesNamespace is the namespace reserved for standalone pages. Each
// collection name is a separate namespace.
const PagesNamespace = "pages"

// Document is an unbound source document with its effective front matter. It
// deliberately has no site, route, or render state so it can be assigned to a
// locale before those values are initialized.
type Document struct {
	Source       string
	RelativePath string
	Namespace    string
	FrontMatter  frontmatter.FrontMatter
	Static       bool
	Included     bool
}

// Identity identifies a translation set within one content namespace.
type Identity struct {
	Namespace      string
	TranslationKey string
}

// Edition is an included document assigned to a configured locale. Documents
// without a TranslationKey are represented in PreparedInput but never appear
// in the translation catalog.
type Edition struct {
	Document
	Locale         config.Locale
	TranslationKey string
}

// PreparedInput is the locale-specific document input consumed by the later
// project coordinator. Documents are sorted by source path and contain only
// editions assigned to Locale.
type PreparedInput struct {
	Locale    config.Locale
	Documents []Edition
}

// Catalog is the validated collection of published translation editions and
// the prepared, locale-filtered document inputs.
type Catalog struct {
	editions map[Identity]map[string]Edition
	inputs   []PreparedInput
}

// CatalogError reports catalog problems in deterministic order.
type CatalogError struct {
	Problems []string
}

func (e *CatalogError) Error() string {
	return "invalid localization catalog:\n - " + strings.Join(e.Problems, "\n - ")
}

// BuildCatalog assigns included documents to locales, validates their
// translation identities, and produces deterministic locale-filtered inputs.
// Publication eligibility must already be reflected in Document.Included;
// excluded documents are intentionally absent from duplicate and sibling
// checks, as they are absent from the build.
func BuildCatalog(registry *config.LocalizationConfig, documents []Document) (*Catalog, error) {
	if registry == nil {
		return nil, fmt.Errorf("cannot build localization catalog: localization is not enabled")
	}
	if err := registry.Validate(); err != nil {
		return nil, err
	}

	documents = append([]Document(nil), documents...)
	sort.SliceStable(documents, func(i, j int) bool {
		return documentSortKey(documents[i]) < documentSortKey(documents[j])
	})

	catalog := &Catalog{editions: make(map[Identity]map[string]Edition)}
	byLocale := make(map[string][]Edition, len(registry.Locales))
	problems := make([]string, 0)
	for _, document := range documents {
		if document.Static || !document.Included {
			continue
		}

		locale, key, errs := assignDocument(registry, document)
		if len(errs) != 0 {
			problems = append(problems, errs...)
			continue
		}
		edition := Edition{Document: document, Locale: locale, TranslationKey: key}
		byLocale[locale.Key] = append(byLocale[locale.Key], edition)
		if key == "" {
			continue
		}

		identity := Identity{Namespace: document.Namespace, TranslationKey: key}
		set := catalog.editions[identity]
		if set == nil {
			set = make(map[string]Edition)
			catalog.editions[identity] = set
		}
		if existing, found := set[locale.Key]; found {
			problems = append(problems, fmt.Sprintf("%s: namespace %q translation_key %q locale %q has duplicate included editions: %s and %s", documentSource(existing.Document), identity.Namespace, identity.TranslationKey, locale.Key, documentSource(existing.Document), documentSource(document)))
			continue
		}
		set[locale.Key] = edition
	}
	if len(problems) != 0 {
		sort.Strings(problems)
		return nil, &CatalogError{Problems: problems}
	}

	catalog.inputs = make([]PreparedInput, 0, len(registry.Locales))
	for _, locale := range catalogLocales(registry) {
		input := PreparedInput{Locale: locale, Documents: append([]Edition(nil), byLocale[locale.Key]...)}
		sort.Slice(input.Documents, func(i, j int) bool {
			return documentSortKey(input.Documents[i].Document) < documentSortKey(input.Documents[j].Document)
		})
		catalog.inputs = append(catalog.inputs, input)
	}
	return catalog, nil
}

// Editions returns the published sibling editions in catalog locale order. The
// returned slice is independent from the catalog.
func (c *Catalog) Editions(identity Identity) []Edition {
	if c == nil {
		return nil
	}
	set := c.editions[identity]
	if len(set) == 0 {
		return nil
	}
	out := make([]Edition, 0, len(set))
	for _, input := range c.inputs {
		if edition, found := set[input.Locale.Key]; found {
			out = append(out, edition)
		}
	}
	return out
}

// PreparedInputs returns the independent, deterministic locale inputs. A
// caller may retain or modify the returned slices without mutating the catalog.
func (c *Catalog) PreparedInputs() []PreparedInput {
	if c == nil {
		return nil
	}
	out := make([]PreparedInput, len(c.inputs))
	for i, input := range c.inputs {
		out[i] = PreparedInput{Locale: input.Locale, Documents: append([]Edition(nil), input.Documents...)}
	}
	return out
}

// catalogLocales orders locales for build inputs. Locales without an explicit
// weight sort before weighted locales; within each group, the configured
// presentation order is retained. This keeps unweighted locale inputs stable
// without assigning them an implied presentation weight.
func catalogLocales(registry *config.LocalizationConfig) []config.Locale {
	locales := registry.OrderedLocales()
	sort.SliceStable(locales, func(i, j int) bool {
		return locales[i].Weight == nil && locales[j].Weight != nil
	})
	return locales
}

func assignDocument(registry *config.LocalizationConfig, document Document) (config.Locale, string, []string) {
	source := documentSource(document)
	problems := make([]string, 0, 3)
	if document.Namespace == "" {
		problems = append(problems, fmt.Sprintf("%s: document namespace is required", source))
	}

	language := registry.DefaultLanguage
	if raw, found := document.FrontMatter["lang"]; found {
		value, ok := raw.(string)
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: lang must be a string", source))
		} else {
			language = value
		}
	}
	locale, found := registry.Locale(language)
	if !found {
		problems = append(problems, fmt.Sprintf("%s: lang %q does not name a configured locale", source, language))
	}

	translationKey := ""
	if raw, found := document.FrontMatter["translation_key"]; found {
		value, ok := raw.(string)
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("%s: translation_key must be a non-empty string", source))
		case strings.TrimSpace(value) == "":
			problems = append(problems, fmt.Sprintf("%s: translation_key must be a non-empty string", source))
		default:
			translationKey = value
		}
	}
	return locale, translationKey, problems
}

func documentSortKey(document Document) string {
	return documentSource(document) + "\x00" + document.Namespace + "\x00" + document.RelativePath
}

func documentSource(document Document) string {
	if document.Source != "" {
		return document.Source
	}
	return document.RelativePath
}
