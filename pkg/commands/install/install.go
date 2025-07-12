package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v3"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/inventory"
	"github.com/ekristen/distillery/pkg/provider"
	"github.com/ekristen/distillery/pkg/spinner"
)

func Execute(ctx context.Context, c *cli.Command) error { //nolint:funlen
	startTime := time.Now().UTC()

	appName := c.Args().First()

	logger := zerolog.New(spinner.NewWriter()).With().Ctx(ctx).Timestamp().Str("app", appName).Logger()

	logger.Info().Msg("starting installation")

	cfg, err := config.New(c.String("config"))
	if err != nil {
		logger.Error().Msg("failed to load configuration")
		return err
	}

	if err := cfg.MkdirAll(); err != nil {
		logger.Error().Msg("failed to create directories")
		return err
	}

	inv := inventory.New(os.DirFS(cfg.BinPath), cfg.BinPath, cfg.GetOptPath(), cfg)

	name := c.Args().First()
	nameParts := strings.Split(name, "@")
	alias := cfg.GetAlias(nameParts[0])
	if alias != nil {
		name = alias.Name
		version := alias.Version
		if len(nameParts) > 1 {
			if version != "latest" {
				logger.Warn().Msg("version specified via cli and alias, ignoring alias version")
			}
			version = nameParts[1]
		}

		_ = c.Set("version", version)

		name = fmt.Sprintf("%s@%s", name, version)
	}

	if c.Bool("use-dist-cache") {
		logger.Warn().Msg("[EXPERIMENTAL FEATURE] using distillery pass-through cache, this may not work as expected")
	}
	logger.Info().Msg("preparing source")

	src, err := NewSource(name, &provider.Options{
		OS:     c.String("os"),
		Arch:   c.String("arch"),
		Config: cfg,
		Logger: logger,
		Settings: map[string]interface{}{
			"version":              c.String("version"),
			"github-token":         c.String("github-token"),
			"gitlab-token":         c.String("gitlab-token"),
			"no-signature-verify":  c.Bool("no-signature-verify"),
			"no-checksum-verify":   c.Bool("no-checksum-verify"),
			"no-score-check":       c.Bool("no-score-check"),
			"include-pre-releases": c.Bool("include-pre-releases"),
			"use-dist-cache":       c.Bool("use-dist-cache"),
			"dist-cache-url":       c.String("dist-cache-url"),
		},
	})

	if err != nil {
		logger.Error().Msgf("failed to create source: %s", err.Error())
		return err
	}

	if c.String("version") == common.Latest {
		logger.Info().Msg("resolving latest version")
	}

	if err := src.PreRun(ctx); err != nil {
		logger.Error().Err(err).Msg("failed to prepare installation")
		return err
	}

	logger.Info().Err(err).Msgf("downloading version %s", src.GetVersion())

	if !c.Bool("force") {
		var installedVersion *inventory.Version

		if c.String("version") == common.Latest {
			installedVersion = inv.GetLatestVersion(fmt.Sprintf("%s/%s", src.GetSource(), src.GetApp()))
		} else {
			installedVersion = inv.GetBinVersion(fmt.Sprintf("%s/%s", src.GetSource(), src.GetApp()), c.String("version"))
		}

		if installedVersion != nil && installedVersion.Version == src.GetVersion() {
			logger.Warn().Bool("ok", true).Msgf("version %s is already installed (reinstall with --force)", src.GetVersion())
			return nil
		}
	}

	logger.Info().Msgf("installing version %s", src.GetVersion())

	if err := src.Run(ctx); err != nil {
		logger.Error().Err(err).Msg("installation failed")
		return err
	}

	endTime := time.Now().UTC()
	elapsed := endTime.Sub(startTime)

	logger.Info().Bool("success", true).Msgf("successfully installed version %s in %s", src.GetVersion(), elapsed)

	return nil
}

func Before(ctx context.Context, c *cli.Command) (context.Context, error) {
	if c.NArg() == 0 {
		return ctx, fmt.Errorf("no binary specified")
	}

	if c.NArg() > 1 {
		for _, arg := range c.Args().Slice() {
			if strings.HasPrefix(arg, "-") {
				return ctx, fmt.Errorf("flags must be specified before the binary(ies)")
			}
		}

		return ctx, fmt.Errorf("currently only one binary can be installed at a time")
	}

	parts := strings.Split(c.Args().First(), "@")
	if len(parts) == 2 {
		_ = c.Set("version", parts[1])
	} else if len(parts) == 1 {
		_ = c.Set("version", "latest")
	} else {
		return ctx, fmt.Errorf("invalid binary specified")
	}

	if c.String("bin") != "" {
		_ = c.Set("bins", "false")
	}

	return common.Before(ctx, c)
}

