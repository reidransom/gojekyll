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

// watchReload observes one source watcher. Localized projects rebuild and
// replace their complete aggregate snapshot for every event.
func (s *Server) watchReload() error {
	s.m.RLock()
	project := s.project
	base := s.Site
	s.m.RUnlock()
	var (
		changes <-chan site.FilesEvent
		err     error
	)
	if project != nil {
		changes, err = project.WatchFiles()
	} else {
		changes, err = base.WatchFiles()
	}
	if err != nil {
		return err
	}
	go func() {
		for change := range changes {
			batch := s.liveReloadBatch(change)
			if s.reload(change) {
				batch.Deliver()
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

// liveReloadBatch captures the intent for one watch event while its source
// paths still resolve against the snapshot that observed the event.
func (s *Server) liveReloadBatch(change site.FilesEvent) liveReloadBatch {
	s.m.RLock()
	defer s.m.RUnlock()
	if s.project != nil {
		return liveReloadBatch{
			transport: s.liveReload,
			intent:    liveReloadIntent{pageReload: true},
		}
	}
	return liveReloadBatch{
		transport: s.liveReload,
		intent:    newLiveReloadIntent(s.Site, change.Paths),
	}
}

type liveReloadBatch struct {
	transport *liveReloadTransport
	intent    liveReloadIntent
}

func (b liveReloadBatch) Deliver() {
	if b.transport != nil {
		b.transport.Deliver(b.intent)
	}
}

func isLiveReloadResource(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	return extension == ".css" || strings.HasPrefix(mime.TypeByExtension(extension), "image/")
}

func (s *Server) reload(change site.FilesEvent) bool {
	s.m.RLock()
	project := s.project
	current := s.Site
	liveReload := s.liveReload
	s.m.RUnlock()

	fmt.Printf("Re-reading: %v %v...\n", change, change.Paths)
	start := time.Now()
	if project != nil {
		replacement, _, err := project.Rebuild()
		if err != nil {
			s.reportReloadError(liveReload, err)
			return false
		}
		s.m.Lock()
		s.project = replacement
		s.m.Unlock()
		fmt.Printf("done (%.2fs)\n", time.Since(start).Seconds())
		return true
	}

	replacement, err := current.Reloaded(change.Paths)
	if err != nil {
		s.reportReloadError(liveReload, err)
		return false
	}
	s.m.Lock()
	s.Site = replacement
	s.m.Unlock()
	// Only clear URL if JEKYLL_URL is not set
	if jekyllURL := os.Getenv("JEKYLL_URL"); jekyllURL != "" {
		replacement.SetAbsoluteURL(jekyllURL)
	} else {
		replacement.SetAbsoluteURL("")
	}
	fmt.Printf("done (%.2fs)\n", time.Since(start).Seconds())
	return true
}

func (s *Server) reportReloadError(liveReload *liveReloadTransport, err error) {
	fmt.Println()
	fmt.Fprintln(os.Stderr, err.Error())
	if liveReload != nil {
		liveReload.Alert(fmt.Sprintf("Error reading site configuration: %s", err))
	}
}
