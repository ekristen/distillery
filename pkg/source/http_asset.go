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
	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/provider"
)

type HTTPAsset struct {
	*asset.Asset

	Source provider.ISource
	URL    string
}

func (a *HTTPAsset) ID() string {
	urlHash := sha256.Sum256([]byte(a.URL))
	urlHashShort := fmt.Sprintf("%x", urlHash)[:9]

	return fmt.Sprintf("%s-%s", a.GetType(), urlHashShort)
}

func (a *HTTPAsset) Path() string {
	return filepath.Join(a.Source.GetSource(), a.Source.GetApp(), a.Source.GetVersion())
}

func (a *HTTPAsset) Download(ctx context.Context) error {
	downloadsDir := a.Source.GetOptions().Config.GetDownloadsPath()
	filename := filepath.Base(a.URL)

	assetFile := filepath.Join(downloadsDir, filename)
	a.DownloadPath = assetFile
	a.Extension = filepath.Ext(a.DownloadPath)

	assetFileHash := assetFile + ".sha256"
	stats, err := os.Stat(assetFileHash)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if stats != nil {
		logrus.Debug("file already downloaded")
		return nil
	}

	logrus.Debugf("downloading asset: %s", a.URL)

	req, err := http.NewRequestWithContext(context.TODO(), "GET", a.URL, http.NoBody)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", common.NAME, common.AppVersion))

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
