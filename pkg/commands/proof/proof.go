package proof

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/urfave/cli/v3"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/distfile"
	"github.com/ekristen/distillery/pkg/inventory"
)

func Execute(ctx context.Context, c *cli.Command) error {
	cfg, err := config.New(c.String("config"))
	if err != nil {
		return err
	}

	if err := cfg.MkdirAll(); err != nil {
		return err
	}

	inv := inventory.New(os.DirFS(cfg.BinPath), cfg.BinPath, cfg.GetOptPath(), cfg)

	df, err := distfile.Build(inv, c.Bool("latest-only"))
	if err != nil {
		return err
	}

	fmt.Println(df)

	return nil
}

func init() {
	cfgDir, _ := os.UserConfigDir()
	homeDir, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		cfgDir = filepath.Join(homeDir, ".config")
	}

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Specify the configuration file to use",
			Sources: cli.EnvVars("DISTILLERY_CONFIG"),
			Value:   filepath.Join(cfgDir, fmt.Sprintf("%s.yaml", common.NAME)),
		},
		&cli.BoolFlag{
			Name:    "latest-only",
			Aliases: []string{"l"},
			Usage:   "Include only the latest version of each binary in the proof",
			Sources: cli.EnvVars("DISTILLERY_PROOF_LATEST_ONLY"),
		},
	}

	cmd := &cli.Command{
		Name:    "proof",
		Aliases: []string{"export"},
		Usage:   "proof",
		Flags:   flags,
		Action:  Execute,
	}

	common.RegisterCommand(cmd)
}
