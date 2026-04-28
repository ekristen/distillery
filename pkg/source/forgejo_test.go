package source_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ekristen/distillery/pkg/clients/forgejo"
	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/osconfig"
	"github.com/ekristen/distillery/pkg/provider"
	"github.com/ekristen/distillery/pkg/source"
)

type roundTripperFunc func(req *http.Request) *http.Response

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func newMockHTTPClient(body string, statusCode int) *http.Client {
	return &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}
		}),
	}
}

func newForgejoWithClient(httpClient *http.Client, version string, settings map[string]interface{}) *source.Forgejo {
	cfg, _ := config.New("")

	if settings == nil {
		settings = map[string]interface{}{
			"forgejo-token":        "",
			"include-pre-releases": false,
		}
	}

	s := &source.Forgejo{
		Provider: provider.Provider{
			Options: &provider.Options{
				Config:   cfg,
				Settings: settings,
			},
			OSConfig: osconfig.New("linux", "amd64"),
		},
		Owner:   "owner",
		Repo:    "repo",
		Version: version,
	}
	s.Client = forgejo.NewClient(httpClient)
	return s
}

func TestForgejo_GetSource_DefaultsToForgejoSource(t *testing.T) {
	s := &source.Forgejo{}
	assert.Equal(t, source.ForgejoSource, s.GetSource())
}

func TestForgejo_GetSource_ReturnsSourceNameWhenSet(t *testing.T) {
	s := &source.Forgejo{SourceName: "codeberg"}
	assert.Equal(t, "codeberg", s.GetSource())
}

func TestForgejo_GetSource_ReturnsCustomProviderName(t *testing.T) {
	s := &source.Forgejo{SourceName: "mygitea"}
	assert.Equal(t, "mygitea", s.GetSource())
}

func TestForgejo_GetOwner(t *testing.T) {
	s := &source.Forgejo{Owner: "myowner"}
	assert.Equal(t, "myowner", s.GetOwner())
}

func TestForgejo_GetRepo(t *testing.T) {
	s := &source.Forgejo{Repo: "myrepo"}
	assert.Equal(t, "myrepo", s.GetRepo())
}

func TestForgejo_GetApp(t *testing.T) {
	s := &source.Forgejo{Owner: "myowner", Repo: "myrepo"}
	assert.Equal(t, "myowner/myrepo", s.GetApp())
}

func TestForgejo_GetID_WithDefaultSource(t *testing.T) {
	s := &source.Forgejo{Owner: "myowner", Repo: "myrepo"}
	assert.Equal(t, "forgejo/myowner/myrepo", s.GetID())
}

func TestForgejo_GetID_WithSourceName(t *testing.T) {
	s := &source.Forgejo{Owner: "myowner", Repo: "myrepo", SourceName: "codeberg"}
	assert.Equal(t, "codeberg/myowner/myrepo", s.GetID())
}

func TestForgejo_GetVersion_NilRelease(t *testing.T) {
	s := &source.Forgejo{}
	assert.Equal(t, common.Unknown, s.GetVersion())
}

func TestForgejo_GetVersion_StripsPrefixV(t *testing.T) {
	s := &source.Forgejo{
		Release: &forgejo.Release{TagName: "v1.2.3"},
	}
	assert.Equal(t, "1.2.3", s.GetVersion())
}

func TestForgejo_GetVersion_NoPrefix(t *testing.T) {
	s := &source.Forgejo{
		Release: &forgejo.Release{TagName: "1.2.3"},
	}
	assert.Equal(t, "1.2.3", s.GetVersion())
}

func TestForgejo_FindRelease_Latest_UsesGetLatestRelease(t *testing.T) {
	body := `{
		"id": 1,
		"tag_name": "v2.0.0",
		"name": "Release 2.0.0",
		"draft": false,
		"prerelease": false,
		"created_at": "2024-01-01T00:00:00Z",
		"published_at": "2024-01-01T00:00:00Z",
		"assets": []
	}`

	callCount := 0
	mockTransport := roundTripperFunc(func(req *http.Request) *http.Response {
		callCount++
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}
	})

	s := newForgejoWithClient(&http.Client{Transport: mockTransport}, provider.VersionLatest, nil)

	err := s.FindRelease(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s.Release)
	assert.Equal(t, "2.0.0", s.GetVersion())
	// Should have used GetLatestRelease (one call), not fallen back to list
	assert.Equal(t, 1, callCount)
}

