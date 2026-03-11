package source

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/sirupsen/logrus"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/clients/forgejo"
	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/provider"
)

const (
	ForgejoSource   = "forgejo"
	CodebergSource  = "codeberg"
	CodebergBaseURL = "https://codeberg.org/api/v1"
)

type Forgejo struct {
	provider.Provider

	Client     *forgejo.Client
	BaseURL    string
	SourceName string

	Owner   string
	Repo    string
	Version string

	Release *forgejo.Release
}

func (s *Forgejo) GetSource() string {
	if s.SourceName != "" {
		return s.SourceName
	}
	return ForgejoSource
}
func (s *Forgejo) GetOwner() string {
	return s.Owner
}
func (s *Forgejo) GetRepo() string {
	return s.Repo
}
func (s *Forgejo) GetApp() string {
	return fmt.Sprintf("%s/%s", s.Owner, s.Repo)
}
func (s *Forgejo) GetID() string {
	return fmt.Sprintf("%s/%s/%s", s.GetSource(), s.GetOwner(), s.GetRepo())
}

func (s *Forgejo) GetVersion() string {
	if s.Release == nil {
		return common.Unknown
	}

	return strings.TrimPrefix(s.Release.TagName, "v")
}

func (s *Forgejo) GetDownloadsDir() string {
	return filepath.Join(s.Options.Config.GetDownloadsPath(), s.GetSource(), s.GetOwner(), s.GetRepo(), s.Version)
}

func (s *Forgejo) sourceRun(ctx context.Context) error {
	cacheFile := filepath.Join(s.Options.Config.GetMetadataPath(), fmt.Sprintf("cache-%s", s.GetID()))

	s.Client = forgejo.NewClient(httpcache.NewTransport(diskcache.New(cacheFile)).Client())
	if s.BaseURL != "" {
		s.Client.SetBaseURL(s.BaseURL)
	}
	token := s.Options.Settings["forgejo-token"].(string)
	if token != "" {
		s.Client.SetToken(token)
	}

	if err := s.FindRelease(ctx); err != nil {
		return err
	}

	if len(s.Release.Assets) == 0 {
		return fmt.Errorf("release found, but no assets found for %s version %s", s.GetApp(), s.Version)
	}

	for _, a := range s.Release.Assets {
		s.Assets = append(s.Assets, &ForgejoAsset{
			Asset:        asset.New(a.Name, "", s.GetOS(), s.GetArch(), s.Version),
			Forgejo:      s,
			ReleaseAsset: a,
		})
	}

	return nil
}

func (s *Forgejo) FindRelease(ctx context.Context) error {
	includePreReleases := s.Options.Settings["include-pre-releases"].(bool)

	release, err := s.findRelease(ctx, includePreReleases)
	if err != nil {
		return err
	}
	if release == nil {
		return fmt.Errorf("release not found")
	}

	s.Release = release

	return nil
}

func (s *Forgejo) findRelease(ctx context.Context, includePreReleases bool) (*forgejo.Release, error) {
	if s.Version == provider.VersionLatest && !includePreReleases {
		release, err := s.Client.GetLatestRelease(ctx, s.Owner, s.Repo)
		if err != nil && !strings.Contains(err.Error(), "404") {
			return nil, err
		}
		if release != nil {
			s.Version = strings.TrimPrefix(release.TagName, "v")
			return release, nil
		}
	}

	return s.findReleaseInList(ctx, includePreReleases)
}

func (s *Forgejo) findReleaseInList(ctx context.Context, includePreReleases bool) (*forgejo.Release, error) {
	releases, err := s.Client.ListReleases(ctx, s.Owner, s.Repo)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			if s.Options.Settings["forgejo-token"].(string) == "" {
				log.Warn("no authentication token provided, a 404 error may be due to permissions")
			}
		}
		return nil, err
	}

	for _, r := range releases {
		logrus.
			WithField("owner", s.GetOwner()).
			WithField("repo", s.GetRepo()).
			Tracef("found release: %s", r.TagName)

		if s.Version == provider.VersionLatest && includePreReleases && r.Prerelease {
			s.Version = strings.TrimPrefix(r.TagName, "v")
			return r, nil
		}

		if strings.TrimPrefix(r.TagName, "v") == strings.TrimPrefix(s.Version, "v") {
			return r, nil
		}
	}

	return nil, nil
}

func (s *Forgejo) PreRun(ctx context.Context) error {
	if err := s.sourceRun(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Forgejo) Run(ctx context.Context) error {
	if err := s.Discover(strings.Split(s.Repo, "/"), s.Version); err != nil {
		return err
	}

	if err := s.CommonRun(ctx); err != nil {
		return err
	}

	return nil
}
