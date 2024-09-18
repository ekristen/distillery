package source

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/google/go-github/v65/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"

	"github.com/ekristen/distillery/pkg/asset"
)

const GitHubSource = "github"

type GitHub struct {
	Source

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
	return filepath.Join(s.Options.DownloadsDir, s.GetSource(), s.GetOwner(), s.GetRepo(), s.Version)
}

func (s *GitHub) GetID() string {
	return strings.Join([]string{s.GetSource(), s.GetOwner(), s.GetRepo(), s.GetOS(), s.GetArch()}, "-")
}

// sourceRun - run the source specific logic
func (s *GitHub) sourceRun(ctx context.Context) error {
	cacheFile := filepath.Join(s.Options.MetadataDir, fmt.Sprintf("cache-%s", s.GetID()))

	s.client = github.NewClient(httpcache.NewTransport(diskcache.New(cacheFile)).Client())
	githubToken := s.Options.Settings["github-token"].(string)
	if githubToken != "" {
		log.Debug("auth token provided")
		s.client = s.client.WithAuthToken(githubToken)
	}

	if err := s.FindRelease(ctx); err != nil {
		return err
	}

	if err := s.GetReleaseAssets(ctx); err != nil {
		return err
	}

	return nil
}

// Run - run the source
func (s *GitHub) Run(ctx context.Context) error {
	if err := s.sourceRun(ctx); err != nil {
		return err
	}

	if err := s.Discover(s.Assets, []string{s.Repo}); err != nil {
		return err
	}

	if err := s.commonRun(ctx); err != nil {
		return err
	}

	return nil
}

// FindRelease - query API to find the version being sought or return an error
func (s *GitHub) FindRelease(ctx context.Context) error {
	var err error
	var release *github.RepositoryRelease

	if s.Version == VersionLatest {
		release, _, err = s.client.Repositories.GetLatestRelease(ctx, s.GetOwner(), s.GetRepo())
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return err
		}

		if release != nil {
			s.Version = strings.TrimPrefix(release.GetTagName(), "v")
		}
	}

	if release == nil {
		releases, _, err := s.client.Repositories.ListReleases(ctx, s.GetOwner(), s.GetRepo(), nil)
		if err != nil {
			if strings.Contains(err.Error(), "404 Not Found") {
				githubToken := s.Options.Settings["github-token"].(string)
				if githubToken == "" {
					log.Warn("no authentication token provided, a 404 error may be due to permissions")
				}
			}

			return err
		}
		for _, r := range releases {
			includePreReleases := s.Options.Settings["include-pre-releases"].(bool)
			if includePreReleases && r.GetPrerelease() {
				s.Version = strings.TrimPrefix(r.GetTagName(), "v")
				release = r
				break
			}

			if r.GetTagName() == s.Version || r.GetName() == fmt.Sprintf("v%s", s.Version) {
				release = r
				break
			}
		}
	}

	if release == nil {
		return fmt.Errorf("release not found")
	}

	log.Infof("installing version: %s", strings.TrimPrefix(release.GetTagName(), "v"))

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

	logrus.Tracef("found %d assets", len(s.Assets))

	if len(s.Assets) == 0 {
		return fmt.Errorf("no assets found")
	}

	return nil
}
