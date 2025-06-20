package common

import (
	"context"
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/urfave/cli/v3"

	"github.com/ekristen/distillery/pkg/spinner"
)

func Flags() []cli.Flag {
	globalFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     "log-level",
			Usage:    "Log Level",
			Aliases:  []string{"l"},
			Sources:  cli.EnvVars("LOG_LEVEL"),
			Value:    "info",
			Category: "Logging Options",
		},
		&cli.StringFlag{
			Name:     "log-format",
			Usage:    "Log output format (pretty or json)",
			Sources:  cli.EnvVars("LOG_FORMAT"),
			Value:    "pretty",
			Category: "Logging Options",
		},
		&cli.BoolFlag{
			Name:     "log-caller",
			Usage:    "include the file/line number of the log entry",
			Sources:  cli.EnvVars("LOG_CALLER"),
			Value:    true,
			Category: "Logging Options",
		},
		&cli.BoolFlag{
			Name:     "no-spinner",
			Usage:    "disable spinner",
			Sources:  cli.EnvVars("NO_SPINNER"),
			Value:    false,
			Category: "Logging Options",
			Hidden:   true,
		},
	}

	return globalFlags
}

func Before(ctx context.Context, c *cli.Command) (context.Context, error) {
	logLevelStr := c.String("log-level")
	level, err := zerolog.ParseLevel(logLevelStr)
	if err != nil {
		// Fallback to info if parsing fails
		level = zerolog.InfoLevel
		log.Error().Msgf("invalid log level '%s', defaulting to 'info'. error: %v\n", logLevelStr, err)
	}
	zerolog.SetGlobalLevel(level)

	var outputWriter io.Writer
	if zerolog.GlobalLevel() == zerolog.InfoLevel && !c.Bool("no-spinner") {
		outputWriter = spinner.NewWriter()
	} else if c.String("log-format") == "json" || c.Bool("json") {
		outputWriter = os.Stdout
	} else {
		outputWriter = zerolog.ConsoleWriter{
			Out: os.Stdout,
		}
	}

	if c.Bool("log-caller") {
		log.Logger = zerolog.New(outputWriter).With().Ctx(ctx).Timestamp().Caller().Logger()
	} else {
		log.Logger = zerolog.New(outputWriter).With().Ctx(ctx).Timestamp().Logger()
	}

	return ctx, nil
}
