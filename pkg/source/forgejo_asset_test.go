package source_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/clients/forgejo"
	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/osconfig"
	"github.com/ekristen/distillery/pkg/provider"
	"github.com/ekristen/distillery/pkg/source"
)

func newForgejoAsset(t *testing.T, token, serverURL string) *source.ForgejoAsset {
	t.Helper()

	cfg, err := config.New("")
	assert.NoError(t, err)
	cfg.CachePath = t.TempDir()

	forgejoSource := &source.Forgejo{
		Provider: provider.Provider{
			Options: &provider.Options{
				Config: cfg,
				Settings: map[string]interface{}{
					"forgejo-token":        token,
					"include-pre-releases": false,
				},
			},
			OSConfig: osconfig.New("linux", "amd64"),
		},
		Owner:   "owner",
		Repo:    "repo",
		Version: "1.0.0",
	}
	forgejoSource.Client = forgejo.NewClient(http.DefaultClient)
	if token != "" {
		forgejoSource.Client.SetToken(token)
	}

	filename := "myapp-linux-amd64.tar.gz"
	downloadURL := fmt.Sprintf("%s/%s", serverURL, filename)

	releaseAsset := &forgejo.ReleaseAsset{
		ID:                 2001,
		Name:               filename,
		BrowserDownloadURL: downloadURL,
	}

	return &source.ForgejoAsset{
		Asset:        asset.New(filename, "", "linux", "amd64", "1.0.0"),
		Forgejo:      forgejoSource,
		ReleaseAsset: releaseAsset,
	}
}

func TestForgejoAsset_ID(t *testing.T) {
	cfg, _ := config.New("")
	forgejoSource := &source.Forgejo{
		Provider: provider.Provider{
			Options: &provider.Options{
				Config: cfg,
				Settings: map[string]interface{}{
					"forgejo-token":        "",
					"include-pre-releases": false,
				},
			},
			OSConfig: osconfig.New("linux", "amd64"),
		},
		Owner:   "owner",
		Repo:    "repo",
		Version: "1.0.0",
	}
	forgejoSource.Client = forgejo.NewClient(http.DefaultClient)

	a := &source.ForgejoAsset{
		Asset:        asset.New("myapp-linux-amd64.tar.gz", "", "linux", "amd64", "1.0.0"),
		Forgejo:      forgejoSource,
		ReleaseAsset: &forgejo.ReleaseAsset{ID: 42},
	}

	assert.Equal(t, fmt.Sprintf("%s-%d", a.GetType(), 42), a.ID())
}

func TestForgejoAsset_Path(t *testing.T) {
	cfg, _ := config.New("")
	forgejoSource := &source.Forgejo{
		Provider: provider.Provider{
			Options: &provider.Options{
				Config: cfg,
				Settings: map[string]interface{}{
					"forgejo-token":        "",
					"include-pre-releases": false,
				},
			},
			OSConfig: osconfig.New("linux", "amd64"),
		},
		Owner:      "owner",
		Repo:       "repo",
		Version:    "1.0.0",
		SourceName: "codeberg",
	}
	forgejoSource.Client = forgejo.NewClient(http.DefaultClient)

	a := &source.ForgejoAsset{
		Asset:        asset.New("myapp-linux-amd64.tar.gz", "", "linux", "amd64", "1.0.0"),
		Forgejo:      forgejoSource,
		ReleaseAsset: &forgejo.ReleaseAsset{ID: 1},
	}

	assert.Equal(t, filepath.Join("codeberg", "owner", "repo", "1.0.0"), a.Path())
}

func TestForgejoAsset_Download_AlreadyDownloaded(t *testing.T) {
	requestMade := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("file content"))
	}))
	defer srv.Close()

	a := newForgejoAsset(t, "", srv.URL)

	// Pre-create the sha256 sidecar to simulate a prior download
	filename := filepath.Base(a.ReleaseAsset.BrowserDownloadURL)
	assetFile := filepath.Join(a.Forgejo.Options.Config.GetDownloadsPath(), filename)
	hashFile := assetFile + ".sha256"
	assert.NoError(t, os.MkdirAll(filepath.Dir(hashFile), 0755))
	assert.NoError(t, os.WriteFile(hashFile, []byte("deadbeef"), 0600))

	err := a.Download(context.Background())
	assert.NoError(t, err)
	assert.False(t, requestMade, "expected no HTTP request when file is already downloaded")
}

func TestForgejoAsset_Download_Success(t *testing.T) {
	content := []byte("binary content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	a := newForgejoAsset(t, "", srv.URL)
	assert.NoError(t, os.MkdirAll(a.Forgejo.Options.Config.GetDownloadsPath(), 0755))

	err := a.Download(context.Background())
	assert.NoError(t, err)

	// Asset file should exist
	assert.FileExists(t, a.DownloadPath)

	// SHA256 sidecar should exist
	assert.FileExists(t, a.DownloadPath+".sha256")
}

func TestForgejoAsset_Download_NonOKStatusReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	a := newForgejoAsset(t, "", srv.URL)
	assert.NoError(t, os.MkdirAll(a.Forgejo.Options.Config.GetDownloadsPath(), 0755))

	err := a.Download(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestForgejoAsset_Download_SetsAuthHeaderWithToken(t *testing.T) {
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()

	a := newForgejoAsset(t, "my-secret-token", srv.URL)
	assert.NoError(t, os.MkdirAll(a.Forgejo.Options.Config.GetDownloadsPath(), 0755))

	err := a.Download(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "token my-secret-token", capturedAuth)
}

func TestForgejoAsset_Download_NoAuthHeaderWithoutToken(t *testing.T) {
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()

	a := newForgejoAsset(t, "", srv.URL)
	assert.NoError(t, os.MkdirAll(a.Forgejo.Options.Config.GetDownloadsPath(), 0755))

	err := a.Download(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, capturedAuth)
}
