package uninstall

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/apex/log"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/provider"

	"github.com/ekristen/distillery/pkg/commands/install"
)

func Execute(ctx context.Context, c *cli.Command) error {
	cfg, err := config.New(c.String("config"))
	if err != nil {
		return err
	}

	src, err := install.NewSource(c.Args().First(), &provider.Options{
		OS:     c.String("os"),
		Arch:   c.String("arch"),
		Config: cfg,
		Settings: map[string]interface{}{
			"version":              c.String("version"),
			"github-token":         c.String("github-token"),
			"gitlab-token":         c.String("gitlab-token"),
			"no-checksum-verify":   c.Bool("no-checksum-verify"),
			"no-score-check":       c.Bool("no-score-check"),
			"include-pre-releases": c.Bool("include-pre-releases"),
		},
	})
	if err != nil {
		return err
	}

	path := filepath.Join(cfg.GetOptPath(), src.GetSource(), src.GetOwner(), src.GetRepo())

	logrus.Trace("path: ", path)

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			log.Warnf("%s does not appear to be installed", c.Args().First())
			return nil
		}

		return err
	}

	if !c.Bool("no-dry-run") {
		log.Warn("dry-run enabled, no changes will be made, use --no-dry-run to perform actions")
	}

	var files []string

	bins, err := discoverBins(path)
	if err != nil {
		return err
	}

	symlinks, err := discoverSymlinks(cfg.BinPath, bins)
	if err != nil {
		return err
	}

	files = append(files, bins...)
	files = append(files, symlinks...)

	msg := "will remove"
	if c.Bool("no-dry-run") {
		msg = "removed"
	}

	for _, file := range files {
		log.Warnf("%s - %s", msg, file)

		if c.Bool("no-dry-run") {
			if err := os.Remove(file); err != nil {
				return err
			}
		}
	}

	log.Warnf("%s - %s", msg, path)

	if c.Bool("no-dry-run") {
		if err := os.RemoveAll(path); err != nil {
			return err
		}

		log.Info("uninstall complete")
	}

	return nil
}

func Before(ctx context.Context, c *cli.Command) (context.Context, error) {
	if c.NArg() == 0 {
		return ctx, fmt.Errorf("no binary specified")
	}

	if c.NArg() > 1 {
		for _, arg := range c.Args().Slice() {
			if arg == "--no-dry-run" {
				_ = c.Set("no-dry-run", "true")
			} else if strings.HasPrefix(arg, "-") {
				return ctx, fmt.Errorf("flags must be specified before the binary(ies)")
			}
		}
	}

	parts := strings.Split(c.Args().First(), "@")
	if len(parts) == 2 {
		_ = c.Set("version", parts[1])
	} else if len(parts) == 1 {
		_ = c.Set("version", "latest")
	} else {
		return ctx, fmt.Errorf("invalid binary specified")
	}

	if c.String("bin") != "" {
		_ = c.Set("bins", "false")
	}

	return common.Before(ctx, c)
}

func Flags() []cli.Flag {
	cfgDir, _ := os.UserConfigDir()

	return []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Specify the configuration file to use",
			Value:   filepath.Join(cfgDir, fmt.Sprintf("%s.yaml", common.NAME)),
			Sources: cli.EnvVars("DISTILLERY_CONFIG"),
		},
		&cli.BoolFlag{
			Name:  "no-dry-run",
			Usage: "Perform all actions",
		},
	}
}

func init() {
	cmd := &cli.Command{
		Name:        "uninstall",
		Usage:       "uninstall binaries",
		Description: `uninstall binaries and all versions`,
		Before:      Before,
		Flags:       append(Flags(), common.Flags()...),
		Action:      Execute,
	}

	common.RegisterCommand(cmd)
}

func discoverSymlinks(path string, bins []string) ([]string, error) {
	var symlinks []string

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		fileInfo, err := os.Lstat(path)
		if err != nil {
			return err
		}

		if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}

			if !slices.Contains(bins, target) {
				return nil
			}

			symlinks = append(symlinks, path)
		}
		return nil
	})
	if err != nil {
		return symlinks, err
	}

	return symlinks, nil
}

func discoverBins(path string) ([]string, error) {
	var bins []string

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		fileInfo, err := os.Lstat(path)
		if err != nil {
			return err
		}

		if fileInfo.Mode()&os.ModeSymlink != os.ModeSymlink {
			bins = append(bins, path)
		}

		return nil
	})
	if err != nil {
		return bins, err
	}

	return bins, nil
}
