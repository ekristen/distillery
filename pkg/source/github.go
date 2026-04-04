package source

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/gregjones/httpcache"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/httpclient"
	"github.com/ekristen/distillery/pkg/provider"
)

const GitHubSource = "github"

type GitHub struct {
	provider.Provider

	client *github.Client

	Version string // Version to find for installation
	Owner   string // Owner of the repository
	Repo    string // Repository name

	Release *github.RepositoryRelease
}

func (s *GitHub) GetSource() string {
	return GitHubSource
}
func (s *GitHub) GetOwner() string {
	return s.Owner
}
func (s *GitHub) GetRepo() string {
	return s.Repo
}
func (s *GitHub) GetApp() string {
	return fmt.Sprintf("%s/%s", s.Owner, s.Repo)
}

func (s *GitHub) GetDownloadsDir() string {
	return filepath.Join(s.Options.Config.GetDownloadsPath(), s.GetSource(), s.GetOwner(), s.GetRepo(), s.Version)
}

func (s *GitHub) GetID() string {
	return strings.Join([]string{s.GetSource(), s.GetOwner(), s.GetRepo(), s.GetOS(), s.GetArch()}, "-")
}

func (s *GitHub) GetVersion() string {
	if s.Release == nil {
		return common.Unknown
	}

	return strings.TrimPrefix(s.Release.GetTagName(), "v")
}

func (s *GitHub) PreRun(ctx context.Context) error {
	if err := s.sourceRun(ctx); err != nil {
		return err
	}

	return nil
}

// Run - run the source
func (s *GitHub) Run(ctx context.Context) error {
	// this is from the Provider struct
	if err := s.Discover(strings.Split(s.Repo, "/"), s.Version); err != nil {
		return err
	}

	if err := s.CommonRun(ctx); err != nil {
		return err
	}

	return nil
}

// sourceRun - run the source specific logic
func (s *GitHub) sourceRun(ctx context.Context) error {
	cacheFile := filepath.Join(s.Options.Config.GetMetadataPath(), fmt.Sprintf("cache-%s", s.GetID()))

	s.client = github.NewClient(httpcache.NewTransport(httpclient.NewDiskCache(cacheFile)).Client())
	useDistCache := s.Options.Settings["use-dist-cache"].(bool)
	if useDistCache {
		s.Logger.Debug().Msg("using dist cache")
		baseURL := s.Options.Settings["dist-cache-url"].(string)
		if baseURL != "" {
			s.Logger.Debug().Msgf("using dist cache with base url: %s", baseURL)
			parsedURL, err := url.Parse(baseURL)
			if err != nil {
				return fmt.Errorf("invalid dist-cache-url %q: %w", baseURL, err)
			}
			if parsedURL.Path == "" {
				parsedURL.Path = "/"
			}
			s.client.BaseURL = parsedURL
		}
	} else {
		githubToken := s.Options.Settings["github-token"].(string)
		if githubToken != "" {
			s.Logger.Debug().Msg("auth token provided")
			s.client = s.client.WithAuthToken(githubToken)
		}
	}

	if err := s.FindRelease(ctx); err != nil {
		return err
	}

	if err := s.GetReleaseAssets(ctx); err != nil {
		return err
	}

	return nil
}

// FindRelease - query API to find the version being sought or return an error
func (s *GitHub) FindRelease(ctx context.Context) error { //nolint:gocyclo
	var err error
	var release *github.RepositoryRelease

	s.Logger.Trace().
		Str("owner", s.GetOwner()).
		Str("repo", s.GetRepo()).
		Msgf("finding release for %s", s.Version)

	includePreReleases := s.Options.Settings["include-pre-releases"].(bool)

	if s.Version == provider.VersionLatest && !includePreReleases {
		release, _, err = s.client.Repositories.GetLatestRelease(ctx, s.GetOwner(), s.GetRepo())
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return err
		}

		if release != nil {
			s.Version = strings.TrimPrefix(release.GetTagName(), "v")
		}
	}

	if release == nil {
		params := &github.ListOptions{
			PerPage: 100,
		}

		for {
			releases, res, err := s.client.Repositories.ListReleases(ctx, s.GetOwner(), s.GetRepo(), params)
			if err != nil {
				if strings.Contains(err.Error(), "404 Not Found") {
					githubToken := s.Options.Settings["github-token"].(string)
					if githubToken == "" {
						return fmt.Errorf("repository %s/%s not found (provide --github-token for private repos)",
							s.GetOwner(), s.GetRepo())
					}
					return fmt.Errorf("repository %s/%s not found", s.GetOwner(), s.GetRepo())
				}

				return err
			}

			for _, r := range releases {
				tagName := strings.TrimPrefix(r.GetTagName(), "v")

				s.Logger.Trace().
					Str("owner", s.GetOwner()).
					Str("repo", s.GetRepo()).
					Str("want", s.Version).
					Str("found", tagName).
					Msgf("found release: %s", tagName)

				if tagName == strings.TrimPrefix(s.Version, "v") {
					release = r
					break
				}
			}

			// If we found the release or there are no more pages, break the loop
			if release != nil || res.NextPage == 0 {
				break
			}

			params.Page = res.NextPage
		}
	}

	if release == nil {
		if s.Version == provider.VersionLatest {
			return fmt.Errorf("no releases found for %s/%s", s.GetOwner(), s.GetRepo())
		}
		return fmt.Errorf("version %s not found for %s/%s", s.Version, s.GetOwner(), s.GetRepo())
	}

	s.Release = release

	return nil
}

func (s *GitHub) GetReleaseAssets(ctx context.Context) error {
	params := &github.ListOptions{
		PerPage: 100,
	}

	for {
		assets, res, err := s.client.Repositories.ListReleaseAssets(
			ctx, s.GetOwner(), s.GetRepo(), s.Release.GetID(), params)
		if err != nil {
			return err
		}

		for _, a := range assets {
			s.Assets = append(s.Assets, &GitHubAsset{
				Asset:        asset.New(a.GetName(), "", s.GetOS(), s.GetArch(), s.Version),
				GitHub:       s,
				ReleaseAsset: a,
			})
		}

		if res.NextPage == 0 {
			break
		}

		params.Page = res.NextPage
	}

	s.Logger.Trace().Msgf("found %d assets", len(s.Assets))

	if len(s.Assets) == 0 {
		return fmt.Errorf("no assets found")
	}

	return nil
}
