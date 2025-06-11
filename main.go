package main

import (
	"os"
	"path"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

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
	ctx := signals.SetupSignalContext()

	defer func() {
		if r := recover(); r != nil {
			// Log panics and exit
			if err, ok := r.(error); ok {
				log.Error().Bool("fail", true).Err(err).Msgf("fatal error: %s", err.Error())
				os.Exit(1)
			}
			panic(r)
		}
	}()

	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = `install any binary from ideally any source`
	app.Description = `install any binary from ideally any detectable source`
	app.Version = common.AppVersion.Summary
	app.Authors = []*cli.Author{
		{
			Name:  "Erik Kristensen",
			Email: "erik@erikkristensen.com",
		},
	}

	app.Before = common.Before
	app.Flags = common.Flags()

	app.Commands = common.GetCommands()
	app.CommandNotFound = func(context *cli.Context, command string) {
		log.Fatal().Bool("warn", true).Msgf("command %s not found", command)
	}

	app.EnableBashCompletion = true

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Error().Bool("fail", true).Err(err).Msgf("command failed: %s", err.Error())
	}
}
