package tags

import (
	"maps"
	"reflect"
	"sync"

	"github.com/reidransom/liquid/render"
)

type includeCacheEntry struct {
	include map[string]interface{}
	output  string
}

type includeCache struct {
	mu      sync.RWMutex
	entries map[string][]includeCacheEntry
}

func newIncludeCache() *includeCache {
	return &includeCache{entries: make(map[string][]includeCacheEntry)}
}

func (c *includeCache) lookup(filename string, include map[string]interface{}) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return findCachedInclude(c.entries[filename], include)
}

func (c *includeCache) store(filename string, include map[string]interface{}, output string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := findCachedInclude(c.entries[filename], include); ok {
		return cached
	}
	c.entries[filename] = append(c.entries[filename], includeCacheEntry{
		include: maps.Clone(include),
		output:  output,
	})
	return output
}

func findCachedInclude(entries []includeCacheEntry, include map[string]interface{}) (string, bool) {
	for _, entry := range entries {
		if reflect.DeepEqual(entry.include, include) {
			return entry.output, true
		}
	}
	return "", false
}

func (tc tagContext) includeCachedTag(rc render.Context) (s string, err error) {
	request, err := parseIncludeRequest(rc)
	if err != nil {
		return "", err
	}
	for _, dir := range tc.includeDirs {
		var resolved resolvedInclude
		resolved, err = request.resolve(dir, rc)
		if err != nil {
			continue
		}
		if s, ok := tc.includeCache.lookup(resolved.filename, request.include); ok {
			return s, nil
		}
		s, err = resolved.render(rc)
		if err == nil {
			return tc.includeCache.store(resolved.filename, request.include, s), nil
		}
	}
	return
}
