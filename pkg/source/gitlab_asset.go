package source

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/clients/gitlab"
	"github.com/ekristen/distillery/pkg/common"
)

type GitLabAsset struct {
	*asset.Asset

	GitLab *GitLab
	Link   *gitlab.Links
}

func (a *GitLabAsset) ID() string {
	return fmt.Sprintf("%s-%d", a.GetType(), a.Link.ID)
}

func (a *GitLabAsset) Path() string {
	return filepath.Join("gitlab", a.GitLab.GetOwner(), a.GitLab.GetRepo(), a.GitLab.Version)
}

func (a *GitLabAsset) Download(ctx context.Context) error { //nolint:dupl,nolintlint
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	downloadsDir := filepath.Join(cacheDir, common.NAME, "downloads")
	filename := filepath.Base(a.Link.URL)

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

	logrus.Debugf("downloading asset: %s", a.Link.URL)

	req, err := http.NewRequestWithContext(context.TODO(), "GET", a.Link.URL, http.NoBody)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", common.NAME, common.AppVersion))

	if a.GitLab.Client.GetToken() != "" {
		req.Header.Set("PRIVATE-TOKEN", a.GitLab.Client.GetToken())
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	hasher := sha256.New()
	tmpFile, err := os.Create(assetFile)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	multiWriter := io.MultiWriter(tmpFile, hasher)

	f, err := os.Create(assetFile)
	if err != nil {
		return err
	}

	// Write the asset's content to the temporary file
	_, err = io.Copy(multiWriter, resp.Body)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}

	logrus.Tracef("hash: %x", hasher.Sum(nil))

	_ = os.WriteFile(assetFileHash, []byte(fmt.Sprintf("%x", hasher.Sum(nil))), 0600)
	a.Hash = string(hasher.Sum(nil))

	logrus.Tracef("Downloaded asset to: %s", tmpFile.Name())

	return nil
}
