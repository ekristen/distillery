package main

import (
	"context"
	"os"
	"path"
	"strings"

	"github.com/apex/log"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/signals"

	_ "github.com/ekristen/distillery/pkg/commands/clean"
	_ "github.com/ekristen/distillery/pkg/commands/completion"
	_ "github.com/ekristen/distillery/pkg/commands/info"
	_ "github.com/ekristen/distillery/pkg/commands/install"
	_ "github.com/ekristen/distillery/pkg/commands/list"
	_ "github.com/ekristen/distillery/pkg/commands/proof"
	_ "github.com/ekristen/distillery/pkg/commands/run"
	_ "github.com/ekristen/distillery/pkg/commands/uninstall"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			// log panics forces exit
			if _, ok := r.(*logrus.Entry); ok {
				os.Exit(1)
			}
			panic(r)
		}
	}()

	app := &cli.Command{
		Name:        path.Base(os.Args[0]),
		Usage:       `install any binary from ideally any source`,
		Description: `install any binary from ideally any detectable source`,
		Version:     strings.TrimLeft(common.AppVersion.Summary, "v"),
		Before:      common.Before,
		Flags:       common.Flags(),
		Commands:    common.GetCommands(),
		CommandNotFound: func(ctx context.Context, c *cli.Command, command string) {
			log.Fatalf("command %s not found.", command)
		},
		EnableShellCompletion: true,
		Authors: []any{
			"Erik Kristensen <erik@erikkristensen.com>",
		},
	}

	ctx := signals.SetupSignalContext()
	if err := app.Run(ctx, os.Args); err != nil {
		log.Error(err.Error())
	}
}
