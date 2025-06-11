package common

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/urfave/cli/v2"

	"github.com/ekristen/distillery/pkg/spinner"
)

func Flags() []cli.Flag {
	globalFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     "log-level",
			Usage:    "Log Level",
			Aliases:  []string{"l"},
			EnvVars:  []string{"LOG_LEVEL"},
			Value:    "info",
			Category: "Logging Options",
		},
		&cli.StringFlag{
			Name:     "log-format",
			Usage:    "Log output format (pretty or json)",
			EnvVars:  []string{"LOG_FORMAT"},
			Value:    "pretty",
			Category: "Logging Options",
		},
		&cli.BoolFlag{
			Name:     "log-caller",
			Usage:    "include the file/line number of the log entry",
			EnvVars:  []string{"LOG_CALLER"},
			Value:    true,
			Category: "Logging Options",
		},
	}

	return globalFlags
}

func Before(c *cli.Context) error {
	logLevelStr := c.String("log-level")
	level, err := zerolog.ParseLevel(logLevelStr)
	if err != nil {
		// Fallback to info if parsing fails
		level = zerolog.InfoLevel
		log.Error().Msgf("invalid log level '%s', defaulting to 'info'. error: %v\n", logLevelStr, err)
	}
	zerolog.SetGlobalLevel(level)

	var outputWriter io.Writer
	outputWriter = zerolog.ConsoleWriter{
		Out: os.Stdout,
	}
	if zerolog.GlobalLevel() == zerolog.InfoLevel {
		outputWriter = spinner.NewWriter()
	}
	if c.String("log-format") == "json" || c.Bool("json") {
		outputWriter = os.Stdout
	}

	if c.Bool("log-caller") {
		log.Logger = zerolog.New(outputWriter).With().Ctx(c.Context).Timestamp().Caller().Logger()
	} else {
		log.Logger = zerolog.New(outputWriter).With().Ctx(c.Context).Timestamp().Logger()
	}

	return nil
}
