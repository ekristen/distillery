package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

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
	req, err := http.NewRequestWithContext(context.TODO(), "GET", "https://ghcr.io/token", http.NoBody)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("service", "ghcr.io")
	q.Add("scope", fmt.Sprintf("repository:homebrew/core/%s:%s", a.Homebrew.GetRepo(), "pull"))
	req.URL.RawQuery = q.Encode()

	a.Homebrew.Logger.Trace().Msgf("request: %s", req.URL.String())

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
	token, err := a.getAuthToken()
	if err != nil {
		return err
	}

	return asset.DownloadHTTP(ctx, a.Asset, a.FileVariant.URL,
		a.Homebrew.Options.Config.GetDownloadsPath(),
		a.Name+".tar.gz",
		&a.Homebrew.Logger,
		func(req *http.Request) {
			req.Header.Set("Authorization", token.Bearer())
		})
}
