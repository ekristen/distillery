package source

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/google/go-github/v78/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/common"
)

const HelmSource = "helm"

type Helm struct {
	GitHub

	AppName string
}

func (s *Helm) GetSource() string {
	return "github"
}
func (s *Helm) GetOwner() string {
	return HelmSource
}
func (s *Helm) GetRepo() string {
	return s.Repo
}
func (s *Helm) GetApp() string {
	return fmt.Sprintf("%s/%s", s.Owner, s.AppName)
}
func (s *Helm) GetID() string {
	return fmt.Sprintf("%s-%s", s.GetSource(), s.GetRepo())
}

func (s *Helm) GetVersion() string {
	if s.Release == nil {
		return common.Unknown
	}

	return strings.TrimPrefix(s.Release.GetTagName(), "v")
}

func (s *Helm) GetDownloadsDir() string {
	return filepath.Join(s.Options.Config.GetDownloadsPath(), s.GetSource(), s.GetOwner(), s.GetRepo(), s.Version)
}

// sourceRun - run the source specific logic (note this is duplicate because of the GetReleaseAssets override)
func (s *Helm) sourceRun(ctx context.Context) error { //nolint:dupl
	cacheFile := filepath.Join(s.Options.Config.GetMetadataPath(), fmt.Sprintf("cache-%s", s.GetID()))

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

func (s *Helm) GetReleaseAssets(_ context.Context) error {
	binName := fmt.Sprintf("%s-v%s-%s-%s.tar.gz", s.AppName, s.Version, s.GetOS(), s.GetArch())
	s.Assets = append(s.Assets, &HTTPAsset{
		Asset:  asset.New(binName, s.AppName, s.GetOS(), s.GetArch(), s.Version),
		Source: s,
		URL: fmt.Sprintf("https://get.helm.sh/helm-v%s-%s-%s.tar.gz",
			s.Version, s.GetOS(), s.GetArch()),
	}, &HTTPAsset{
		Asset:  asset.New(binName+".sha256sum", "", s.GetOS(), s.GetArch(), s.Version),
		Source: s,
		URL: fmt.Sprintf("https://get.helm.sh/helm-v%s-%s-%s.tar.gz.sha256sum",
			s.Version, s.GetOS(), s.GetArch()),
	})

	return nil
}

func (s *Helm) PreRun(ctx context.Context) error {
	if err := s.sourceRun(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Helm) Run(ctx context.Context) error {
	// this is from the Provider struct
	if err := s.Discover(strings.Split(s.Repo, "/"), s.Version); err != nil {
		return err
	}

	if err := s.CommonRun(ctx); err != nil {
		return err
	}

	return nil
}
