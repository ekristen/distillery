package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestFlags(t *testing.T) {
	expectedFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     "log-level",
			Usage:    "Log Level",
			Aliases:  []string{"l"},
			EnvVars:  []string{"LOG_LEVEL"},
			Value:    "info",
			Category: "Logging Options",
		},
		&cli.BoolFlag{
			Name:     "log-caller",
			Usage:    "log the caller (aka line number and file)",
			Category: "Logging Options",
		},
		&cli.BoolFlag{
			Name:     "log-disable-color",
			Usage:    "disable log coloring",
			Category: "Logging Options",
		},
		&cli.BoolFlag{
			Name:     "log-full-timestamp",
			Usage:    "force log output to always show full timestamp",
			Category: "Logging Options",
		},
	}

	flags := Flags()
	assert.Equal(t, expectedFlags, flags)
}
