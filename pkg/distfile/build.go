package distfile

import (
	"fmt"
	"strings"

	"github.com/ekristen/distillery/pkg/inventory"
)

// Build generates a Distfile string from the inventory data.
func Build(inv *inventory.Inventory) (string, error) {
	var builder strings.Builder
	seenVersions := make(map[string]bool)

	// Iterate over the bins and their versions
	for _, bin := range inv.Bins {
		for _, version := range bin.Versions {
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