func Flags() []cli.Flag { //nolint:funlen
	cfgDir, _ := os.UserConfigDir()
	homeDir, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		cfgDir = filepath.Join(homeDir, ".config")
	}

	return []cli.Flag{
		&cli.StringFlag{
			Name:  "version",
			Usage: "Specify a version to install",
			Value: "latest",
		},
		&cli.StringFlag{
			Name:     "asset",
			Usage:    "The exact name of the asset to use, useful when auto-detection fails",
			Category: "Target Selection",
			Hidden:   true,
		},
		&cli.StringFlag{
			Name:     "suffix",
			Usage:    "Specify the suffix to use for the binary (default is auto-detect based on OS)",
			Category: "Target Selection",
			Hidden:   true,
		},
		&cli.StringFlag{
			Name:     "bin",
			Usage:    "Install only the selected binary",
			Category: "Target Selection",
			Hidden:   true,
		},
		&cli.BoolFlag{
			Name:     "bins",
			Usage:    "Install all binaries",
			Category: "Target Selection",
			Value:    true,
			Hidden:   true,
		},
		&cli.StringFlag{
			Name:  "os",
			Usage: "Specify the OS to install",
			Value: runtime.GOOS,
		},
		&cli.StringFlag{
			Name:  "arch",
			Usage: "Specify the architecture to install",
			Value: runtime.GOARCH,
		},
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Specify the configuration file to use",
			Sources: cli.EnvVars("DISTILLERY_CONFIG"),
			Value:   filepath.Join(cfgDir, fmt.Sprintf("%s.yaml", common.NAME)),
		},
		&cli.StringFlag{
			Name:     "github-token",
			Usage:    "GitHub token to use for GitHub API requests",
			Sources:  cli.EnvVars("DISTILLERY_GITHUB_TOKEN"),
			Category: "Authentication",
		},
		&cli.StringFlag{
			Name:     "gitlab-token",
			Usage:    "GitLab token to use for GitLab API requests",
			Sources:  cli.EnvVars("DISTILLERY_GITLAB_TOKEN"),
			Category: "Authentication",
		},
		&cli.BoolFlag{
			Name:    "include-pre-releases",
			Usage:   "include pre-releases in the list of available versions",
			Sources: cli.EnvVars("DISTILLERY_INCLUDE_PRE_RELEASES"),
			Aliases: []string{"pre"},
		},
		&cli.BoolFlag{
			Name:    "no-checksum-verify",
			Usage:   "disable checksum verification",
			Sources: cli.EnvVars("DISTILLERY_NO_CHECKSUM_VERIFY"),
		},
		&cli.BoolFlag{
			Name:    "no-signature-verify",
			Usage:   "disable signature verification",
			Sources: cli.EnvVars("DISTILLERY_NO_SIGNATURE_VERIFY"),
		},
		&cli.BoolFlag{
			Name:  "no-score-check",
			Usage: "disable scoring check",
		},
		&cli.BoolFlag{
			Name:  "force",
			Usage: "force the installation of the binary even if it is already installed",
		},
		&cli.BoolFlag{
			Name:    "use-dist-cache",
			Sources: cli.EnvVars("DISTILLERY_USE_CACHE"),
			Usage:   "[EXPERIMENTAL] use the distillery pass-through cache for github to avoid authentication",
		},
		&cli.StringFlag{
			Name:    "dist-cache-url",
			Value:   "https://api.github.cache.dist.sh",
			Sources: cli.EnvVars("DISTILLERY_CACHE_URL"),
			Usage: "[EXPERIMENTAL] specify the base url for the distillery pass-through cache" +
				" for github to avoid authentication and rate limiting",
			Hidden: true,
		},
	}
}

func init() {
	cmd := &cli.Command{
		Name:        "install",
		Usage:       "install [provider/]owner/repo[@version]",
		Description: fmt.Sprintf(`install binaries fast. default location is $HOME/.%s/bin`, common.NAME),
		Before:      Before,
		Flags:       append(Flags(), common.Flags()...),
		Action:      Execute,
		ArgsUsage:   "[provider/]owner/repo[@version]",
	}

	common.RegisterCommand(cmd)
}
