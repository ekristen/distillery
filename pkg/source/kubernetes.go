package source

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/google/go-github/v66/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/common"
)

const KubernetesSource = "kubernetes"

type Kubernetes struct {
	GitHub

	AppName string
}

func (s *Kubernetes) GetSource() string {
	return "github"
}
func (s *Kubernetes) GetOwner() string {
	return KubernetesSource
}
func (s *Kubernetes) GetRepo() string {
	return s.Repo
}
func (s *Kubernetes) GetApp() string {
	return fmt.Sprintf("%s/%s", s.Owner, s.AppName)
}
func (s *Kubernetes) GetID() string {
	return fmt.Sprintf("%s-%s", s.GetSource(), s.GetRepo())
}

func (s *Kubernetes) GetVersion() string {
	if s.Release == nil {
		return common.Unknown
	}

	return strings.TrimPrefix(s.Release.GetTagName(), "v")
}

func (s *Kubernetes) GetDownloadsDir() string {
	return filepath.Join(s.Options.Config.GetDownloadsPath(), s.GetSource(), s.GetOwner(), s.GetRepo(), s.Version)
}

// sourceRun - run the source specific logic (note this is duplicate because of the GetReleaseAssets override)
func (s *Kubernetes) sourceRun(ctx context.Context) error { //nolint:dupl
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

func (s *Kubernetes) GetReleaseAssets(_ context.Context) error {
	binName := fmt.Sprintf("%s-%s-%s-%s", s.AppName, s.Version, s.GetOS(), s.GetArch())
	s.Assets = append(s.Assets, &HTTPAsset{
		Asset:  asset.New(binName, s.AppName, s.GetOS(), s.GetArch(), s.Version),
		Source: s,
		URL: fmt.Sprintf("https://dl.k8s.io/release/v%s/bin/%s/%s/%s",
			s.Version, s.GetOS(), s.GetArch(), s.AppName),
	}, &HTTPAsset{
		Asset:  asset.New(binName+".sha256", "", s.GetOS(), s.GetArch(), s.Version),
		Source: s,
		URL: fmt.Sprintf("https://dl.k8s.io/release/v%s/bin/%s/%s/%s.sha256",
			s.Version, s.GetOS(), s.GetArch(), s.AppName),
	}, &HTTPAsset{
		Asset:  asset.New(binName+".sig", "", s.GetOS(), s.GetArch(), s.Version),
		Source: s,
		URL: fmt.Sprintf("https://dl.k8s.io/release/v%s/bin/%s/%s/%s.sig",
			s.Version, s.GetOS(), s.GetArch(), s.AppName),
	}, &HTTPAsset{
		Asset:  asset.New(binName+".cert", "", s.GetOS(), s.GetArch(), s.Version),
		Source: s,
		URL: fmt.Sprintf("https://dl.k8s.io/release/v%s/bin/%s/%s/%s.cert",
			s.Version, s.GetOS(), s.GetArch(), s.AppName),
	})

	return nil
}

func (s *Kubernetes) PreRun(ctx context.Context) error {
	if err := s.sourceRun(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Kubernetes) Run(ctx context.Context) error {
	// this is from the Provider struct
	if err := s.Discover(strings.Split(s.Repo, "/"), s.Version); err != nil {
		return err
	}

	if err := s.CommonRun(ctx); err != nil {
		return err
	}

	return nil
}
