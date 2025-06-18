package list

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/inventory"
)

func Execute(c *cli.Context) error {
	cfg, err := config.New(c.String("config"))
	if err != nil {
		return err
	}

	inv := inventory.New(os.DirFS(cfg.GetPath()), cfg.GetPath(), cfg.GetOptPath(), cfg)

	tableData := pterm.TableData{{"Name", "Versions"}}
	for _, key := range inv.GetBinsSortedKeys() {
		bin := inv.Bins[key]
		versions := bin.ListVersions()
		// Sort lexicographically
		sort.Strings(versions)
		// Reverse to get latest first
		for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
			versions[i], versions[j] = versions[j], versions[i]
		}
		displayVersions := versions
		extra := ""
		if len(versions) > 3 {
			displayVersions = versions[:3]
			extra = " (+" + strconv.Itoa(len(versions)-3) + ")"
		}
		tableData = append(tableData, []string{key, strings.Join(displayVersions, ", ") + extra})
	}
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()

	return nil
}

func Before(c *cli.Context) error {
	_ = c.Set("no-spinner", "true")
	_ = c.Set("log-caller", "false")

	return common.Before(c)
}

func init() {
	cmd := &cli.Command{
		Name:        "list",
		Usage:       "list installed binaries and versions",
		Description: `list installed binaries and versions`,
		Before:      Before,
		Flags:       common.Flags(),
		Action:      Execute,
	}

	common.RegisterCommand(cmd)
}
