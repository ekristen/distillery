package info

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
)

func Execute(ctx context.Context, c *cli.Command) error {
	cfg, err := config.New(c.String("config"))
	if err != nil {
		return err
	}

	log.Info().Msg("version information")
	log.Info().Msgf("  distillery/%s", common.AppVersion.Summary)
	fmt.Println("")
	log.Info().Msg("system information")
	log.Info().Msgf("     os: %s", runtime.GOOS)
	log.Info().Msgf("   arch: %s", runtime.GOARCH)
	fmt.Println("")
	log.Info().Msg("configuration")
	log.Info().Msgf("   home: %s", cfg.Path)
	log.Info().Msgf("    bin: %s", cfg.BinPath)
	log.Info().Msgf("    opt: %s", filepath.FromSlash(cfg.GetOptPath()))
	log.Info().Msgf("  cache: %s", filepath.FromSlash(cfg.GetCachePath()))
	fmt.Println("")
	log.Warn().Msg("To cleanup all of distillery, remove the following directories:")
	log.Warn().Msgf("  - %s", filepath.FromSlash(cfg.GetCachePath()))
	log.Warn().Msgf("  - %s", cfg.BinPath)
	log.Warn().Msgf("  - %s", filepath.FromSlash(cfg.GetOptPath()))

	path := os.Getenv("PATH")
	if !strings.Contains(path, cfg.BinPath) {
		fmt.Println("")
		log.Warn().Msg("Problem: distillery will not work correctly")
		log.Warn().Msgf("  - %s is not in your PATH", cfg.BinPath)
		fmt.Println("")
	}

	return nil
}

func Flags() []cli.Flag {
	return []cli.Flag{}
}

func Before(ctx context.Context, c *cli.Command) (context.Context, error) {
	_ = c.Set("no-spinner", "true")
	_ = c.Set("log-caller", "false")

	return common.Before(ctx, c)
}

func init() {
	cmd := &cli.Command{
		Name:        "info",
		Usage:       "info",
		Description: `general information about distillery and the rendered configuration`,
		Flags:       append(Flags(), common.Flags()...),
		Before:      Before,
		Action:      Execute,
	}

	common.RegisterCommand(cmd)
}
