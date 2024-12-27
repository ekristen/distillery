package proof

import (
	"context"
	"fmt"
	"os"

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

	df, err := distfile.Build(inv)
	if err != nil {
		return err
	}

	fmt.Println(df)

	return nil
}

func init() {
	cmd := &cli.Command{
		Name:    "proof",
		Aliases: []string{"export"},
		Usage:   "generate a Distfile",
		Action:  Execute,
	}

	common.RegisterCommand(cmd)
}
