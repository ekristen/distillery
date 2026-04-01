package common

import (
	"context"
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"

	"github.com/urfave/cli/v3"

	"github.com/ekristen/distillery/pkg/console"
	"github.com/ekristen/distillery/pkg/spinner"
)

const (
	OutputAuto    = "auto"
	OutputSpinner = "spinner"
	OutputText    = "text"
	OutputJSON    = "json"
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
			Name:     "output",
			Usage:    "Output format: auto, spinner, text, json",
			Aliases:  []string{"o"},
			Sources:  cli.EnvVars("DISTILLERY_OUTPUT"),
			Value:    OutputAuto,
			Category: "Logging Options",
		},
		&cli.BoolFlag{
			Name:     "log-caller",
			Usage:    "include the file/line number of the log entry",
			Sources:  cli.EnvVars("LOG_CALLER"),
			Value:    true,
			Category: "Logging Options",
		},
		// Deprecated: use --output text instead
		&cli.BoolFlag{
			Name:     "no-spinner",
			Usage:    "disable spinner (deprecated: use --output text)",
			Sources:  cli.EnvVars("NO_SPINNER"),
			Value:    false,
			Category: "Logging Options",
			Hidden:   true,
		},
		// Deprecated: use --output json instead
		&cli.StringFlag{
			Name:     "log-format",
			Usage:    "Log output format (deprecated: use --output)",
			Sources:  cli.EnvVars("LOG_FORMAT"),
			Value:    "pretty",
			Category: "Logging Options",
			Hidden:   true,
		},
	}

	return globalFlags
}

// resolveOutputMode determines the effective output mode from flags, handling
// backwards compatibility with --no-spinner and --log-format.
func resolveOutputMode(c *cli.Command) string {
	output := c.String("output")

	// Explicit --output always wins
	if output != OutputAuto {
		return output
	}

	// Backwards compat: --no-spinner maps to text
	if c.Bool("no-spinner") {
		return OutputText
	}

	// Backwards compat: --log-format json maps to json
	if c.String("log-format") == "json" {
		return OutputJSON
	}

	return OutputAuto
}

func Before(ctx context.Context, c *cli.Command) (context.Context, error) {
	logLevelStr := c.String("log-level")
	level, err := zerolog.ParseLevel(logLevelStr)
	if err != nil {
		level = zerolog.InfoLevel
		log.Error().Msgf("invalid log level '%s', defaulting to 'info'. error: %v\n", logLevelStr, err)
	}
	zerolog.SetGlobalLevel(level)

	mode := resolveOutputMode(c)

	// For "auto" mode: spinner if interactive TTY at info level, text otherwise
	if mode == OutputAuto {
		if zerolog.GlobalLevel() == zerolog.InfoLevel && term.IsTerminal(int(os.Stderr.Fd())) {
			mode = OutputSpinner
		} else if zerolog.GlobalLevel() == zerolog.InfoLevel {
			mode = OutputText
		}
	}

	var outputWriter io.Writer
	switch mode {
	case OutputSpinner:
		outputWriter = spinner.NewWriter()
	case OutputText:
		outputWriter = console.NewWriter()
	case OutputJSON:
		outputWriter = os.Stdout
	default:
		// debug/trace levels or any unrecognized mode
		outputWriter = zerolog.ConsoleWriter{Out: os.Stdout}
	}

	if c.Bool("log-caller") {
		log.Logger = zerolog.New(outputWriter).With().Ctx(ctx).Timestamp().Caller().Logger()
	} else {
		log.Logger = zerolog.New(outputWriter).With().Ctx(ctx).Timestamp().Logger()
	}

	return ctx, nil
}
