package install

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/inventory"
	"github.com/ekristen/distillery/pkg/provider"
)

// Options holds the parameters for a single install operation.
type Options struct {
	App                string
	Version            string
	OS                 string
	Arch               string
	Force              bool
	Config             *config.Config
	GitHubToken        string
	GitLabToken        string
	NoSignatureVerify  bool
	NoChecksumVerify   bool
	NoScoreCheck       bool
	IncludePreReleases bool
	UseDistCache       bool
	DistCacheURL       string
	Logger             zerolog.Logger
}

// DoInstall performs an install with explicit options, safe for concurrent use.
func DoInstall(ctx context.Context, opts *Options) error {
	startTime := time.Now().UTC()

	appName := opts.App
	logger := opts.Logger.With().Str("app", appName).Logger()

	logger.Info().Msg("starting installation")

	cfg := opts.Config
	inv := inventory.New(os.DirFS(cfg.BinPath), cfg.BinPath, cfg.GetOptPath(), cfg)

	name := appName
	nameParts := strings.Split(name, "@")
	version := opts.Version

	alias := cfg.GetAlias(nameParts[0])
	if alias != nil {
		name = alias.Name
		aliasVersion := alias.Version
		if len(nameParts) > 1 {
			if aliasVersion != "latest" {
				logger.Warn().Msg("version specified via cli and alias, ignoring alias version")
			}
			aliasVersion = nameParts[1]
		}
		version = aliasVersion
		name = fmt.Sprintf("%s@%s", name, version)
	} else if len(nameParts) == 2 {
		version = nameParts[1]
	}

	if opts.UseDistCache {
		logger.Warn().Msg("[EXPERIMENTAL FEATURE] using distillery pass-through cache, this may not work as expected")
	}
	logger.Info().Msg("preparing source")

	src, err := NewSource(name, &provider.Options{
		OS:     opts.OS,
		Arch:   opts.Arch,
		Config: cfg,
		Logger: logger,
		Settings: map[string]interface{}{
			"version":              version,
			"github-token":         opts.GitHubToken,
			"gitlab-token":         opts.GitLabToken,
			"no-signature-verify":  opts.NoSignatureVerify,
			"no-checksum-verify":   opts.NoChecksumVerify,
			"no-score-check":       opts.NoScoreCheck,
			"include-pre-releases": opts.IncludePreReleases,
			"use-dist-cache":       opts.UseDistCache,
			"dist-cache-url":       opts.DistCacheURL,
		},
	})
	if err != nil {
		logger.Error().Bool("fail", true).Msgf("failed to create source: %s", err.Error())
		return err
	}

	if version == common.Latest {
		logger.Info().Msg("resolving latest version")
	}

	if err := src.PreRun(ctx); err != nil {
		logger.Error().Bool("fail", true).Err(err).Msgf("%s", err)
		if strings.Contains(err.Error(), "403") && strings.Contains(err.Error(), "rate limit") {
			logger.Warn().Bool("hint", true).Msg("set DISTILLERY_GITHUB_TOKEN or try --use-dist-cache")
		}
		return err
	}

	logger.Info().Msgf("downloading version %s", src.GetVersion())

	if !opts.Force {
		var installedVersion *inventory.Version

		if version == common.Latest {
			installedVersion = inv.GetLatestVersion(fmt.Sprintf("%s/%s", src.GetSource(), src.GetApp()))
		} else {
			installedVersion = inv.GetBinVersion(fmt.Sprintf("%s/%s", src.GetSource(), src.GetApp()), version)
		}

		if installedVersion != nil && installedVersion.Version == src.GetVersion() {
			logger.Warn().Bool("ok", true).Msgf("version %s is already installed (reinstall with --force)", src.GetVersion())
			return nil
		}
	}

	logger.Info().Msgf("installing version %s", src.GetVersion())

	if err := src.Run(ctx); err != nil {
		logger.Error().Bool("fail", true).Err(err).Msgf("installation failed: %s", err)
		return err
	}

	elapsed := time.Since(startTime)
	logger.Info().Bool("success", true).Msgf("successfully installed version %s in %s", src.GetVersion(), elapsed)

	return nil
}

// OptionsFromCLI builds Options from a cli.Command, for use by the install command's Execute.
func OptionsFromCLI(c *cli.Command, cfg *config.Config) *Options {
	return &Options{
		App:                c.Args().First(),
		Version:            c.String("version"),
		OS:                 c.String("os"),
		Arch:               c.String("arch"),
		Force:              c.Bool("force"),
		Config:             cfg,
		GitHubToken:        c.String("github-token"),
		GitLabToken:        c.String("gitlab-token"),
		NoSignatureVerify:  c.Bool("no-signature-verify"),
		NoChecksumVerify:   c.Bool("no-checksum-verify"),
		NoScoreCheck:       c.Bool("no-score-check"),
		IncludePreReleases: c.Bool("include-pre-releases"),
		UseDistCache:       c.Bool("use-dist-cache"),
		DistCacheURL:       c.String("dist-cache-url"),
		Logger:             log.Logger,
	}
}
