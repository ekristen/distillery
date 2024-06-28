package source

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/sirupsen/logrus"

	"github.com/ekristen/distillery/pkg/common"
)

var ignoreFileExtensions = []string{
	".txt",
	".sbom",
}

var executableMimetypes = []string{
	"application/x-mach-binary",
	"application/x-executable",
	"application/vnd.microsoft.portable-executable",
}

type ISource interface {
	GetSource() string
	GetOwner() string
	GetRepo() string
	GetApp() string
	GetID() string
	Run(context.Context, string, string) error
}

type Source struct {
	Options *Options

	File string
}

func (s *Source) GetOS() string {
	return s.Options.OS
}
func (s *Source) GetArch() string {
	return s.Options.Arch
}

func (s *Source) ExtractInstall(repo, binOs, binArch, version string) error { //nolint:gocyclo
	bins := true
	bin := ""

	hasSuffix := false
	var files []string
	if strings.HasSuffix(s.File, ".tar.gz") {
		hasSuffix = true
		raf, err := os.OpenFile(s.File, os.O_RDONLY, 0755)
		if err != nil {
			return err
		}

		tmpdir, err := os.MkdirTemp("", common.NAME)
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpdir)

		files, err = Untar(tmpdir, raf)
		if err != nil {
			return err
		}
	} else {
		files = []string{s.File}
	}

	for _, file := range files {
		m, err := mimetype.DetectFile(file)
		if err != nil {
			return err
		}

		logrus.Debugf("filename: %s, mimetype: %s", file, m.String())

		if slices.Contains(ignoreFileExtensions, m.Extension()) {
			continue
		}

		found := false
		if bins {
			// TODO: fuzzy matching?
			// cp file to $HOME/.distillery/bin
			if slices.Contains(executableMimetypes, m.String()) {
				found = true

				logrus.Debugf("found executable: %s, %s, %s", file, m.String(), m.Extension())

				dstFilename := filepath.Base(file)
				if !hasSuffix {
					dstFilename = repo
				}

				destBinaryName := fmt.Sprintf("%s-%s-%s-%s", repo, version, binOs, binArch)
				destBinFilename := filepath.Join(s.Options.BinDir, destBinaryName)
				simpleBinFilename := filepath.Join(s.Options.BinDir, dstFilename)

				if err := s.CopyFile(file, destBinFilename); err != nil {
					return err
				}

				// create symlink
				// TODO: check if symlink exists
				// TODO: allow override
				if runtime.GOOS == binOs && runtime.GOARCH == binArch {
					_ = os.Remove(simpleBinFilename)
					_ = os.Symlink(destBinFilename, simpleBinFilename)
				}
			}
		} else {
			if bin == m.String() {
				found = true
				// cp file to $HOME/.distillery/bin
				// TODO: implement
			}
		}

		if !found {
			return fmt.Errorf("the request binary was not found in the release")
		}
	}

	return nil
}

func (s *Source) CopyFile(srcFile, dstFile string) error {
	// Open the source file for reading
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(dstFile, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy the contents of the source file to the destination file
	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	return nil
}

type Options struct {
	OS           string
	Arch         string
	HomeDir      string
	CacheDir     string
	BinDir       string
	MetadataDir  string
	DownloadsDir string
}

func New(source string, opts *Options) ISource {
	version := "latest"
	versionParts := strings.Split(source, "@")
	if len(versionParts) > 1 {
		source = versionParts[0]
		version = versionParts[1]
	}

	parts := strings.Split(source, "/")
	if len(parts) == 2 {
		// could be github or homebrew
		if parts[0] == "homebrew" {
			return &Homebrew{
				Source:  Source{Options: opts},
				Formula: parts[1],
				Version: version,
			}
		}

		return &GitHub{
			Source:  Source{Options: opts},
			Owner:   parts[0],
			Repo:    parts[1],
			Version: version,
		}
	} else if len(parts) >= 3 {
		if strings.HasPrefix(parts[0], "github") {
			return &GitHub{
				Source:  Source{Options: opts},
				Owner:   parts[1],
				Repo:    parts[2],
				Version: version,
			}
		} else if strings.HasPrefix(parts[0], "gitlab") {
			return &GitLab{
				Source:  Source{Options: opts},
				Owner:   parts[1],
				Repo:    parts[2],
				Version: version,
			}
		}

		return nil
	}

	return nil
}
