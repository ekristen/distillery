package source

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-github/v80/github"
	"github.com/sirupsen/logrus"

	"github.com/ekristen/distillery/pkg/asset"
)

type GitHubAsset struct {
	*asset.Asset

	GitHub       *GitHub
	ReleaseAsset *github.ReleaseAsset
}

func (a *GitHubAsset) ID() string {
	return fmt.Sprintf("%s-%d", a.GetType(), a.ReleaseAsset.GetID())
}

func (a *GitHubAsset) Path() string {
	return filepath.Join("github", a.GitHub.GetOwner(), a.GitHub.GetRepo(), a.GitHub.Version)
}

func (a *GitHubAsset) Download(ctx context.Context) error {
	rc, url, err := a.GitHub.client.Repositories.DownloadReleaseAsset(
		ctx, a.GitHub.GetOwner(), a.GitHub.GetRepo(), a.ReleaseAsset.GetID(), http.DefaultClient)
	if err != nil {
		return err
	}
	defer rc.Close()

	if url != "" {
		logrus.Tracef("url: %s", url)
	}

	downloadsDir := a.GitHub.Options.Config.GetDownloadsPath()

	filename := a.ID()

	assetFile := filepath.Join(downloadsDir, filename)
	a.DownloadPath = assetFile
	a.Extension = filepath.Ext(a.DownloadPath)

	assetFileHash := fmt.Sprintf("%s.sha256", assetFile)

	stats, err := os.Stat(assetFileHash)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if stats != nil {
		logrus.Debugf("file already downloaded: %s", assetFile)
		return nil
	}

	// TODO: verify hash, add overwrite for force.

	hasher := sha256.New()

	// Create a temporary file
	tmpFile, err := os.Create(assetFile)
	if err != nil {
		return err
	}
	defer func(tmpFile *os.File) {
		_ = tmpFile.Close()
	}(tmpFile)

	multiWriter := io.MultiWriter(tmpFile, hasher)

	// Write the asset's content to the temporary file
	_, err = io.Copy(multiWriter, rc)
	if err != nil {
		return err
	}

	logrus.Tracef("hash: %x", hasher.Sum(nil))

	_ = os.WriteFile(fmt.Sprintf("%s.sha256", assetFile), []byte(fmt.Sprintf("%x", hasher.Sum(nil))), 0600)
	a.Hash = string(hasher.Sum(nil))

	logrus.Tracef("Downloaded asset to: %s", tmpFile.Name())
	logrus.Tracef("Release asset name: %s", a.ReleaseAsset.GetName())

	return nil
}
