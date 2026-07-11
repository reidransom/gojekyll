package site

import (
	"github.com/reidransom/jigyll/collection"
)

func (s *Site) findPostCollection() *collection.Collection {
	for _, c := range s.Collections {
		if c.Name == "posts" {
			return c
		}
	}
	return nil
}

func (s *Site) setPostVariables() {
	c := s.findPostCollection()
	if c == nil {
		return
	}
	var (
		ps      = c.Pages()
		related = ps
	)
	if len(related) > 10 {
		related = related[:10]
	}
	s.drop["categories"] = groupPagesBy(ps, func(p Page) []string { return p.Categories() })
	s.drop["tags"] = groupPagesBy(ps, func(p Page) []string { return p.Tags() })
	s.drop["related_posts"] = related
}

func groupPagesBy(ps []Page, getter func(Page) []string) map[string][]Page {
	groups := map[string][]Page{}
	for _, p := range ps {
		for _, k := range getter(p) {
			groups[k] = append(groups[k], p)
		}
	}
	return groups
}
