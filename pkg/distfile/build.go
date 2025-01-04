package distfile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ekristen/distillery/pkg/inventory"
)

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
				_, err := fmt.Fprintf(&builder, "install %s/%s/%s@%s\n", bin.Source, bin.Owner, bin.Repo, version.Version)
				if err != nil {
					return "", err
				}
				seenVersions[version.Version] = true
			}
		}
	}

	return builder.String(), nil
}
