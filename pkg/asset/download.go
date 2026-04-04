package asset

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/httpclient"
)

// DownloadHTTP downloads a file from a URL, caching by sha256 hash file.
// It sets DownloadPath, Extension, and Hash on the provided Asset.
// beforeRequest allows callers to add custom headers (auth tokens, etc).
func DownloadHTTP(ctx context.Context, a *Asset, url, downloadsDir, filename string,
	logger *zerolog.Logger, beforeRequest func(*http.Request)) error {
	assetFile := filepath.Join(downloadsDir, filename)
	a.DownloadPath = assetFile
	a.Extension = filepath.Ext(assetFile)

	assetFileHash := assetFile + ".sha256"
	stats, err := os.Stat(assetFileHash)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if stats != nil {
		logger.Debug().Msgf("file already downloaded: %s", assetFile)
		return nil
	}

	logger.Debug().Msgf("downloading asset: %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", common.NAME, common.AppVersion))

	if beforeRequest != nil {
		beforeRequest(req)
	}

	resp, err := httpclient.NewSafeClient().Do(req)
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

	if _, err = io.Copy(multiWriter, resp.Body); err != nil {
		return err
	}

	hexHash := fmt.Sprintf("%x", hasher.Sum(nil))
	logger.Trace().Msgf("hash: %s", hexHash)

	_ = os.WriteFile(assetFileHash, []byte(hexHash), 0600)
	a.Hash = hexHash

	logger.Trace().Msgf("Downloaded asset to: %s", tmpFile.Name())

	return nil
}
