package run

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/urfave/cli/v3"

	"github.com/rs/zerolog/log"

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

func Execute(ctx context.Context, c *cli.Command) error { //nolint:gocyclo,funlen
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

	parallel := c.Int("parallel")

	if parallel > 1 {
		log.Info().Msgf("running parallel installs with concurrency: %d", parallel)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(commands))

	sem := make(chan struct{}, parallel)

	for i, command := range commands {
		if command.Action == "install" {
			wg.Add(1)
			go func(_ int, command distfile.Command) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				logger := log.With().Str("app", command.Args[0]).Logger()

				installText := fmt.Sprintf("Setting up %s", command.Args[0])
				logger.Info().Msg(installText)

				args := append([]string{"install"}, command.Args...)
				if installErr := instCmd.Run(ctx, args); installErr != nil {
					errCh <- installErr
					logger.Error().Msgf("Failed %s: %s", command.Args[0], installErr.Error())
				} else {
					logger.Info().Msgf("Completed %s", command.Args[0])
				}
			}(i, command)
		} else {
			// this is for any other action that's detected that we don't support right now
			wg.Done()
		}

		select {
		case <-ctx.Done(): //nolint:staticcheck
			return nil
		default:
			continue
		}
	}

	wg.Wait()
	close(errCh)

	var didError bool
	for err := range errCh {
		if err != nil {
			didError = true
		}
	}

	if didError {
		return errors.New("one or more install commands failed")
	}

	return nil
}

func init() {
	flags := []cli.Flag{
		&cli.IntFlag{
			Name:    "parallel",
			Aliases: []string{"p"},
			Usage:   "EXPERIMENTAL FEATURE: number of parallel installs to run",
			Value:   1,
		},
	}

	cmd := &cli.Command{
		Name:        "run",
		Usage:       "run [Distfile]",
		Description: `run a Distfile to install binaries`,
		Action:      Execute,
		Before:      common.Before,
		Flags:       append(flags, common.Flags()...),
	}

	common.RegisterCommand(cmd)
}
