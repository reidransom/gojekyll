package site

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/localization"
	"github.com/reidransom/jigyll/pages"
	"github.com/reidransom/jigyll/utils"
)

// LocalizedBuild enters the project-level production seam for an opt-in
// localized site. It prepares isolated locale sites, renders them serially to
// one sibling staging tree, and promotes that complete generation only after
// every locale has succeeded.
//
// Callers must use Site.Write for a non-localized site. Keeping that path
// separate preserves existing single-site lifecycle behavior.
func LocalizedBuild(base *Site) (int, error) {
	if base == nil || !base.cfg.Enabled() {
		return 0, fmt.Errorf("localized build requires localization configuration")
	}
	project, err := newLocalizationProject(base)
	if err != nil {
		return 0, err
	}
	return project.Build()
}

type localizationProject struct {
	base     *Site
	catalog  *localization.Catalog
	data     *localization.DataCatalog
	locales  []config.Locale
	prepared []*Site
}

func newLocalizationProject(base *Site) (*localizationProject, error) {
	if err := base.cfg.Localization.Validate(); err != nil {
		return nil, err
	}
	if base.cfg.Incremental {
		return nil, fmt.Errorf("localized builds do not support incremental mode")
	}
	documents, err := discoverLocalizedDocuments(base)
	if err != nil {
		return nil, err
	}
	catalog, err := localization.BuildCatalog(base.cfg.Localization, documents)
	if err != nil {
		return nil, err
	}
	data, err := localization.DiscoverData(filepath.Join(base.SourceDir(), base.cfg.DataDir), base.cfg.Localization)
	if err != nil {
		return nil, err
	}
	return &localizationProject{
		base:    base,
		catalog: catalog,
		data:    data,
		locales: base.cfg.Localization.OrderedLocales(),
	}, nil
}