// --- FindRelease: latest + GetLatestRelease 404 falls back to list but still fails ---
//
// When GetLatestRelease returns a 404, findRelease swallows the error and falls
// through to findReleaseInList. However, s.Version is still "latest" at that
// point and findReleaseInList has no code path to match the literal string
// "latest" against a tag name (the pre-release branch requires includePreReleases=true,
// and the exact-match branch compares the tag against "latest"). The result is
// "release not found". This test documents that behavior.
func TestForgejo_FindRelease_Latest_404_ResultsInReleaseNotFound(t *testing.T) {
	listBody := `[{
		"id": 1,
		"tag_name": "v1.5.0",
		"name": "Release 1.5.0",
		"draft": false,
		"prerelease": false,
		"created_at": "2024-01-01T00:00:00Z",
		"published_at": "2024-01-01T00:00:00Z",
		"assets": []
	}]`

	callCount := 0
	mockTransport := roundTripperFunc(func(req *http.Request) *http.Response {
		callCount++
		if strings.Contains(req.URL.Path, "/releases/latest") {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(listBody)),
			Header:     make(http.Header),
		}
	})

	s := newForgejoWithClient(&http.Client{Transport: mockTransport}, provider.VersionLatest, nil)

	err := s.FindRelease(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "release not found")
	// Both the latest endpoint and the list endpoint are called
	assert.Equal(t, 2, callCount)
}

func TestForgejo_FindRelease_Latest_PropagatesNon404Error(t *testing.T) {
	mockTransport := roundTripperFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("server error")),
			Header:     make(http.Header),
		}
	})

	s := newForgejoWithClient(&http.Client{Transport: mockTransport}, provider.VersionLatest, nil)

	err := s.FindRelease(context.Background())
	assert.Error(t, err)
}

func TestForgejo_FindRelease_SpecificVersion_MatchedFromList(t *testing.T) {
	listBody := `[
		{"id": 2, "tag_name": "v2.0.0", "draft": false, "prerelease": false,
		 "created_at": "2024-01-01T00:00:00Z", "published_at": "2024-01-01T00:00:00Z", "assets": []},
		{"id": 1, "tag_name": "v1.0.0", "draft": false, "prerelease": false,
		 "created_at": "2024-01-01T00:00:00Z", "published_at": "2024-01-01T00:00:00Z", "assets": []}
	]`

	s := newForgejoWithClient(newMockHTTPClient(listBody, http.StatusOK), "1.0.0", nil)

	err := s.FindRelease(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s.Release)
	assert.Equal(t, "v1.0.0", s.Release.TagName)
}

func TestForgejo_FindRelease_SpecificVersion_VPrefixStripped(t *testing.T) {
	listBody := `[
		{"id": 1, "tag_name": "v3.1.0", "draft": false, "prerelease": false,
		 "created_at": "2024-01-01T00:00:00Z", "published_at": "2024-01-01T00:00:00Z", "assets": []}
	]`

	// Pass version with a v-prefix to ensure both sides are stripped for comparison
	s := newForgejoWithClient(newMockHTTPClient(listBody, http.StatusOK), "v3.1.0", nil)

	err := s.FindRelease(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s.Release)
	assert.Equal(t, "v3.1.0", s.Release.TagName)
}

func TestForgejo_FindRelease_VersionNotFound(t *testing.T) {
	listBody := `[
		{"id": 1, "tag_name": "v1.0.0", "draft": false, "prerelease": false,
		 "created_at": "2024-01-01T00:00:00Z", "published_at": "2024-01-01T00:00:00Z", "assets": []}
	]`

	s := newForgejoWithClient(newMockHTTPClient(listBody, http.StatusOK), "9.9.9", nil)

	err := s.FindRelease(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "release not found")
}

func TestForgejo_FindRelease_Latest_IncludePreReleases_ReturnsFirstPreRelease(t *testing.T) {
	listBody := `[
		{"id": 2, "tag_name": "v2.0.0-beta.1", "draft": false, "prerelease": true,
		 "created_at": "2024-06-01T00:00:00Z", "published_at": "2024-06-01T00:00:00Z", "assets": []},
		{"id": 1, "tag_name": "v1.0.0", "draft": false, "prerelease": false,
		 "created_at": "2024-01-01T00:00:00Z", "published_at": "2024-01-01T00:00:00Z", "assets": []}
	]`

	callCount := 0
	mockTransport := roundTripperFunc(func(req *http.Request) *http.Response {
		callCount++
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(listBody)),
			Header:     make(http.Header),
		}
	})

	settings := map[string]interface{}{
		"forgejo-token":        "",
		"include-pre-releases": true,
	}
	s := newForgejoWithClient(&http.Client{Transport: mockTransport}, provider.VersionLatest, settings)

	err := s.FindRelease(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s.Release)
	assert.True(t, s.Release.Prerelease)
	assert.Equal(t, "2.0.0-beta.1", s.GetVersion())
	// Must not have called GetLatestRelease — only the list endpoint
	assert.Equal(t, 1, callCount)
}

func TestForgejo_FindRelease_404_NoToken_NoError(t *testing.T) {
	// We can't easily assert the log warning, but we can verify the error is
	// propagated and the code does not panic when the token is empty.
	mockTransport := roundTripperFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("not found")),
			Header:     make(http.Header),
		}
	})

	settings := map[string]interface{}{
		"forgejo-token":        "",
		"include-pre-releases": false,
	}
	s := newForgejoWithClient(&http.Client{Transport: mockTransport}, "1.0.0", settings)

	err := s.FindRelease(context.Background())
	assert.Error(t, err)
}
