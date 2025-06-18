package list

import (
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/inventory"
)

func Execute(c *cli.Context) error {
	cfg, err := config.New(c.String("config"))
	if err != nil {
		return err
	}

	inv := inventory.New(os.DirFS(cfg.GetPath()), cfg.GetPath(), cfg.GetOptPath(), cfg)

	for _, key := range inv.GetBinsSortedKeys() {
		bin := inv.Bins[key]
		log.Info().Msgf("%s (versions: %s)", key, strings.Join(bin.ListVersions(), ", "))
	}

	return nil
}

func Before(c *cli.Context) error {
	_ = c.Set("no-spinner", "true")
	_ = c.Set("log-caller", "false")

	return common.Before(c)
}

func init() {
	cmd := &cli.Command{
		Name:        "list",
		Usage:       "list installed binaries and versions",
		Description: `list installed binaries and versions`,
		Before:      Before,
		Flags:       common.Flags(),
		Action:      Execute,
	}

	common.RegisterCommand(cmd)
}
