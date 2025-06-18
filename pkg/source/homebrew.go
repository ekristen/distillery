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
	"github.com/ekristen/distillery/pkg/clients/homebrew"
	"github.com/ekristen/distillery/pkg/provider"
)

const HomebrewSource = "homebrew"

type Homebrew struct {
	provider.Provider

	client *homebrew.Client

	Formula string
	Version string
}

func (s *Homebrew) GetSource() string {
	return HomebrewSource
}
func (s *Homebrew) GetOwner() string {
	return HomebrewSource
}
func (s *Homebrew) GetRepo() string {
	return s.Formula
}
func (s *Homebrew) GetApp() string {
	return s.Formula
}
func (s *Homebrew) GetID() string {
	return s.Formula
}

func (s *Homebrew) GetVersion() string {
	return strings.TrimPrefix(s.Version, "v")
}

func (s *Homebrew) GetDownloadsDir() string {
	return filepath.Join(s.Options.Config.GetDownloadsPath(), s.GetSource(), s.GetOwner(), s.GetRepo(), s.Version)
}

func (s *Homebrew) sourceRun(ctx context.Context) error {
	cacheFile := filepath.Join(s.Options.Config.GetMetadataPath(), fmt.Sprintf("cache-%s", s.GetID()))

	s.client = homebrew.NewClient(httpcache.NewTransport(diskcache.New(cacheFile)).Client())

	log.Debug().Msg("fetching formula")

	formula, err := s.client.GetFormula(ctx, s.Formula)
	if err != nil {
		return err
	}

	if s.Version == "latest" {
		s.Version = formula.Versions.Stable
	} else {
		// match major/minor
		log.Debug().Msg("selecting version")
	}

	if len(formula.Dependencies) > 0 {
		return fmt.Errorf("formula with dependencies are not currently supported")
	}

	for osSlug, variant := range formula.Bottle.Stable.Files {
		newVariant := variant
		osSlug = strings.ReplaceAll(osSlug, "_", "-")
		osSlug = strings.ReplaceAll(osSlug, "x86-64", "x86_64")

		slugParts := strings.Split(osSlug, "-")
		slugArch := "amd64"
		slugCodename := slugParts[0]
		if len(slugParts) > 1 {
			slugArch = slugParts[0]
			slugCodename = slugParts[1]
		}

		name := fmt.Sprintf("%s-%s-%s-%s", formula.Name, s.Version, slugCodename, slugArch)

		s.Assets = append(s.Assets, &HomebrewAsset{
			Asset:       asset.New(name, "", s.GetOS(), s.GetArch(), s.Version),
			FileVariant: &newVariant,
			Homebrew:    s,
		})
	}

	return nil
}

// PreRun - run the source specific logic
func (s *Homebrew) PreRun(ctx context.Context) error {
	if err := s.sourceRun(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Homebrew) Run(ctx context.Context) error {
	if err := s.Discover([]string{s.Formula}, s.Version); err != nil {
		return err
	}

	if err := s.CommonRun(ctx); err != nil {
		return err
	}

	return nil
}
