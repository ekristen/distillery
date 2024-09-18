package info

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/apex/log"
	"github.com/urfave/cli/v2"

	"github.com/ekristen/distillery/pkg/common"
)

func Execute(c *cli.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	binDir := filepath.Join(homeDir, fmt.Sprintf(".%s", common.NAME), "bin")
	optDir := filepath.Join(homeDir, fmt.Sprintf(".%s", common.NAME), "opt")

	log.Infof("distillery/%s", common.AppVersion.Summary)
	log.Infof("     os: %s", runtime.GOOS)
	log.Infof("   arch: %s", runtime.GOARCH)
	log.Infof("   home: %s", homeDir)
	log.Infof("    bin: %s", binDir)
	log.Infof("    opt: %s", optDir)
	log.Infof("  cache: %s", filepath.Join(cacheDir, common.NAME))

	log.Warnf("To cleanup all of distillery, remove the following directories:")
	log.Warnf("  - %s", filepath.Join(cacheDir, common.NAME))
	log.Warnf("  - %s", filepath.Join(homeDir, fmt.Sprintf(".%s", common.NAME)))

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