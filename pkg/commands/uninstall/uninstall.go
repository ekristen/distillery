package uninstall

import (
	"github.com/ekristen/distillery/pkg/common"
	"github.com/urfave/cli/v2"
)

func Execute(c *cli.Context) error {
	return nil
}

func init() {
	cmd := &cli.Command{
		Name:        "uninstall",
		Usage:       "uninstall",
		Description: `list installed binaries`,
		Before:      common.Before,
		Flags:       common.Flags(),
		Action:      Execute,
	}

	common.RegisterCommand(cmd)
}
