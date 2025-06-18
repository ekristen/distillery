package source

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/clients/homebrew"
)

type HomebrewAsset struct {
	*asset.Asset

	Homebrew    *Homebrew
	FileVariant *homebrew.FileVariant
}

func (a *HomebrewAsset) ID() string {
	return fmt.Sprintf("%s-%s", a.GetType(), a.FileVariant.Sha256[:9])
}

func (a *HomebrewAsset) Path() string {
	return filepath.Join("homebrew", a.Homebrew.GetRepo(), a.Homebrew.Version)
}

type GHCRAuth struct {
	Token string `json:"token"`
}

func (g *GHCRAuth) Bearer() string {
	return fmt.Sprintf("Bearer %s", g.Token)
}

func (a *HomebrewAsset) getAuthToken() (*GHCRAuth, error) {
	// https://ghcr.io/token",service="ghcr.io",scope="repository:homebrew/core/ffmpeg:pull"
	req, err := http.NewRequestWithContext(context.TODO(), "GET", "https://ghcr.io/token", http.NoBody)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("service", "ghcr.io")
	q.Add("scope", fmt.Sprintf("repository:homebrew/core/%s:%s", a.Homebrew.GetRepo(), "pull"))
	req.URL.RawQuery = q.Encode()

	log.Trace().Msgf("request: %s", req.URL.String())

	var t *GHCRAuth

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, err
	}

	return t, nil
}

func (a *HomebrewAsset) Download(ctx context.Context) error {
	downloadsDir := a.Homebrew.Options.Config.GetDownloadsPath()
	filename := filepath.Base(a.Name + ".tar.gz")

	assetFile := filepath.Join(downloadsDir, filename)
	a.DownloadPath = assetFile
	a.Extension = filepath.Ext(a.DownloadPath)

	assetFileHash := assetFile + ".sha256"
	stats, err := os.Stat(assetFileHash)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if stats != nil {
		log.Debug().Msg("file already downloaded")
		return nil
	}

	token, err := a.getAuthToken()
	if err != nil {
		return err
	}

	// TODO: lookup manifest to determine how the file is stored ...
	req, err := http.NewRequestWithContext(context.TODO(), "GET", a.FileVariant.URL, http.NoBody)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", token.Bearer())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

	log.Trace().Msgf("hash: %x", hasher.Sum(nil))

	_ = os.WriteFile(assetFileHash, []byte(string(hasher.Sum(nil))), 0600)
	a.Hash = string(hasher.Sum(nil))

	log.Trace().Msgf("Downloaded asset to: %s", tmpFile.Name())

	return nil
}
