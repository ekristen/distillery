package inventory

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/ekristen/distillery/pkg/config"
)

type Inventory struct {
	config *config.Config
	Bins   map[string]*Bin

	latestPaths map[string]string
}

func (i *Inventory) AddVersion(path, target string) error {
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

	source := strings.TrimPrefix(strings.TrimPrefix(target, i.config.GetOptPath()), "/")
	baseSourceParts := strings.SplitAfterN(source, "/", 4)
	baseSource := strings.TrimSuffix(strings.Join(baseSourceParts[:3], ""), "/")

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

		target, err := os.Readlink(filepath.Join(basePath, path))
		if err != nil {
			logrus.WithError(err).Warn("failed to read symlink")
		}

		if err := inv.AddVersion(path, target); err != nil {
			logrus.WithError(err).Warn("failed to add version")
		}

		return nil
	})

	inv.setLatestVersion()

	return inv
}