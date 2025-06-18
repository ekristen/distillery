package inventory

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/ekristen/distillery/pkg/config"
)

type Inventory struct {
	config *config.Config
	Bins   map[string]*Bin

	latestPaths map[string]string
}

func (i *Inventory) GetBinsSortedKeys() []string {
	sortedKeys := make([]string, 0, len(i.Bins))
	for key := range i.Bins {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)
	return sortedKeys
}

// SetConfig - set the configuration for the inventory - primarily used for tests only
func (i *Inventory) SetConfig(cfg *config.Config) {
	i.config = cfg
}

func (i *Inventory) AddVersion(path, target string) error {
	path = filepath.ToSlash(path)
	target = filepath.ToSlash(target)

	binName := filepath.Base(path)
	version := "latest"
	latest := true

	vParts := strings.Split(binName, "@")
	if len(vParts) == 2 {
		binName = vParts[0]
		version = vParts[1]
		latest = false
	} else {
		binName = vParts[0]
	}

	if i.Bins == nil {
		i.Bins = make(map[string]*Bin)
	}
	if i.latestPaths == nil {
		i.latestPaths = make(map[string]string)
	}

	// TODO: this doesn't work on windows :facepalm:

	relativeBin, _ := filepath.Rel(i.config.GetOptPath(), target)
	relativeBin = filepath.ToSlash(relativeBin)

	relativeParts := strings.Split(relativeBin, "/") // note: using / because we are standardizing via ToSlash
	baseSource := filepath.ToSlash(filepath.Join(relativeParts[:3]...))

	if i.Bins[baseSource] == nil {
		src := strings.TrimPrefix(strings.TrimPrefix(target, i.config.GetOptPath()), "/")
		parts := strings.Split(src, "/")

		i.Bins[baseSource] = &Bin{
			Name:     binName,
			Versions: make([]*Version, 0),
			Source:   parts[0],
			Owner:    parts[1],
			Repo:     parts[2],
		}
	}

	if latest {
		i.latestPaths[baseSource] = target
		return nil
	}

	if i.latestPaths[baseSource] == target {
		latest = true
	}

	i.Bins[baseSource].Versions = append(i.Bins[baseSource].Versions, &Version{
		Version: version,
		Path:    path,
		Latest:  latest,
		Target:  target,
	})

	return nil
}

func (i *Inventory) Count() int {
	return len(i.Bins)
}

func (i *Inventory) FullCount() int {
	count := 0
	for _, bin := range i.Bins {
		count += len(bin.Versions)
	}

	return count
}

func (i *Inventory) GetBinVersions(name string) *Bin {
	return i.Bins[name]
}

func (i *Inventory) GetBinVersion(name, version string) *Version {
	bin := i.GetBinVersions(name)
	if bin == nil {
		return nil
	}

	for _, v := range bin.Versions {
		if v.Latest && version == "latest" {
			return v
		} else if v.Version == version {
			return v
		}
	}

	return nil
}

func (i *Inventory) GetLatestVersion(name string) *Version {
	bin := i.GetBinVersions(name)
	if bin == nil {
		return nil
	}

	for _, v := range bin.Versions {
		if v.Latest {
			return v
		}
	}

	return nil
}

func (i *Inventory) setLatestVersion() {
	for baseSource, bin := range i.Bins {
		latestPath, exists := i.latestPaths[baseSource]
		if !exists {
			continue
		}

		for _, version := range bin.Versions {
			if version.Target == latestPath {
				version.Latest = true
			}
		}
	}
}

func New(fileSystem fs.FS, basePath, binPath string, cfg *config.Config) *Inventory {
	inv := &Inventory{
		config: cfg,
	}

	// scan the ~/.distillery/bin directory
	// for all the bins and versions
	// and return a new Inventory
	_ = fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		fileInfo, err := d.Info()
		if err != nil {
			return err
		}

		// if a symlink it's a version ...
		if fileInfo.Mode()&os.ModeSymlink != os.ModeSymlink {
			return nil
		}

		target, err := os.Readlink(filepath.ToSlash(filepath.Join(basePath, path)))
		if err != nil {
			log.Warn().Err(err).Msg("failed to read symlink")
		}

		path = filepath.ToSlash(path)
		target = filepath.ToSlash(target)

		log.Trace().Str("path", path).Str("target", target).Msg("adding version")

		if err := inv.AddVersion(path, target); err != nil {
			log.Warn().Err(err).Msg("failed to add version")
		}

		return nil
	})

	inv.setLatestVersion()

	return inv
}
