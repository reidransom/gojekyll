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
	activePolicy := len(registry.RequiredTranslations) != 0
	var records []documentRecord
	if activePolicy {
		records = make([]documentRecord, 0, len(documents))
	}
	for _, document := range documents {
		if document.Static || !document.Included {
			continue
		}

		assignment := assignDocument(registry, document)
		problems = append(problems, assignment.problems...)
		if activePolicy {
			valid, errs := validateExemptions(registry, document, assignment)
			records = append(records, documentRecord{
				document:   document,
				assignment: assignment,
				exemptions: valid,
			})
			problems = append(problems, errs...)
		}
		if len(assignment.problems) != 0 {
			continue
		}

		edition := Edition{Document: document, Locale: assignment.locale, TranslationKey: assignment.translationKey}
		byLocale[assignment.locale.Key] = append(byLocale[assignment.locale.Key], edition)
		if assignment.translationKey == "" {
			continue
		}

		identity := Identity{Namespace: document.Namespace, TranslationKey: assignment.translationKey}
		set := catalog.editions[identity]
		if set == nil {
			set = make(map[string]Edition)
			catalog.editions[identity] = set
		}
		if existing, found := set[assignment.locale.Key]; found {
			problems = append(problems, fmt.Sprintf("%s: namespace %q translation_key %q locale %q has duplicate included editions: %s and %s", documentSource(existing.Document), identity.Namespace, identity.TranslationKey, assignment.locale.Key, documentSource(existing.Document), documentSource(document)))
			continue
		}
		set[assignment.locale.Key] = edition
	}
	if activePolicy {
		problems = append(problems, requiredTranslationProblems(registry, catalog.editions, records)...)
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

type documentAssignment struct {
	locale         config.Locale
	translationKey string
	validLocale    bool
	problems       []string
}

type documentRecord struct {
	document   Document
	assignment documentAssignment
	exemptions map[string]struct{}
}

func assignDocument(registry *config.LocalizationConfig, document Document) documentAssignment {
	source := documentSource(document)
	assignment := documentAssignment{}
	if document.Namespace == "" {
		assignment.problems = append(assignment.problems, fmt.Sprintf("%s: document namespace is required", source))
	}

	language := registry.DefaultLanguage
	langIsString := true
	if raw, found := document.FrontMatter["lang"]; found {
		value, ok := raw.(string)
		if !ok {
			assignment.problems = append(assignment.problems, fmt.Sprintf("%s: lang must be a string", source))
			langIsString = false
		} else {
			language = value
		}
	}
	locale, found := registry.Locale(language)
	if !found {
		assignment.problems = append(assignment.problems, fmt.Sprintf("%s: lang %q does not name a configured locale", source, language))
	} else if langIsString {
		assignment.locale = locale
		assignment.validLocale = true
	}

	if raw, found := document.FrontMatter["translation_key"]; found {
		value, ok := raw.(string)
		switch {
		case !ok:
			assignment.problems = append(assignment.problems, fmt.Sprintf("%s: translation_key must be a non-empty string", source))
		case strings.TrimSpace(value) == "":
			assignment.problems = append(assignment.problems, fmt.Sprintf("%s: translation_key must be a non-empty string", source))
		default:
			assignment.translationKey = value
		}
	}
	return assignment
}

func validateExemptions(registry *config.LocalizationConfig, document Document, assignment documentAssignment) (map[string]struct{}, []string) {
	raw, found := document.FrontMatter["translation_exempt"]
	if !found {
		return nil, nil
	}

	wrongOwner := assignment.validLocale && assignment.locale.Key != registry.DefaultLanguage
	problems := make([]string, 0)
	if wrongOwner {
		problems = append(problems, fmt.Sprintf("%s: translation_exempt is only allowed on default locale %q editions", documentSource(document), registry.DefaultLanguage))
	}

	var values []interface{}
	switch value := raw.(type) {
	case []interface{}:
		values = value
	case []string:
		values = make([]interface{}, len(value))
		for index, locale := range value {
			values[index] = locale
		}
	default:
		return nil, append(problems, fmt.Sprintf("%s: translation_exempt must be a sequence of locale-key strings", documentSource(document)))
	}

	valid := make(map[string]struct{})
	counts := make(map[string]int, len(values))
	for _, rawLocale := range values {
		if locale, ok := rawLocale.(string); ok {
			counts[locale]++
		}
	}
	seen := make(map[string]struct{}, len(counts))
	for index, rawLocale := range values {
		locale, ok := rawLocale.(string)
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: translation_exempt[%d] must be a locale-key string", documentSource(document), index))
			continue
		}
		if _, duplicate := seen[locale]; duplicate {
			problems = append(problems, fmt.Sprintf("%s: translation_exempt[%d]: duplicate locale %q", documentSource(document), index, locale))
			continue
		}
		seen[locale] = struct{}{}
		if _, exists := registry.Locales[locale]; !exists {
			problems = append(problems, fmt.Sprintf("%s: translation_exempt[%d]: unknown locale %q", documentSource(document), index, locale))
			continue
		}
		if !isRequiredLocale(registry.RequiredTranslations, locale) {
			problems = append(problems, fmt.Sprintf("%s: translation_exempt[%d]: locale %q is not required", documentSource(document), index, locale))
			continue
		}
		if counts[locale] > 1 || !assignment.validLocale || wrongOwner {
			continue
		}
		valid[locale] = struct{}{}
	}
	return valid, problems
}

func requiredTranslationProblems(registry *config.LocalizationConfig, editions map[Identity]map[string]Edition, records []documentRecord) []string {
	problems := make([]string, 0)
	reportedMissing := make(map[Identity]map[string]struct{})
	for _, record := range records {
		document := record.document
		assignment := record.assignment
		if len(assignment.problems) != 0 || assignment.locale.Key != registry.DefaultLanguage {
			continue
		}
		exempt := record.exemptions
		if assignment.translationKey == "" {
			for _, locale := range registry.RequiredTranslations {
				if _, covered := exempt[locale]; !covered {
					problems = append(problems, fmt.Sprintf("%s: default-locale document requires a translation_key or translation_exempt covering required locale %q", documentSource(document), locale))
				}
			}
			continue
		}

		identity := Identity{Namespace: document.Namespace, TranslationKey: assignment.translationKey}
		set := editions[identity]
		for _, locale := range registry.RequiredTranslations {
			if _, exempted := exempt[locale]; exempted {
				if _, found := set[locale]; found {
					problems = append(problems, fmt.Sprintf("%s: translation_exempt for required locale %q is redundant because namespace %q translation_key %q has an included edition", documentSource(document), locale, identity.Namespace, identity.TranslationKey))
				}
				continue
			}
			if _, found := set[locale]; !found {
				reported := reportedMissing[identity]
				if reported == nil {
					reported = make(map[string]struct{})
					reportedMissing[identity] = reported
				}
				if _, alreadyReported := reported[locale]; alreadyReported {
					continue
				}
				reported[locale] = struct{}{}
				problems = append(problems, fmt.Sprintf("namespace %q translation_key %q is missing required locale %q", identity.Namespace, identity.TranslationKey, locale))
			}
		}
	}
	return problems
}

func isRequiredLocale(required []string, locale string) bool {
	for _, candidate := range required {
		if candidate == locale {
			return true
		}
	}
	return false
}

func documentSortKey(document Document) string {
	return documentSource(document) + "\x00" + document.Namespace + "\x00" + document.RelativePath
}

func documentSource(document Document) string {
	if document.RelativePath != "" {
		return document.RelativePath
	}
	return document.Source
}
