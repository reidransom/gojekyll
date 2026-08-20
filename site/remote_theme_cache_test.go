package site

import (
	"archive/tar"
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestRemoteThemeCacheKey(t *testing.T) {
	lower := remoteThemeSpec{Owner: "owner", Repo: "repo", Ref: "0123456789012345678901234567890123456789"}
	upper := remoteThemeSpec{Owner: "OWNER", Repo: "REPO", Ref: "0123456789012345678901234567890123456789"}
	require.Equal(t, remoteThemeCacheKey(lower), remoteThemeCacheKey(upper))
	require.Len(t, remoteThemeCacheKey(lower), 64)
}

func TestRemoteThemeResolverCache(t *testing.T) {
	archive := remoteThemeArchiveBytes(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	resolver := testRemoteThemeResolver(t, server.URL)
	spec := remoteThemeSpec{Owner: "owner", Repo: "repo", Ref: "0123456789012345678901234567890123456789"}
	first, err := resolver.resolve(context.Background(), spec)
	require.NoError(t, err)
	second, err := resolver.resolve(context.Background(), spec)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.EqualValues(t, 1, requests.Load())
	require.True(t, validRemoteThemeCache(first))
}

func TestRemoteThemeResolverCacheConcurrent(t *testing.T) {
	archive := remoteThemeArchiveBytes(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	resolver := testRemoteThemeResolver(t, server.URL)
	spec := remoteThemeSpec{Owner: "owner", Repo: "repo", Ref: "0123456789012345678901234567890123456789"}
	paths := make([]string, 2)
	errs := make([]error, 2)
	var group sync.WaitGroup
	for i := range paths {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			paths[index], errs[index] = resolver.resolve(context.Background(), spec)
		}(i)
	}
	group.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	require.Equal(t, paths[0], paths[1])
	require.EqualValues(t, 1, requests.Load())
}

func TestRemoteThemeResolverCacheFailuresLeaveNoEntry(t *testing.T) {
	spec := remoteThemeSpec{Owner: "owner", Repo: "repo", Ref: "0123456789012345678901234567890123456789"}
	for _, tc := range []struct {
		name    string
		status  int
		archive []byte
	}{
		{name: "failed download", status: http.StatusBadGateway},
		{name: "invalid archive", status: http.StatusOK, archive: []byte("not a gzip archive")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write(tc.archive)
			}))
			defer server.Close()
			resolver := testRemoteThemeResolver(t, server.URL)

			_, err := resolver.resolve(context.Background(), spec)
			require.Error(t, err)
			_, statErr := os.Stat(filepath.Join(resolver.cacheRoot, remoteThemeCacheKey(spec)))
			require.True(t, os.IsNotExist(statErr))
		})
	}
}

func TestRemoteThemeResolverReplacesCorruptCache(t *testing.T) {
	archive := remoteThemeArchiveBytes(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	resolver := testRemoteThemeResolver(t, server.URL)
	spec := remoteThemeSpec{Owner: "owner", Repo: "repo", Ref: "0123456789012345678901234567890123456789"}
	cachePath := filepath.Join(resolver.cacheRoot, remoteThemeCacheKey(spec))
	require.NoError(t, os.MkdirAll(cachePath, 0o755))

	resolved, err := resolver.resolve(context.Background(), spec)
	require.NoError(t, err)
	require.Equal(t, cachePath, resolved)
	require.EqualValues(t, 1, requests.Load())
	require.True(t, validRemoteThemeCache(cachePath))
}

func testRemoteThemeResolver(t *testing.T, archiveURL string) *remoteThemeResolver {
	t.Helper()
	resolver := newRemoteThemeResolver()
	resolver.cacheRoot = filepath.Join(t.TempDir(), "cache")
	resolver.archiveURL = func(remoteThemeSpec) string { return archiveURL }
	resolver.limits = remoteThemeLimits{CompressedBytes: 1 << 20, ExtractedBytes: 1 << 20, FileBytes: 1 << 20, Entries: 100}
	return resolver
}

func remoteThemeArchiveBytes(t *testing.T) []byte {
	t.Helper()
	archivePath := writeRemoteThemeArchive(t, []remoteThemeArchiveEntry{
		{name: "theme/", typeflag: tar.TypeDir},
		{name: "theme/_layouts/default.html", typeflag: tar.TypeReg, body: "layout"},
	})
	archive, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	return archive
}
