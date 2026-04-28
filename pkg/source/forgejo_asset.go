package source

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/clients/forgejo"
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
	return asset.DownloadHTTP(ctx, a.Asset, a.ReleaseAsset.BrowserDownloadURL,
		a.Forgejo.Options.Config.GetDownloadsPath(),
		filepath.Base(a.ReleaseAsset.BrowserDownloadURL),
		&a.Forgejo.Logger,
		func(req *http.Request) {
			if a.Forgejo.Client.GetToken() != "" {
				req.Header.Set("Authorization", fmt.Sprintf("token %s", a.Forgejo.Client.GetToken()))
			}
		})
}
