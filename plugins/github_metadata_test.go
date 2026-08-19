package plugins

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/google/go-github/github"
	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/liquid"
	"github.com/stretchr/testify/require"
)

func TestGitHubMetadataSiteDrop(t *testing.T) {
	t.Run("populates repository and contributors", func(t *testing.T) {
		var paths []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			switch r.URL.Path {
			case "/repos/owner/repository":
				writeGitHubRepository(w)
			case "/repos/owner/repository/contributors":
				require.Equal(t, "100", r.URL.Query().Get("per_page"))
				fmt.Fprint(w, `[{"login":"octocat","id":1,"avatar_url":"https://avatars.example/octocat","html_url":"https://github.com/octocat","site_admin":false,"contributions":7}]`)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		data := map[string]interface{}{}
		plugin := jekyllGithubMetadataPlugin{client: githubTestClient(t, server)}
		require.NoError(t, plugin.ModifySiteDrop(githubMetadataSite(t), data))
		require.Equal(t, []string{"/repos/owner/repository", "/repos/owner/repository/contributors"}, paths)
		require.Equal(t, "owner/repository", githubDropValue(t, data, "repository_nwo"))
		require.Equal(t, "repository", *githubDropValue(t, data, "repository_name").(*string))
		require.Equal(t, "repository", *githubDropValue(t, data, "project_title").(*string))

		contributors := githubDropValue(t, data, "contributors").([]map[string]interface{})
		require.Equal(t, []map[string]interface{}{{
			"login":         "octocat",
			"id":            int64(1),
			"avatar_url":    "https://avatars.example/octocat",
			"html_url":      "https://github.com/octocat",
			"site_admin":    false,
			"contributions": 7,
		}}, contributors)

		rendered, err := liquid.NewEngine().ParseAndRenderString(`{% for contributor in site.github.contributors %}{{ contributor.login }}|{{ contributor.html_url }}|{{ contributor.avatar_url }}{% endfor %}`, liquid.Bindings{"site": data})
		require.NoError(t, err)
		require.Equal(t, "octocat|https://github.com/octocat|https://avatars.example/octocat", rendered)
	})

	t.Run("installs an empty contributor collection", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/repos/owner/repository":
				writeGitHubRepository(w)
			case "/repos/owner/repository/contributors":
				fmt.Fprint(w, `[]`)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		data := map[string]interface{}{}
		plugin := jekyllGithubMetadataPlugin{client: githubTestClient(t, server)}
		require.NoError(t, plugin.ModifySiteDrop(githubMetadataSite(t), data))
		require.Empty(t, githubDropValue(t, data, "contributors"))
	})
}

