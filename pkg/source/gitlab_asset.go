package source

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/clients/gitlab"
)

type GitLabAsset struct {
	*asset.Asset

	GitLab *GitLab
	Link   *gitlab.Links
}

func (a *GitLabAsset) ID() string {
	return fmt.Sprintf("%s-%d", a.GetType(), a.Link.ID)
}

func (a *GitLabAsset) Path() string {
	return filepath.Join("gitlab", a.GitLab.GetOwner(), a.GitLab.GetRepo(), a.GitLab.Version)
}

func (a *GitLabAsset) Download(ctx context.Context) error {
	return asset.DownloadHTTP(ctx, a.Asset, a.Link.URL,
		a.GitLab.Options.Config.GetDownloadsPath(),
		filepath.Base(a.Link.URL),
		&a.GitLab.Logger,
		func(req *http.Request) {
			if a.GitLab.Client.GetToken() != "" {
				req.Header.Set("PRIVATE-TOKEN", a.GitLab.Client.GetToken())
			}
		})
}
