package distfile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ekristen/distillery/pkg/inventory"
)

// specialSources are sources that use 2-part install syntax (source/name@version)
// rather than the standard 3-part syntax (source/owner/repo@version).
var specialSources = map[string]bool{
	"hashicorp": true,
	"homebrew":  true,
}

// githubMetaSources are Owner values under "github" source that represent
// distinct install sources with their own syntax (source/app@version).
var githubMetaSources = map[string]bool{
	"kubernetes": true,
	"helm":       true,
}

// formatInstallLine produces the correct install command for a given bin and version.
func formatInstallLine(bin *inventory.Bin, version string) string {
	// Sources like hashicorp and homebrew store as source/name/version in inventory,
	// but install as source/name@version (Owner is the product name, Repo is the version dir).
	if specialSources[bin.Source] {
		return fmt.Sprintf("install %s/%s@%s", bin.Source, bin.Owner, version)
	}

	// Kubernetes, Helm, etc. are stored under github/ but install as their own source.
	if bin.Source == "github" && githubMetaSources[bin.Owner] {
		return fmt.Sprintf("install %s/%s@%s", bin.Owner, bin.Repo, version)
	}

	return fmt.Sprintf("install %s/%s/%s@%s", bin.Source, bin.Owner, bin.Repo, version)
}

// Build generates a Distfile string from the inventory data.
func Build(inv *inventory.Inventory, latest bool) (string, error) {
	var builder strings.Builder
	seenVersions := make(map[string]bool)

	// Sort bins by their names
	binNames := make([]string, 0, len(inv.Bins))
	for binName := range inv.Bins {
		binNames = append(binNames, binName)
	}
	sort.Strings(binNames)

	for _, binName := range binNames {
		bin := inv.Bins[binName]

		sort.Slice(bin.Versions, func(i, j int) bool {
			return bin.Versions[i].Version < bin.Versions[j].Version
		})

		for _, version := range bin.Versions {
			if latest && !version.Latest {
				continue
			}

			if !seenVersions[version.Version] {
				line := formatInstallLine(bin, version.Version)
				if _, err := fmt.Fprintln(&builder, line); err != nil {
					return "", err
				}
				seenVersions[version.Version] = true
			}
		}
	}

	return builder.String(), nil
}
