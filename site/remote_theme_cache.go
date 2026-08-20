package site

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (r *remoteThemeResolver) resolve(ctx context.Context, spec remoteThemeSpec) (string, error) {
	cacheRoot, err := r.themeCacheRoot()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return "", fmt.Errorf("create remote theme cache: %w", err)
	}

	key := remoteThemeCacheKey(spec)
	cachePath := filepath.Join(cacheRoot, key)
	lockPath := filepath.Join(cacheRoot, key+".lock")
	var resolved string
	if err := withRemoteThemeLock(lockPath, func() error {
		if validRemoteThemeCache(cachePath) {
			resolved = cachePath
			return nil
		}
		if err := os.RemoveAll(cachePath); err != nil {
			return err
		}
		return r.materializeRemoteTheme(ctx, spec, cacheRoot, cachePath)
	}); err != nil {
		return "", fmt.Errorf("materialize remote theme cache: %w", err)
	}
	if resolved != "" {
		return resolved, nil
	}
	return cachePath, nil
}

func (r *remoteThemeResolver) themeCacheRoot() (string, error) {
	if r.cacheRoot != "" {
		return r.cacheRoot, nil
	}
	root, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate remote theme cache: %w", err)
	}
	return filepath.Join(root, "jigyll", "themes"), nil
}

func remoteThemeCacheKey(spec remoteThemeSpec) string {
	normalized := strings.ToLower(spec.Owner) + "/" + strings.ToLower(spec.Repo) + "@" + strings.ToLower(spec.Ref)
	digest := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(digest[:])
}

func validRemoteThemeCache(cachePath string) bool {
	info, err := os.Lstat(cachePath)
	if err != nil || !info.IsDir() {
		return false
	}
	layoutDir, err := os.Lstat(filepath.Join(cachePath, "_layouts"))
	if err != nil || !layoutDir.IsDir() {
		return false
	}
	layout, err := os.Lstat(filepath.Join(cachePath, "_layouts", "default.html"))
	return err == nil && layout.Mode().IsRegular()
}

func (r *remoteThemeResolver) materializeRemoteTheme(ctx context.Context, spec remoteThemeSpec, cacheRoot, cachePath string) error {
	archive, err := os.CreateTemp(cacheRoot, ".remote-theme-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create temporary archive: %w", err)
	}
	archivePath := archive.Name()
	defer os.Remove(archivePath)
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close temporary archive: %w", err)
	}
	if err := os.Remove(archivePath); err != nil {
		return fmt.Errorf("prepare temporary archive: %w", err)
	}
	if err := r.downloadArchive(ctx, spec, archivePath); err != nil {
		return fmt.Errorf("download remote theme: %w", err)
	}

	temporaryTheme, err := os.MkdirTemp(cacheRoot, ".remote-theme-*")
	if err != nil {
		return fmt.Errorf("create temporary theme directory: %w", err)
	}
	defer os.RemoveAll(temporaryTheme)
	if err := extractRemoteThemeArchive(archivePath, temporaryTheme, r.limits); err != nil {
		return fmt.Errorf("extract remote theme: %w", err)
	}
	if !validRemoteThemeCache(temporaryTheme) {
		return fmt.Errorf("remote theme extraction did not produce a valid theme")
	}
	if err := os.Rename(temporaryTheme, cachePath); err != nil {
		return fmt.Errorf("publish remote theme cache: %w", err)
	}
	return nil
}
