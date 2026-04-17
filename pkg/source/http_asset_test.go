package source_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/osconfig"
	"github.com/ekristen/distillery/pkg/provider"
	"github.com/ekristen/distillery/pkg/source"
)

func newKubernetesHTTPAsset(t *testing.T, cfg *config.Config, serverURL, version string) *source.HTTPAsset {
	t.Helper()

	k := &source.Kubernetes{
		GitHub: source.GitHub{
			Provider: provider.Provider{
				Options:  &provider.Options{Config: cfg},
				OSConfig: osconfig.New("linux", "amd64"),
			},
			Owner:   source.KubernetesSource,
			Repo:    source.KubernetesSource,
			Version: version,
		},
		AppName: "kubectl",
	}

	return &source.HTTPAsset{
		Asset:  asset.New("kubectl-"+version+"-linux-amd64", "kubectl", "linux", "amd64", version),
		Source: k,
		URL:    serverURL + "/kubectl",
	}
}

func tempConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.New("")
	assert.NoError(t, err)
	cfg.CachePath = t.TempDir()
	return cfg
}

func TestHTTPAsset_Download_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("binary content"))
	}))
	defer srv.Close()

	a := newKubernetesHTTPAsset(t, tempConfig(t), srv.URL, "1.35.3")

	err := a.Download(context.Background())
	assert.NoError(t, err)
	// Binary and sha256 sidecar should both be written
	assert.FileExists(t, a.DownloadPath)
	assert.FileExists(t, a.DownloadPath+".sha256")
	// Download path must be version-scoped, not the flat downloads root
	assert.Contains(t, a.DownloadPath, "1.35.3")
}

func TestHTTPAsset_Download_CacheHitSkipsRequest(t *testing.T) {
	requestMade := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
		_, _ = w.Write([]byte("binary content"))
	}))
	defer srv.Close()

	a := newKubernetesHTTPAsset(t, tempConfig(t), srv.URL, "1.35.3")

	// Pre-create the sha256 sidecar to simulate a prior download
	hashFile := filepath.Join(a.Source.GetDownloadsDir(), "kubectl.sha256")
	assert.NoError(t, os.MkdirAll(filepath.Dir(hashFile), 0755))
	assert.NoError(t, os.WriteFile(hashFile, []byte("deadbeef"), 0600))

	err := a.Download(context.Background())
	assert.NoError(t, err)
	assert.False(t, requestMade, "expected no HTTP request when file is already downloaded")
}

func TestHTTPAsset_Download_VersionsDownloadSeparately(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte("binary content"))
	}))
	defer srv.Close()

	// Both assets share the same cache root to exercise version isolation
	cfg := tempConfig(t)
	assert.NoError(t, newKubernetesHTTPAsset(t, cfg, srv.URL, "1.33.7").Download(context.Background()))
	assert.NoError(t, newKubernetesHTTPAsset(t, cfg, srv.URL, "1.35.3").Download(context.Background()))
	assert.Equal(t, 2, requests, "each version should trigger a separate download")
}
