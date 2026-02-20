package source

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/clients/hashicorp"
	"github.com/ekristen/distillery/pkg/common"
)

type HashicorpAsset struct {
	*asset.Asset

	Hashicorp *Hashicorp
	Build     *hashicorp.Build
	Release   *hashicorp.Release
}

func (a *HashicorpAsset) ID() string {
	urlHash := sha256.Sum256([]byte(a.Build.URL))
	urlHashShort := fmt.Sprintf("%x", urlHash)[:9]

	return fmt.Sprintf("%s-%s", a.GetType(), urlHashShort)
}

func (a *HashicorpAsset) Path() string {
	return filepath.Join("hashicorp", a.Hashicorp.GetRepo(), a.Hashicorp.Version)
}

func (a *HashicorpAsset) Download(ctx context.Context) error {
	logger := a.Hashicorp.Logger

	downloadsDir := a.Hashicorp.Options.Config.GetDownloadsPath()
	filename := filepath.Base(a.Build.URL)

	assetFile := filepath.Join(downloadsDir, filename)
	a.DownloadPath = assetFile
	a.Extension = filepath.Ext(a.DownloadPath)

	assetFileHash := assetFile + ".sha256"
	stats, err := os.Stat(assetFileHash)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if stats != nil {
		logger.Debug().Msg("file already downloaded")
		return nil
	}

	logger.Debug().Msgf("downloading asset: %s", a.Build.URL)

	req, err := http.NewRequestWithContext(ctx, "GET", a.Build.URL, http.NoBody)
	if err != nil {
		return err
	}
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

	// Write the asset's content to the file and hasher simultaneously
	_, err = io.Copy(multiWriter, resp.Body)
	if err != nil {
		return err
	}

	logger.Trace().Msgf("hash: %x", hasher.Sum(nil))

	_ = os.WriteFile(assetFileHash, []byte(fmt.Sprintf("%x", hasher.Sum(nil))), 0600)
	a.Hash = string(hasher.Sum(nil))

	logger.Trace().Msgf("Downloaded asset to: %s", tmpFile.Name())

	return nil
}