// Build constructs locale sites in deterministic order before rendering any
// of them. Every prepared site has its own config, data map, pages,
// collections, routes, renderer, and plugin instances. Static files are read
// only by the first locale site and therefore retain project-root URLs.
func (p *localizationProject) Build() (int, error) {
	stage, err := os.MkdirTemp(filepath.Dir(p.base.DestDir()), ".jigyll-localized-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(stage)

	if err := copyKeepFiles(p.base.DestDir(), stage, p.base.cfg.KeepFiles); err != nil {
		return 0, err
	}
	inputs := make(map[string]map[string]struct{}, len(p.locales))
	for _, input := range p.catalog.PreparedInputs() {
		sources := make(map[string]struct{}, len(input.Documents))
		for _, edition := range input.Documents {
			sources[edition.Source] = struct{}{}
		}
		inputs[input.Locale.Key] = sources
	}
	for index, locale := range p.locales {
		localeData, err := p.data.Data(locale.Key)
		if err != nil {
			return 0, err
		}
		derived, err := p.base.cfg.DeriveLocale(locale.Key)
		if err != nil {
			return 0, err
		}
		messages, err := p.data.Messages(locale.Key)
		if err != nil {
			return 0, err
		}
		derived.Destination = stage
		site := New(p.base.flags)
		site.cfg = derived
		site.localeKey = locale.Key
		site.localePrefix = localeRoutePrefix(p.base.cfg.Localization, locale)
		site.includeStatic = index == 0
		site.data = localeData
		site.localizedSources = inputs[locale.Key]
		site.localizationContext = &localizedSiteContext{
			site:         site,
			locale:       locale,
			registry:     p.base.cfg.Localization,
			messages:     messages,
			pageInfo:     make(map[pages.Page]localizedPageInfo),
			routePages:   make(map[string]pages.Page),
			sharedAssets: make(map[string]struct{}),
		}
		if err := site.Read(); err != nil {
			return 0, fmt.Errorf("preparing locale %q: %w", locale.Key, err)
		}
		p.prepared = append(p.prepared, site)
	}
	if err := validateLocalizedRoutes(p.prepared); err != nil {
		return 0, err
	}
	if err := p.bindLocalizationContexts(); err != nil {
		return 0, err
	}

	count := 0
	for _, site := range p.prepared {
		if err := site.setTimeZone(); err != nil {
			return count, err
		}
		if err := site.ensureRendered(); err != nil {
			return count, fmt.Errorf("rendering locale %q: %w", site.localeKey, err)
		}
		for _, document := range site.OutputDocs() {
			count++
			if site.cfg.DryRun {
				continue
			}
			if err := site.WriteDoc(document); err != nil {
				return count, fmt.Errorf("writing locale %q: %w", site.localeKey, err)
			}
		}
	}
	if p.base.cfg.DryRun {
		return count, nil
	}
	if err := promoteLocalizedGeneration(stage, p.base.DestDir()); err != nil {
		return count, err
	}
	return count, nil
}

func (p *localizationProject) bindLocalizationContexts() error {
	editionsBySource := map[string]localization.Edition{}
	for _, input := range p.catalog.PreparedInputs() {
		for _, edition := range input.Documents {
			editionsBySource[edition.Source] = edition
		}
	}

	groups := map[localization.Identity]map[string]pages.Page{}
	allPages := make(map[*Site][]pages.Page, len(p.prepared))
	for _, site := range p.prepared {
		allPages[site] = site.Pages()
		for _, page := range allPages[site] {
			edition, found := editionsBySource[page.Source()]
			if !found || edition.TranslationKey == "" {
				continue
			}
			identity := localization.Identity{Namespace: edition.Namespace, TranslationKey: edition.TranslationKey}
			if groups[identity] == nil {
				groups[identity] = map[string]pages.Page{}
			}
			groups[identity][edition.Locale.Key] = page
		}
	}
	sharedAssets := map[string]struct{}{}
	for _, site := range p.prepared {
		for _, document := range site.OutputDocs() {
			if document.IsStatic() {
				sharedAssets[document.URL()] = struct{}{}
			}
		}
	}

	for _, site := range p.prepared {
		context := site.localizationContext
		for _, candidateSite := range p.prepared {
			for _, page := range allPages[candidateSite] {
				edition := editionsBySource[page.Source()]
				identity := localization.Identity{Namespace: edition.Namespace, TranslationKey: edition.TranslationKey}
				all := map[string]pages.Page(nil)
				if identity.TranslationKey != "" {
					all = groups[identity]
				}
				locale, found := p.base.cfg.Localization.Locale(candidateSite.localeKey)
				if !found {
					return fmt.Errorf("prepared locale %q is not configured", candidateSite.localeKey)
				}
				context.pageInfo[page] = localizedPageInfo{identity: identity, locale: locale, page: page, all: all}
			}
		}
		for _, page := range allPages[site] {
			context.routePages[page.URL()] = page
			context.routePages[localeRelativeRoute(site.localePrefix, page.URL())] = page
		}
		context.sharedAssets = sharedAssets
	}
	return nil
}

func localeRoutePrefix(localizationConfig *config.LocalizationConfig, locale config.Locale) string {
	if locale.Default && !localizationConfig.DefaultLanguageInSubdir {
		return ""
	}
	return locale.Key
}

func discoverLocalizedDocuments(site *Site) ([]localization.Document, error) {
	var documents []localization.Document
	discover := func(source, relativePath, namespace, typename string) error {
		document, err := localization.DiscoverDocument(&site.cfg, source, relativePath, namespace, typename)
		if err != nil {
			return err
		}
		documents = append(documents, document)
		return nil
	}
	if err := filepath.Walk(site.SourceDir(), func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(site.SourceDir(), filename)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		if info.IsDir() {
			if relativePath != "." && (site.Exclude(relativePath) || strings.HasPrefix(filepath.Base(relativePath), "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if site.Exclude(relativePath) || strings.HasPrefix(relativePath, "_") {
			return nil
		}
		return discover(filename, relativePath, localization.PagesNamespace, "pages")
	}); err != nil {
		return nil, fmt.Errorf("discovering localized pages: %w", err)
	}

	collectionNames := make([]string, 0, len(site.cfg.Collections))
	for name := range site.cfg.Collections {
		collectionNames = append(collectionNames, name)
	}
	sort.Strings(collectionNames)
	for _, name := range collectionNames {
		directory := filepath.Join(site.SourceDir(), "_"+name)
		err := filepath.Walk(directory, func(filename string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if info.IsDir() {
				return nil
			}
			relativePath, err := filepath.Rel(site.SourceDir(), filename)
			if err != nil {
				return err
			}
			relativePath = filepath.ToSlash(relativePath)
			if site.Exclude(relativePath) {
				return nil
			}
			return discover(filename, relativePath, name, name)
		})
		if err != nil {
			return nil, fmt.Errorf("discovering localized collection %q: %w", name, err)
		}
	}
	return documents, nil
}

func validateLocalizedRoutes(sites []*Site) error {
	routes := make(map[string]string)
	destinations := make(map[string]string)
	for _, site := range sites {
		for _, document := range site.OutputDocs() {
			route := document.URL()
			owner := fmt.Sprintf("locale %q source %q", site.localeKey, document.Source())
			if previous, exists := routes[route]; exists {
				return fmt.Errorf("localized route collision at %q between %s and %s", route, previous, owner)
			}
			routes[route] = owner
			destination := filepath.Clean(destinationRelativePath(document))
			if previous, exists := destinations[destination]; exists {
				return fmt.Errorf("localized destination collision at %q between %s and %s", destination, previous, owner)
			}
			destinations[destination] = owner
		}
	}
	return nil
}

func copyKeepFiles(source, destination string, keepFiles []string) error {
	for _, keep := range keepFiles {
		from := filepath.Join(source, keep)
		info, err := os.Stat(from)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if err := copyLocalizedPath(from, filepath.Join(destination, keep), info); err != nil {
			return err
		}
	}
	return nil
}

func copyLocalizedPath(source, destination string, info os.FileInfo) error {
	if !info.IsDir() {
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		return utils.CopyFileContents(destination, source, info.Mode())
	}
	return filepath.Walk(source, func(path string, entry os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, entry.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return utils.CopyFileContents(target, path, entry.Mode())
	})
}

func promoteLocalizedGeneration(stage, destination string) error {
	backup := destination + ".jigyll-localized-backup"
	if err := os.RemoveAll(backup); err != nil {
		return err
	}
	destinationExists := false
	if _, err := os.Stat(destination); err == nil {
		destinationExists = true
		if err := os.Rename(destination, backup); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(stage, destination); err != nil {
		if destinationExists {
			_ = os.Rename(backup, destination)
		}
		return err
	}
	if destinationExists {
		return os.RemoveAll(backup)
	}
	return nil
}
