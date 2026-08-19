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

func getGitHubContributors(ctx context.Context, client *github.Client, nwo string) ([]map[string]interface{}, error) {
	owner, name, err := githubRepositoryName(nwo)
	if err != nil {
		return nil, err
	}

	contributors := make([]map[string]interface{}, 0)
	for page := 1; ; {
		items, response, err := client.Repositories.ListContributors(ctx, owner, name, &github.ListContributorsOptions{
			ListOptions: github.ListOptions{Page: page, PerPage: 100},
		})
		if err != nil {
			return nil, fmt.Errorf("listing GitHub contributors for %s page %d: %w", nwo, page, err)
		}
		if response == nil {
			return nil, fmt.Errorf("listing GitHub contributors for %s page %d: missing response", nwo, page)
		}

		for _, contributor := range items {
			contributors = append(contributors, githubContributorToLiquid(contributor))
		}
		if response.NextPage == 0 {
			return contributors, nil
		}
		page = response.NextPage
	}
}

func githubContributorToLiquid(contributor *github.Contributor) map[string]interface{} {
	result := make(map[string]interface{})
	if contributor == nil {
		return result
	}

	setGitHubContributorProfile(result, contributor)
	setGitHubContributorURLs(result, contributor)
	setGitHubContributorProperties(result, contributor)
	return result
}

func setGitHubContributorProfile(result map[string]interface{}, contributor *github.Contributor) {
	if contributor.Login != nil {
		result["login"] = *contributor.Login
	}
	if contributor.AvatarURL != nil {
		result["avatar_url"] = *contributor.AvatarURL
	}
	if contributor.GravatarID != nil {
		result["gravatar_id"] = *contributor.GravatarID
	}
	if contributor.URL != nil {
		result["url"] = *contributor.URL
	}
	if contributor.HTMLURL != nil {
		result["html_url"] = *contributor.HTMLURL
	}
}

func setGitHubContributorURLs(result map[string]interface{}, contributor *github.Contributor) {
	if contributor.FollowersURL != nil {
		result["followers_url"] = *contributor.FollowersURL
	}
	if contributor.FollowingURL != nil {
		result["following_url"] = *contributor.FollowingURL
	}
	if contributor.GistsURL != nil {
		result["gists_url"] = *contributor.GistsURL
	}
	if contributor.StarredURL != nil {
		result["starred_url"] = *contributor.StarredURL
	}
	if contributor.SubscriptionsURL != nil {
		result["subscriptions_url"] = *contributor.SubscriptionsURL
	}
	if contributor.OrganizationsURL != nil {
		result["organizations_url"] = *contributor.OrganizationsURL
	}
	if contributor.ReposURL != nil {
		result["repos_url"] = *contributor.ReposURL
	}
	if contributor.EventsURL != nil {
		result["events_url"] = *contributor.EventsURL
	}
	if contributor.ReceivedEventsURL != nil {
		result["received_events_url"] = *contributor.ReceivedEventsURL
	}
}

func setGitHubContributorProperties(result map[string]interface{}, contributor *github.Contributor) {
	if contributor.ID != nil {
		result["id"] = *contributor.ID
	}
	if contributor.Type != nil {
		result["type"] = *contributor.Type
	}
	if contributor.SiteAdmin != nil {
		result["site_admin"] = *contributor.SiteAdmin
	}
	if contributor.Contributions != nil {
		result["contributions"] = *contributor.Contributions
	}
}
