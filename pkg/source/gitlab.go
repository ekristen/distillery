package source

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/rs/zerolog/log"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/clients/gitlab"
	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/provider"
)

const GitLabSource = "gitlab"

type GitLab struct {
	provider.Provider

	Client  *gitlab.Client
	BaseURL string

	Owner   string
	Repo    string
	Version string

	Release *gitlab.Release
}

func (s *GitLab) GetSource() string {
	return GitLabSource
}
func (s *GitLab) GetOwner() string {
	return s.Owner
}
func (s *GitLab) GetRepo() string {
	return s.Repo
}
func (s *GitLab) GetApp() string {
	return fmt.Sprintf("%s/%s", s.Owner, s.Repo)
}
func (s *GitLab) GetID() string {
	return fmt.Sprintf("%s/%s/%s", s.GetSource(), s.GetOwner(), s.GetRepo())
}

func (s *GitLab) GetVersion() string {
	if s.Release == nil {
		return common.Unknown
	}

	return strings.TrimPrefix(s.Release.TagName, "v")
}

func (s *GitLab) GetDownloadsDir() string {
	return filepath.Join(s.Options.Config.GetDownloadsPath(), s.GetSource(), s.GetOwner(), s.GetRepo(), s.Version)
}

func (s *GitLab) sourceRun(ctx context.Context) error {
	cacheFile := filepath.Join(s.Options.Config.GetMetadataPath(), fmt.Sprintf("cache-%s", s.GetID()))

	s.Client = gitlab.NewClient(httpcache.NewTransport(diskcache.New(cacheFile)).Client())
	if s.BaseURL != "" {
		s.Client.SetBaseURL(s.BaseURL)
	}
	token := s.Options.Settings["gitlab-token"].(string)
	if token != "" {
		s.Client.SetToken(token)
	}

	if err := s.FindRelease(ctx); err != nil {
		return err
	}

	if s.Release.Assets == nil {
		return fmt.Errorf("release found, but no assets found for %s version %s", s.GetApp(), s.Version)
	}

	for _, a := range s.Release.Assets.Links {
		s.Assets = append(s.Assets, &GitLabAsset{
			Asset:  asset.New(filepath.Base(a.DirectAssetURL), "", s.GetOS(), s.GetArch(), s.Version),
			GitLab: s,
			Link:   a,
		})
	}

	return nil
}

func (s *GitLab) FindRelease(ctx context.Context) error {
	var err error
	var release *gitlab.Release

	includePreReleases := s.Options.Settings["include-pre-releases"].(bool)

	if s.Version == provider.VersionLatest && !includePreReleases {
		release, err = s.Client.GetLatestRelease(ctx, fmt.Sprintf("%s/%s", s.Owner, s.Repo))
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return err
		}

		if release != nil {
			s.Version = strings.TrimPrefix(release.TagName, "v")
		}
	}

	if release == nil {
		releases, err := s.Client.ListReleases(ctx, fmt.Sprintf("%s/%s", s.Owner, s.Repo))
		if err != nil {
			if strings.Contains(err.Error(), "404 Not Found") {
				gitlabToken := s.Options.Settings["gitlab-token"].(string)
				if gitlabToken == "" {
					log.Warn().Msg("no authentication token provided, a 404 error may be due to permissions")
				}
			}

			return err
		}

		for _, r := range releases {
			s.Logger.Trace().
				Str("owner", s.GetOwner()).
				Str("repo", s.GetRepo()).
				Msgf("found release: %s", r.TagName)

			if includePreReleases && r.UpcomingRelease {
				s.Version = strings.TrimPrefix(r.TagName, "v")
				release = r
				break
			}

			tagName := strings.TrimPrefix(r.TagName, "v")

			if tagName == s.Version {
				release = r
				break
			}
		}
	}

	if release == nil {
		return fmt.Errorf("release not found")
	}

	s.Release = release

	return nil
}

func (s *GitLab) PreRun(ctx context.Context) error {
	if err := s.sourceRun(ctx); err != nil {
		return err
	}

	return nil
}

func (s *GitLab) Run(ctx context.Context) error {
	if err := s.Discover(strings.Split(s.Repo, "/"), s.Version); err != nil {
		return err
	}

	if err := s.CommonRun(ctx); err != nil {
		return err
	}

	return nil
}
