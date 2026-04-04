package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
)

func Execute(ctx context.Context, c *cli.Command) error {
	cfg, err := config.New(c.String("config"))
	if err != nil {
		log.Error().Bool("fail", true).Err(err).Msgf("failed to load configuration: %s", err)
		return err
	}

	if err := cfg.MkdirAll(); err != nil {
		log.Error().Bool("fail", true).Err(err).Msgf("failed to create directories: %s", err)
		return err
	}

	apps := c.Args().Slice()

	// Single app: run directly
	if len(apps) == 1 {
		opts := OptionsFromCLI(c, cfg)
		return DoInstall(ctx, opts)
	}

	// Multiple apps: run concurrently
	var wg sync.WaitGroup
	errCh := make(chan error, len(apps))

	for _, app := range apps {
		wg.Add(1)
		go func(app string) {
			defer wg.Done()
			opts := OptionsFromCLI(c, cfg)
			opts.App = app
			// Strip hint before parsing version (hint is handled later in NewSource)
			cleaned, _ := extractHint(app)
			parts := strings.SplitN(cleaned, "@", 2)
			if len(parts) == 2 {
				opts.Version = parts[1]
			}
			if err := DoInstall(ctx, opts); err != nil {
				errCh <- err
			}
		}(app)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d of %d installs failed", len(errs), len(apps))
	}

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
	}

	// Strip binary hint before parsing version (hint is handled later in NewSource)
	firstArg, _ := extractHint(c.Args().First())
	parts := strings.Split(firstArg, "@")
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
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Specify the configuration file to use",
			Sources: cli.EnvVars("DISTILLERY_CONFIG"),
			Value:   filepath.Join(cfgDir, fmt.Sprintf("%s.yaml", common.NAME)),
		},
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
		&cli.StringFlag{
			Name:     "forgejo-token",
			Usage:    "Forgejo token to use for Forgejo/Codeberg API requests",
			Sources:  cli.EnvVars("DISTILLERY_FORGEJO_TOKEN"),
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
		Usage:       "install [provider/]owner/repo[:binary][@version]",
		Description: fmt.Sprintf(`install binaries fast. default location is $HOME/.%s/bin`, common.NAME),
		Before:      Before,
		Flags:       append(Flags(), common.Flags()...),
		Action:      Execute,
		ArgsUsage:   "[provider/]owner/repo[:binary][@version]",
	}

	common.RegisterCommand(cmd)
}
