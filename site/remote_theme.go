package site

import (
	"fmt"
	"regexp"
	"strings"
)

// remoteThemeSpec identifies the immutable GitHub revision used as a theme.
type remoteThemeSpec struct {
	Owner string
	Repo  string
	Ref   string
}

var githubPathSegment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
var commitSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

// parseRemoteTheme accepts the intentionally narrow remote-theme compatibility
// subset: a GitHub repository at an immutable, full commit SHA.
func parseRemoteTheme(raw string) (remoteThemeSpec, error) {
	if strings.Count(raw, "@") != 1 {
		return remoteThemeSpec{}, invalidRemoteTheme(raw, "must contain exactly one @ separator")
	}

	repository, ref, _ := strings.Cut(raw, "@")
	if ref == "" {
		return remoteThemeSpec{}, invalidRemoteTheme(raw, "revision is required")
	}
	if strings.Count(repository, "/") != 1 {
		return remoteThemeSpec{}, invalidRemoteTheme(raw, "repository must be owner/repository")
	}

	owner, repo, _ := strings.Cut(repository, "/")
	if !validGitHubPathSegment(owner) {
		return remoteThemeSpec{}, invalidRemoteTheme(raw, "owner must be a GitHub path segment")
	}
	if !validGitHubPathSegment(repo) {
		return remoteThemeSpec{}, invalidRemoteTheme(raw, "repository must be a GitHub path segment")
	}
	if !commitSHA.MatchString(ref) {
		return remoteThemeSpec{}, invalidRemoteTheme(raw, "revision must be a 40-character hexadecimal commit SHA")
	}

	return remoteThemeSpec{
		Owner: strings.ToLower(owner),
		Repo:  strings.ToLower(repo),
		Ref:   strings.ToLower(ref),
	}, nil
}

func validGitHubPathSegment(segment string) bool {
	return segment != "." && segment != ".." && githubPathSegment.MatchString(segment)
}

func invalidRemoteTheme(raw, reason string) error {
	return fmt.Errorf("invalid remote theme %q: %s", raw, reason)
}
