package run

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/urfave/cli/v2"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/distfile"
)

func discover(cwd string) (string, error) {
	localDistfile := filepath.Join(cwd, "Distfile")
	if _, err := os.Stat(localDistfile); err == nil {
		return localDistfile, nil
	}

	// Check $HOME directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	homeDistfile := filepath.Join(homeDir, "Distfile")
	if _, err := os.Stat(homeDistfile); err == nil {
		return homeDistfile, nil
	}

	// If neither exist, return an error
	return "", errors.New("no Distfile found in current directory or $HOME")
}

func Execute(c *cli.Context) error {
	var df string
	if c.Args().Len() == 0 {
		// Check current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		df, err = discover(cwd)
		if err != nil {
			return err
		}
	} else {
		df = c.Args().First()
		if _, err := os.Stat(df); err != nil {
			return errors.New("no Distfile found")
		}
	}

	cfg, err := config.New(c.String("config"))
	if err != nil {
		return err
	}

	if err := cfg.MkdirAll(); err != nil {
		return err
	}

	commands, err := distfile.Parse(df)
	if err != nil {
		return err
	}

	instCmd := common.GetCommand("install")

	didError := false
	for _, command := range commands {
		if command.Action == "install" {
			ctx := cli.NewContext(c.App, nil, nil)
			a := []string{"install"}
			a = append(a, command.Args...)
			if err := instCmd.Run(ctx, a...); err != nil {
				didError = true
				log.WithError(err).Error("error running install command")
			}
		}

		select { //nolint:gosimple
		case <-c.Context.Done():
			return nil
		}
	}

	if didError {
		return errors.New("one or more install commands failed")
	}

	return nil
}

func init() {
	cmd := &cli.Command{
		Name:        "run",
		Usage:       "run [Distfile]",
		Description: `run a Distfile to install binaries`,
		Action:      Execute,
	}

	common.RegisterCommand(cmd)
}
