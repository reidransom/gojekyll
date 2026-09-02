package localization

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/frontmatter"
)

// DiscoverDocument reads a source document without constructing a page. It
// applies the same front-matter-default lookup used by ordinary site discovery
// before returning the document for locale assignment. namespace is
// PagesNamespace for standalone pages and the collection name for collection
// documents; typename is the existing defaults type ("pages" or a collection
// name).
//
// The returned Included value applies the current unpublished setting. A
// caller that filters drafts or future posts must set Included to false before
// BuildCatalog; doing so guarantees those editions cannot create duplicate
// catalog identities or appear in prepared inputs.
func DiscoverDocument(cfg *config.Config, source, relativePath, namespace, typename string) (Document, error) {
	if cfg == nil {
		return Document{}, fmt.Errorf("discovering %s: configuration is required", source)
	}
	relativePath = filepath.ToSlash(relativePath)
	hasFrontMatter, err := frontmatter.FileHasFrontMatter(source)
	if err != nil {
		return Document{}, fmt.Errorf("discovering %s: %w", source, err)
	}

	static := !hasFrontMatter && cfg.RequiresFrontMatter(relativePath)
	defaultsType := typename
	if static {
		// Static files are untyped, matching pages.NewFile.
		defaultsType = ""
	}
	effective := frontmatter.FrontMatter(cfg.GetFrontMatterDefaults(defaultsType, relativePath))
	if hasFrontMatter {
		contents, err := os.ReadFile(source)
		if err != nil {
			return Document{}, fmt.Errorf("discovering %s: %w", source, err)
		}
		raw, err := frontmatter.Read(&contents, nil)
		if err != nil {
			return Document{}, fmt.Errorf("discovering %s: %w", source, err)
		}
		effective = effective.Merged(raw)
	}

	return Document{
		Source:       source,
		RelativePath: relativePath,
		Namespace:    namespace,
		FrontMatter:  effective,
		Static:       static,
		Included:     !static && (effective.Bool("published", true) || cfg.Unpublished),
	}, nil
}
