package plugins

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func (p jekyllGithubMetadataPlugin) githubClient(ctx context.Context) *github.Client {
	if p.client != nil {
		return p.client
	}

	var token string
	for _, name := range []string{"JEKYLL_GITHUB_TOKEN", "GITHUB_TOKEN", "OCTOKIT_ACCESS_TOKEN"} {
		if value := os.Getenv(name); value != "" {
			token = value
			break
		}
	}

	var source oauth2.TokenSource
	if token != "" {
		source = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	}
	return github.NewClient(oauth2.NewClient(ctx, source))
}

func getGitHubRepo(ctx context.Context, client *github.Client, nwo string) (*github.Repository, error) {
	owner, name, err := githubRepositoryName(nwo)
	if err != nil {
		return nil, err
	}

	repo, _, err := client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("getting GitHub repository %s: %w", nwo, err)
	}
	return repo, nil
}

func githubRepositoryName(nwo string) (string, string, error) {
	parts := strings.SplitN(nwo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid GitHub repository %q", nwo)
	}
	return parts[0], parts[1], nil
}
