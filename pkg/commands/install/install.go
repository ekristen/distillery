package install

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ekristen/distillery/pkg/common"
	source2 "github.com/ekristen/distillery/pkg/source"
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
	metadataDir := filepath.Join(cacheDir, common.NAME, "metadata")
	downloadsDir := filepath.Join(cacheDir, common.NAME, "downloads")
	_ = os.MkdirAll(binDir, 0755)
	_ = os.MkdirAll(metadataDir, 0755)
	_ = os.MkdirAll(downloadsDir, 0755)

	source := source2.New(c.Args().First(), &source2.Options{
		OS:           c.String("os"),
		Arch:         c.String("arch"),
		HomeDir:      homeDir,
		CacheDir:     cacheDir,
		BinDir:       binDir,
		MetadataDir:  metadataDir,
		DownloadsDir: downloadsDir,
	})

	// For now let people know we are not supporting these sources even though we are detecting them
	switch source.(type) {
	case *source2.GitLab:
		return fmt.Errorf("gitlab is not supported yet")
	case *source2.Homebrew:
		return fmt.Errorf("homebrew is not supported yet")
	case *source2.Hashicorp:
		return fmt.Errorf("hashicorp is not supported yet")
	}

	fmt.Println(" source: ", source.GetSource())
	fmt.Println("    app: ", source.GetApp())
	fmt.Println("version: ", c.String("version"))
	fmt.Println("     os: ", c.String("os"))
	fmt.Println("   arch: ", c.String("arch"))

	// list releases using github golang sdk
	// download the binary
	// extract the binary
	// move the binary to the correct location
	// create a symlink to the binary

	if err := source.Run(c.Context, c.String("version"), c.String("github-token")); err != nil {
		return err
	}

	// TODO: inspect file, inspect files, extract files, move files, create symlinks
	fmt.Println("installation complete")

	return nil
}

func Before(c *cli.Context) error {
	if c.NArg() == 0 {
		return fmt.Errorf("no binary specified")
	}

	if c.NArg() > 1 {
		return fmt.Errorf("only one binary can be specified")
	}

	parts := strings.Split(c.Args().First(), "@")
	if len(parts) == 2 {
		_ = c.Set("version", parts[1])
	} else if len(parts) == 1 {
		_ = c.Set("version", "latest")
	} else {
		return fmt.Errorf("invalid binary specified")
	}

	if c.String("bin") != "" {
		_ = c.Set("bins", "false")
	}

	return common.Before(c)
}

func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "version",
			Usage: "Specify a version to install",
			Value: "latest",
		},
		&cli.StringFlag{
			Name:     "asset",
			Usage:    "The exact name of the asset to use, useful when auto-detection fails",
			Category: "Target Selection",
		},
		&cli.StringFlag{
			Name:     "suffix",
			Usage:    "Specify the suffix to use for the binary (default is auto-detect based on OS)",
			Category: "Target Selection",
		},
		&cli.StringFlag{
			Name:     "bin",
			Usage:    "Install only the selected binary",
			Category: "Target Selection",
		},
		&cli.BoolFlag{
			Name:     "bins",
			Usage:    "Install all binaries",
			Category: "Target Selection",
			Value:    true,
		},
		&cli.StringFlag{
			Name:  "os",
			Usage: "Specify the OS to install",
			Value: runtime.GOOS,
		},
		&cli.StringFlag{
			Name:  "arch",
			Usage: "Specify the architecture to install",
			Value: runtime.GOARCH,
		},
		&cli.StringFlag{
			Name:     "github-token",
			Usage:    "GitHub token to use for GitHub API requests",
			EnvVars:  []string{"DISTILLERY_GITHUB_TOKEN"},
			Category: "Authentication",
		},
	}
}

func init() {
	cmd := &cli.Command{
		Name:        "install",
		Usage:       "install",
		Description: fmt.Sprintf(`install a binary. default location is $HOME/.%s/bin`, common.NAME),
		Before:      Before,
		Flags:       append(Flags(), common.Flags()...),
		Action:      Execute,
		Args:        true,
		ArgsUsage:   " binary[@version]",
	}

	common.RegisterCommand(cmd)
}
