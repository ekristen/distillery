package source

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/clients/hashicorp"
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
	return asset.DownloadHTTP(ctx, a.Asset, a.Build.URL,
		a.Hashicorp.Options.Config.GetDownloadsPath(),
		filepath.Base(a.Build.URL),
		&a.Hashicorp.Logger, nil)
}
