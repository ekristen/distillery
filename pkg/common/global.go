package common

import (
	"fmt"
	"path"
	"runtime"

	"github.com/apex/log"
	clilog "github.com/apex/log/handlers/cli"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
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

	return globalFlags
}

func Before(c *cli.Context) error {
	log.SetHandler(clilog.Default)

	formatter := &logrus.TextFormatter{
		DisableColors: c.Bool("log-disable-color"),
		FullTimestamp: c.Bool("log-full-timestamp"),
	}
	if c.Bool("log-caller") {
		logrus.SetReportCaller(true)

		formatter.CallerPrettyfier = func(f *runtime.Frame) (string, string) {
			return "", fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)
		}
	}

	logrus.SetFormatter(formatter)

	switch c.String("log-level") {
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
		log.SetLevel(log.DebugLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
		log.SetLevel(log.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
		log.SetLevel(log.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
		log.SetLevel(log.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
		log.SetLevel(log.ErrorLevel)
	}

	return nil
}
