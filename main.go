package main

import (
	"os"
	"path"

	"github.com/apex/log"
	"github.com/rancher/wrangler/v3/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/ekristen/distillery/pkg/common"

	_ "github.com/ekristen/distillery/pkg/commands/clean"
	_ "github.com/ekristen/distillery/pkg/commands/completion"
	_ "github.com/ekristen/distillery/pkg/commands/info"
	_ "github.com/ekristen/distillery/pkg/commands/install"
	_ "github.com/ekristen/distillery/pkg/commands/list"
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
		log.Fatalf("command %s not found.", command)
	}

	app.EnableBashCompletion = true

	ctx := signals.SetupSignalContext()
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Error(err.Error())
	}
}
