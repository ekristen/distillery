package clean

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/urfave/cli/v3"

	"github.com/ekristen/distillery/pkg/common"
)

func Execute(c *cli.Context) error { //nolint:gocyclo
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	binDir := filepath.Join(homeDir, fmt.Sprintf(".%s", common.NAME), "bin")

	sims := make(map[string]map[string]string)
	targets := make([]string, 0)
	bins := make([]string, 0)

	if !c.Bool("no-dry-run") {
		log.Warn("dry-run enabled, no changes will be made, use --no-dry-run to perform actions")
	}

	_ = filepath.Walk(binDir, func(path string, info os.FileInfo, err error) error {
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
			simpleName := info.Name()
			version := "latest"
			parts := strings.Split(info.Name(), "@")
			if len(parts) > 1 {
				simpleName = parts[0]
				version = parts[1]
			}

			// Get the target of the symlink
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}

			targets = append(targets, target)

			if _, ok := sims[simpleName]; !ok {
				sims[simpleName] = make(map[string]string)
			}

			sims[simpleName][version] = path
		} else {
			bins = append(bins, path)
		}

		return nil
	})

	log.Warn("orphaned binaries:")
	for _, path := range bins {
		found := false

		for _, p := range targets {
			if path == p {
				found = true
				break
			}
		}

		if found {
			continue
		}

		log.Warnf("  - %s", path)

		if c.Bool("no-dry-run") {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}

	return nil
}

func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "no-dry-run",
			Usage: "Perform all actions",
		},
	}
}

func init() {
	cmd := &cli.Command{
		Name:        "clean",
		Usage:       "clean",
		Description: `cleanup`,
		Flags:       append(Flags(), common.Flags()...),
		Action:      Execute,
	}

	common.RegisterCommand(cmd)
}
