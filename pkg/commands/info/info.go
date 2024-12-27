package info

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/apex/log"
	"github.com/urfave/cli/v3"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
)

func Execute(ctx context.Context, c *cli.Command) error {
	cfg, err := config.New(c.String("config"))
	if err != nil {
		return err
	}

	log.Info("version information")
	log.Infof("  distillery/%s", common.AppVersion.Summary)
	fmt.Println("")
	log.Infof("system information")
	log.Infof("     os: %s", runtime.GOOS)
	log.Infof("   arch: %s", runtime.GOARCH)
	fmt.Println("")
	log.Infof("configuration")
	log.Infof("   home: %s", cfg.Path)
	log.Infof("    bin: %s", cfg.BinPath)
	log.Infof("    opt: %s", filepath.FromSlash(cfg.GetOptPath()))
	log.Infof("  cache: %s", filepath.FromSlash(cfg.GetCachePath()))
	fmt.Println("")
	log.Warnf("To cleanup all of distillery, remove the following directories:")
	log.Warnf("  - %s", filepath.FromSlash(cfg.GetCachePath()))
	log.Warnf("  - %s", cfg.BinPath)
	log.Warnf("  - %s", filepath.FromSlash(cfg.GetOptPath()))

	path := os.Getenv("PATH")
	if !strings.Contains(path, cfg.BinPath) {
		fmt.Println("")
		log.Warnf("Problem: distillery will not work correctly")
		log.Warnf("  - %s is not in your PATH", cfg.BinPath)
		fmt.Println("")
	}

	return nil
}

func Flags() []cli.Flag {
	return []cli.Flag{}
}

func init() {
	cmd := &cli.Command{
		Name:        "info",
		Usage:       "info",
		Description: `general information about distillery and the rendered configuration`,
		Flags:       append(Flags(), common.Flags()...),
		Action:      Execute,
	}

	common.RegisterCommand(cmd)
}
