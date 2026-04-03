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
	logger := a.Options.Logger

	var err error
	a.KeyID, err = a.MatchedAsset.GetGPGKeyID()
	if err != nil {
		logger.Trace().Err(err).Msg("unable to get GPG key")
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
		logger.Debug().Msgf("file already downloaded: %s", assetFile)
		return nil
	}

	logger.Debug().Msgf("downloading GPG key: %d", a.KeyID)

	url := fmt.Sprintf("https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x%s", fmt.Sprintf("%X", a.KeyID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to download key: %w", err)
	}

	resp, err := http.DefaultClient.Do(req) //nolint:gosec // URL hardcoded to Ubuntu keyserver, only key ID varies
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

	_, err = io.Copy(multiWriter, resp.Body)
	if err != nil {
		return err
	}

	logger.Trace().Msgf("hash: %x", hasher.Sum(nil))

	_ = os.WriteFile(assetFileHash, []byte(fmt.Sprintf("%x", hasher.Sum(nil))), 0600)
	a.Hash = fmt.Sprintf("%x", hasher.Sum(nil))

	logger.Trace().Msgf("Downloaded GPG key to: %s", tmpFile.Name())

	return nil
}
