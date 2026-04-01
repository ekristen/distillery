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
	"github.com/ekristen/distillery/pkg/clients/forgejo"
	"github.com/ekristen/distillery/pkg/common"
)

type ForgejoAsset struct {
	*asset.Asset

	Forgejo      *Forgejo
	ReleaseAsset *forgejo.ReleaseAsset
}

func (a *ForgejoAsset) ID() string {
	return fmt.Sprintf("%s-%d", a.GetType(), a.ReleaseAsset.ID)
}

func (a *ForgejoAsset) Path() string {
	return filepath.Join(a.Forgejo.GetSource(), a.Forgejo.GetOwner(), a.Forgejo.GetRepo(), a.Forgejo.Version)
}

func (a *ForgejoAsset) Download(ctx context.Context) error {
	downloadsDir := a.Forgejo.Options.Config.GetDownloadsPath()
	filename := filepath.Base(a.ReleaseAsset.BrowserDownloadURL)

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

	logrus.Debugf("downloading asset: %s", a.ReleaseAsset.BrowserDownloadURL)

	req, err := http.NewRequestWithContext(ctx, "GET", a.ReleaseAsset.BrowserDownloadURL, http.NoBody)
	if err != nil {
		return err
	}

	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", common.NAME, common.AppVersion))
	if a.Forgejo.Client.GetToken() != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", a.Forgejo.Client.GetToken()))
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

	if _, err := io.Copy(multiWriter, resp.Body); err != nil {
		return err
	}

	logrus.Tracef("hash: %x", hasher.Sum(nil))

	_ = os.WriteFile(assetFileHash, []byte(fmt.Sprintf("%x", hasher.Sum(nil))), 0600)
	a.Hash = string(hasher.Sum(nil))

	logrus.Tracef("Downloaded asset to: %s", tmpFile.Name())

	return nil
}
