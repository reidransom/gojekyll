package site

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const codeloadGitHubHost = "codeload.github.com"

type remoteThemeLimits struct {
	CompressedBytes int64
	ExtractedBytes  int64
	FileBytes       int64
	Entries         int64
}

type remoteThemeResolver struct {
	client     *http.Client
	cacheRoot  string
	archiveURL func(remoteThemeSpec) string
	limits     remoteThemeLimits
}

func newRemoteThemeResolver() *remoteThemeResolver {
	return &remoteThemeResolver{
		client: &http.Client{
			Timeout:       30 * time.Second,
			CheckRedirect: checkCodeloadRedirect,
		},
		archiveURL: remoteThemeArchiveURL,
		limits: remoteThemeLimits{
			CompressedBytes: 128 << 20,
			ExtractedBytes:  512 << 20,
			FileBytes:       64 << 20,
			Entries:         20_000,
		},
	}
}

func remoteThemeArchiveURL(spec remoteThemeSpec) string {
	return "https://" + codeloadGitHubHost + "/" + spec.Owner + "/" + spec.Repo + "/tar.gz/" + spec.Ref
}

func checkCodeloadRedirect(req *http.Request, _ []*http.Request) error {
	if req.URL.Scheme != "https" || req.URL.Host != codeloadGitHubHost {
		return fmt.Errorf("remote theme redirect must remain on https://%s", codeloadGitHubHost)
	}
	return nil
}

func (r *remoteThemeResolver) downloadArchive(ctx context.Context, spec remoteThemeSpec, destination string) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.archiveURL(spec), nil)
	if err != nil {
		return fmt.Errorf("create archive request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("download archive: %w", err)
	}
	defer func() {
		closeErr := resp.Body.Close()
		if err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download archive: expected HTTP 200, got %s", resp.Status)
	}

	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(destination)
		}
	}()
	defer func() {
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	written, err := io.Copy(file, io.LimitReader(resp.Body, r.limits.CompressedBytes+1))
	if err != nil {
		return fmt.Errorf("save archive: %w", err)
	}
	if written > r.limits.CompressedBytes {
		return fmt.Errorf("download archive: compressed archive exceeds %d bytes", r.limits.CompressedBytes)
	}
	return nil
}
