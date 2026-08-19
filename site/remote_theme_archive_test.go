package site

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type remoteThemeArchiveEntry struct {
	name     string
	typeflag byte
	body     string
	linkname string
}

func TestExtractRemoteThemeArchive(t *testing.T) {
	t.Run("extracts a theme beneath its top-level directory", func(t *testing.T) {
		archivePath := writeRemoteThemeArchive(t, []remoteThemeArchiveEntry{
			{name: "theme-sha/", typeflag: tar.TypeDir},
			{name: "theme-sha/_layouts/default.html", typeflag: tar.TypeReg, body: "theme layout"},
			{name: "theme-sha/_includes/head.html", typeflag: tar.TypeReg, body: "head"},
		})
		destination := filepath.Join(t.TempDir(), "theme")

		require.NoError(t, extractRemoteThemeArchive(archivePath, destination, remoteThemeLimits{Entries: 10, FileBytes: 100, ExtractedBytes: 100}))
		contents, err := os.ReadFile(filepath.Join(destination, "_layouts", "default.html"))
		require.NoError(t, err)
		require.Equal(t, "theme layout", string(contents))
	})

	for _, tc := range []struct {
		name  string
		entry remoteThemeArchiveEntry
	}{
		{name: "absolute path", entry: remoteThemeArchiveEntry{name: "/outside", typeflag: tar.TypeReg}},
		{name: "traversal path", entry: remoteThemeArchiveEntry{name: "theme/../../outside", typeflag: tar.TypeReg}},
		{name: "Windows path", entry: remoteThemeArchiveEntry{name: "theme\\outside", typeflag: tar.TypeReg}},
		{name: "symbolic link", entry: remoteThemeArchiveEntry{name: "theme/link", typeflag: tar.TypeSymlink, linkname: "target"}},
		{name: "hard link", entry: remoteThemeArchiveEntry{name: "theme/link", typeflag: tar.TypeLink, linkname: "target"}},
		{name: "device", entry: remoteThemeArchiveEntry{name: "theme/device", typeflag: tar.TypeChar}},
	} {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			archivePath := writeRemoteThemeArchive(t, []remoteThemeArchiveEntry{tc.entry})
			destination := filepath.Join(t.TempDir(), "theme")
			err := extractRemoteThemeArchive(archivePath, destination, remoteThemeLimits{Entries: 10, FileBytes: 100, ExtractedBytes: 100})
			require.Error(t, err)
			_, statErr := os.Stat(destination)
			require.True(t, os.IsNotExist(statErr))
		})
	}

	t.Run("enforces extraction limits", func(t *testing.T) {
		archivePath := writeRemoteThemeArchive(t, []remoteThemeArchiveEntry{{name: "theme/file", typeflag: tar.TypeReg, body: "too large"}})
		destination := filepath.Join(t.TempDir(), "theme")
		err := extractRemoteThemeArchive(archivePath, destination, remoteThemeLimits{Entries: 10, FileBytes: 3, ExtractedBytes: 100})
		require.ErrorContains(t, err, "exceeds 3 bytes")
		_, statErr := os.Stat(destination)
		require.True(t, os.IsNotExist(statErr))
	})
}

func TestRemoteThemeDownloadArchive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/archive", r.URL.Path)
		_, _ = w.Write([]byte("archive"))
	}))
	defer server.Close()

	resolver := newRemoteThemeResolver()
	resolver.archiveURL = func(remoteThemeSpec) string { return server.URL + "/archive" }
	resolver.limits.CompressedBytes = 10
	destination := filepath.Join(t.TempDir(), "archive.tgz")

	require.NoError(t, resolver.downloadArchive(context.Background(), remoteThemeSpec{}, destination))
	contents, err := os.ReadFile(destination)
	require.NoError(t, err)
	require.Equal(t, "archive", string(contents))
}

func writeRemoteThemeArchive(t *testing.T, entries []remoteThemeArchiveEntry) string {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := &tar.Header{Name: entry.name, Typeflag: entry.typeflag, Size: int64(len(entry.body)), Linkname: entry.linkname}
		require.NoError(t, tarWriter.WriteHeader(header))
		if entry.body != "" {
			_, err := tarWriter.Write([]byte(entry.body))
			require.NoError(t, err)
		}
	}
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())

	archivePath := filepath.Join(t.TempDir(), "theme.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, buffer.Bytes(), 0o600))
	return archivePath
}