func TestGitHubMetadataContributorsPagination(t *testing.T) {
	requests := map[string]int{}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/owner/repository/contributors", r.URL.Path)
		require.Equal(t, "100", r.URL.Query().Get("per_page"))
		page := r.URL.Query().Get("page")
		requests[page]++
		switch page {
		case "1":
			w.Header().Set("Link", fmt.Sprintf("<%s/repos/owner/repository/contributors?page=2&per_page=100>; rel=\"next\"", server.URL))
			fmt.Fprint(w, `[{"login":"first"},{"login":"second"}]`)
		case "2":
			fmt.Fprint(w, `[{"login":"third"}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	contributors, err := getGitHubContributors(context.Background(), githubTestClient(t, server), "owner/repository")
	require.NoError(t, err)
	require.Equal(t, map[string]int{"1": 1, "2": 1}, requests)
	require.Equal(t, []map[string]interface{}{
		{"login": "first"},
		{"login": "second"},
		{"login": "third"},
	}, contributors)
}

func TestGitHubMetadataContributorConversion(t *testing.T) {
	contributor := &github.Contributor{
		Login:             github.String("octocat"),
		ID:                github.Int64(1),
		AvatarURL:         github.String("avatar"),
		GravatarID:        github.String("gravatar"),
		URL:               github.String("url"),
		HTMLURL:           github.String("html"),
		FollowersURL:      github.String("followers"),
		FollowingURL:      github.String("following"),
		GistsURL:          github.String("gists"),
		StarredURL:        github.String("starred"),
		SubscriptionsURL:  github.String("subscriptions"),
		OrganizationsURL:  github.String("organizations"),
		ReposURL:          github.String("repos"),
		EventsURL:         github.String("events"),
		ReceivedEventsURL: github.String("received-events"),
		Type:              github.String("User"),
		SiteAdmin:         github.Bool(false),
		Contributions:     github.Int(0),
	}

	require.Equal(t, map[string]interface{}{
		"login":               "octocat",
		"id":                  int64(1),
		"avatar_url":          "avatar",
		"gravatar_id":         "gravatar",
		"url":                 "url",
		"html_url":            "html",
		"followers_url":       "followers",
		"following_url":       "following",
		"gists_url":           "gists",
		"starred_url":         "starred",
		"subscriptions_url":   "subscriptions",
		"organizations_url":   "organizations",
		"repos_url":           "repos",
		"events_url":          "events",
		"received_events_url": "received-events",
		"type":                "User",
		"site_admin":          false,
		"contributions":       0,
	}, githubContributorToLiquid(contributor))
	require.Empty(t, githubContributorToLiquid(&github.Contributor{}))
}

func TestGitHubMetadataContributorFailure(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repository":
			writeGitHubRepository(w)
		case "/repos/owner/repository/contributors":
			if r.URL.Query().Get("page") == "1" {
				w.Header().Set("Link", fmt.Sprintf("<%s/repos/owner/repository/contributors?page=2&per_page=100>; rel=\"next\"", server.URL))
				fmt.Fprint(w, `[{"login":"partial"}]`)
				return
			}
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("JEKYLL_GITHUB_TOKEN", "secret-token")
	data := map[string]interface{}{}
	plugin := jekyllGithubMetadataPlugin{client: githubTestClient(t, server)}
	err := plugin.ModifySiteDrop(githubMetadataSite(t), data)
	require.Error(t, err)
	require.ErrorContains(t, err, "listing GitHub contributors for owner/repository page 2")
	require.NotContains(t, err.Error(), "secret-token")
	require.NotContains(t, data, "github")
}

func TestGitHubMetadataClientAuthentication(t *testing.T) {
	for _, test := range []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{name: "jekyll token wins", env: map[string]string{"JEKYLL_GITHUB_TOKEN": "jekyll", "GITHUB_TOKEN": "github", "OCTOKIT_ACCESS_TOKEN": "octokit"}, expected: "Bearer jekyll"},
		{name: "GitHub token fallback", env: map[string]string{"GITHUB_TOKEN": "github", "OCTOKIT_ACCESS_TOKEN": "octokit"}, expected: "Bearer github"},
		{name: "Octokit token fallback", env: map[string]string{"OCTOKIT_ACCESS_TOKEN": "octokit"}, expected: "Bearer octokit"},
		{name: "no token", env: map[string]string{}, expected: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, name := range []string{"JEKYLL_GITHUB_TOKEN", "GITHUB_TOKEN", "OCTOKIT_ACCESS_TOKEN"} {
				t.Setenv(name, test.env[name])
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, test.expected, r.Header.Get("Authorization"))
				writeGitHubRepository(w)
			}))
			defer server.Close()

			client := jekyllGithubMetadataPlugin{}.githubClient(context.Background())
			client.BaseURL = githubTestURL(t, server)
			_, _, err := client.Repositories.Get(context.Background(), "owner", "repository")
			require.NoError(t, err)
		})
	}
}

func TestGitHubMetadataPageOverrides(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repository":
			writeGitHubRepository(w)
		case "/repos/owner/repository/contributors":
			fmt.Fprint(w, `[]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	for key, value := range map[string]string{
		"PAGES_API_URL":         "https://api.example",
		"JEKYLL_BUILD_REVISION": "revision",
		"PAGES_ENV":             "production",
		"PAGES_HELP_URL":        "https://help.example",
		"PAGES_GITHUB_HOSTNAME": "https://github.example",
		"PAGES_PAGES_HOSTNAME":  "pages.example",
	} {
		t.Setenv(key, value)
	}

	data := map[string]interface{}{}
	plugin := jekyllGithubMetadataPlugin{client: githubTestClient(t, server)}
	require.NoError(t, plugin.ModifySiteDrop(githubMetadataSite(t), data))
	require.Equal(t, "https://api.example", githubDropValue(t, data, "api_url"))
	require.Equal(t, "revision", githubDropValue(t, data, "build_revision"))
	require.Equal(t, "production", githubDropValue(t, data, "environment"))
	require.Equal(t, "https://help.example", githubDropValue(t, data, "help_url"))
	require.Equal(t, "https://github.example", githubDropValue(t, data, "hostname"))
	require.Equal(t, "pages.example", githubDropValue(t, data, "pages_hostname"))
}

func githubMetadataSite(t *testing.T) siteFake {
	t.Helper()
	cfg := config.Default()
	cfg.Source = t.TempDir()
	cfg.Set("repository", "owner/repository")
	return siteFake{c: cfg, e: liquid.NewEngine()}
}

func githubTestClient(t *testing.T, server *httptest.Server) *github.Client {
	t.Helper()
	client := github.NewClient(server.Client())
	client.BaseURL = githubTestURL(t, server)
	return client
}

func githubTestURL(t *testing.T, server *httptest.Server) *url.URL {
	t.Helper()
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	return baseURL
}

func githubDropValue(t *testing.T, data map[string]interface{}, key string) interface{} {
	t.Helper()
	drop := reflect.ValueOf(data["github"])
	require.True(t, drop.IsValid())
	value := drop.MapIndex(reflect.ValueOf(key))
	require.True(t, value.IsValid(), "site.github.%s is missing", key)
	return value.Interface()
}

func writeGitHubRepository(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"name":"repository","owner":{"login":"owner","url":"https://api.github.com/users/owner","gravatar_id":""},"archive_url":"https://api.github.com/repos/owner/repository/{archive_format}{/ref}","clone_url":"https://github.com/owner/repository.git","git_url":"git://github.com/owner/repository.git","issues_url":"https://api.github.com/repos/owner/repository/issues{/number}","language":"Go","url":"https://api.github.com/repos/owner/repository","releases_url":"https://api.github.com/repos/owner/repository/releases{/id}","has_downloads":true}`)
}
