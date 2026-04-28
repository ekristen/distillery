package source

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"

	"github.com/ekristen/distillery/pkg/asset"
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
	logger := a.Source.GetOptions().Logger
	return asset.DownloadHTTP(ctx, a.Asset, a.URL,
		a.Source.GetDownloadsDir(),
		filepath.Base(a.URL),
		&logger, nil)
}
