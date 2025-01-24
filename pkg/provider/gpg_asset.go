package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/ekristen/distillery/pkg/asset"
)

type GPGAsset struct {
	*asset.Asset

	KeyID   uint64
	Options *Options

	Source ISource
}

func (a *GPGAsset) ID() string {
	return fmt.Sprintf("%s-%d", a.GetType(), a.KeyID)
}

func (a *GPGAsset) Path() string {
	return filepath.Join("gpg", strconv.FormatUint(a.KeyID, 10))
}

func (a *GPGAsset) Download(ctx context.Context) error {
	var err error
	a.KeyID, err = a.MatchedAsset.GetGPGKeyID()
	if err != nil {
		logrus.WithError(err).Trace("unable to get GPG key")
		return err
	}

	downloadsDir := a.Options.Config.GetDownloadsPath()
	filename := strconv.FormatUint(a.KeyID, 10)

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

	logrus.Debugf("downloading asset: %d", a.KeyID)

	url := fmt.Sprintf("https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x%s", fmt.Sprintf("%X", a.KeyID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to download key: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download key: server returned status %s", resp.Status)
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
