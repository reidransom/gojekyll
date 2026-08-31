package server

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/reidransom/jigyll/site"
)

// Create a goroutine that rebuilds the site when files change.
func (s *Server) watchReload() error {
	// Note: reload swaps in a new site but the watcher still uses the old
	// site's include/exclude configuration until the server restarts.
	changes, err := s.Site.WatchFiles()
	if err != nil {
		return err
	}
	go func() {
		for change := range changes {
			// Get current site reference with lock protection
			s.m.Lock()
			site := s.Site
			s.m.Unlock()

			// Resolve an entire watch batch before rebuilding, while its source
			// paths still match the current site's routes.
			intent := newLiveReloadIntent(site, change.Paths)
			// reload the site
			s.reload(change)
			// Deliver the already-built batch intent after the new site is ready.
			if liveReload := s.currentLiveReloader(); liveReload != nil {
				liveReload.Deliver(intent)
			}
		}
	}()
	return nil
}

// liveReloadIntent describes the minimum work a client must perform for one
// watch batch. A page reload always takes precedence over resource updates.
type liveReloadIntent struct {
	pageReload    bool
	resourcePaths []string
}

func newLiveReloadIntent(s *site.Site, paths []string) liveReloadIntent {
	if s.RequiresFullReload(paths) {
		return liveReloadIntent{pageReload: true}
	}

	var (
		intent liveReloadIntent
		seen   map[string]struct{}
	)
	for _, path := range paths {
		resourcePath, found := s.FilenameURLPath(path)
		if !found {
			continue
		}
		if !isLiveReloadResource(resourcePath) {
			return liveReloadIntent{pageReload: true}
		}
		if seen == nil {
			seen = make(map[string]struct{}, len(paths))
		}
		if _, duplicate := seen[resourcePath]; duplicate {
			continue
		}
		seen[resourcePath] = struct{}{}
		intent.resourcePaths = append(intent.resourcePaths, resourcePath)
	}
	return intent
}

func isLiveReloadResource(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	return extension == ".css" || strings.HasPrefix(mime.TypeByExtension(extension), "image/")
}

func (s *Server) reload(change site.FilesEvent) {
	s.m.Lock()
	defer s.m.Unlock()

	// similar code to site.WatchRebuild
	fmt.Printf("Re-reading: %v %v...\n", change, change.Paths)
	start := time.Now()
	site, err := s.Site.Reloaded(change.Paths)
	if err != nil {
		fmt.Println()
		fmt.Fprintln(os.Stderr, err.Error())
		if liveReload := s.currentLiveReloader(); liveReload != nil {
			liveReload.Alert(fmt.Sprintf("Error reading site configuration: %s", err))
		}
		return
	}
	s.Site = site
	// Only clear URL if JEKYLL_URL is not set
	if jekyllURL := os.Getenv("JEKYLL_URL"); jekyllURL != "" {
		s.Site.SetAbsoluteURL(jekyllURL)
	} else {
		s.Site.SetAbsoluteURL("")
	}
	fmt.Printf("done (%.2fs)\n", time.Since(start).Seconds())
}
