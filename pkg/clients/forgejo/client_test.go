package forgejo_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ekristen/distillery/pkg/clients/forgejo"
)

func loadTestData(t *testing.T, filename string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/" + filename)
	assert.NoError(t, err)
	return string(data)
}

type roundTripperFunc func(req *http.Request) *http.Response

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func newMockClient(responseBody string, statusCode int) *http.Client {
	return &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader(responseBody)),
				Header:     make(http.Header),
			}
		}),
	}
}

func TestListReleases(t *testing.T) {
	mockResponse := loadTestData(t, "list-releases.json")
	client := forgejo.NewClient(newMockClient(mockResponse, http.StatusOK))
	client.SetToken("test-token")

	releases, err := client.ListReleases(context.Background(), "owner", "repo")
	assert.NoError(t, err)
	assert.NotNil(t, releases)
	assert.Equal(t, 2, len(releases))
	assert.Equal(t, "v1.1.0", releases[0].TagName)
	assert.Equal(t, 1, len(releases[0].Assets))
	assert.Equal(t, "myapp-linux-amd64.tar.gz", releases[0].Assets[0].Name)
	assert.Contains(t, releases[0].Assets[0].BrowserDownloadURL, "v1.1.0")
}

func TestGetLatestRelease(t *testing.T) {
	mockResponse := loadTestData(t, "latest-release.json")
	client := forgejo.NewClient(newMockClient(mockResponse, http.StatusOK))
	client.SetToken("test-token")

	release, err := client.GetLatestRelease(context.Background(), "owner", "repo")
	assert.NoError(t, err)
	assert.NotNil(t, release)
	assert.Equal(t, "v1.1.0", release.TagName)
	assert.Equal(t, 1, len(release.Assets))
}

func TestGetRelease(t *testing.T) {
	mockResponse := loadTestData(t, "get-release.json")
	client := forgejo.NewClient(newMockClient(mockResponse, http.StatusOK))
	client.SetToken("test-token")

	release, err := client.GetRelease(context.Background(), "owner", "repo", "v1.0.0")
	assert.NoError(t, err)
	assert.NotNil(t, release)
	assert.Equal(t, "v1.0.0", release.TagName)
	assert.False(t, release.Prerelease)
}

func TestSetBaseURL(t *testing.T) {
	const customBase = "https://git.example.com/api/v1"
	var capturedURL string

	mockTransport := roundTripperFunc(func(req *http.Request) *http.Response {
		capturedURL = req.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("[]")),
			Header:     make(http.Header),
		}
	})

	client := forgejo.NewClient(&http.Client{Transport: mockTransport})
	client.SetBaseURL(customBase)

	_, err := client.ListReleases(context.Background(), "owner", "repo")
	assert.NoError(t, err)
	assert.Contains(t, capturedURL, "git.example.com")
}

func TestTokenHeader(t *testing.T) {
	t.Run("with token", func(t *testing.T) {
		var capturedHeader string

		mockTransport := roundTripperFunc(func(req *http.Request) *http.Response {
			capturedHeader = req.Header.Get("Authorization")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("[]")),
				Header:     make(http.Header),
			}
		})

		client := forgejo.NewClient(&http.Client{Transport: mockTransport})
		client.SetToken("test-token")

		_, err := client.ListReleases(context.Background(), "owner", "repo")
		assert.NoError(t, err)
		assert.Equal(t, "token test-token", capturedHeader)
	})

	t.Run("without token", func(t *testing.T) {
		var capturedHeader string

		mockTransport := roundTripperFunc(func(req *http.Request) *http.Response {
			capturedHeader = req.Header.Get("Authorization")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("[]")),
				Header:     make(http.Header),
			}
		})

		client := forgejo.NewClient(&http.Client{Transport: mockTransport})

		_, err := client.ListReleases(context.Background(), "owner", "repo")
		assert.NoError(t, err)
		assert.Equal(t, "", capturedHeader)
	})
}

func TestForgejoClientErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		testFunc   func() error
		shouldFail bool
	}{
		{
			name: "ListReleases_InvalidURL",
			testFunc: func() error {
				client := forgejo.NewClient(newMockClient("", http.StatusOK))
				client.SetBaseURL("://invalid-url-%%")
				_, err := client.ListReleases(context.Background(), "owner", "repo")
				return err
			},
			shouldFail: true,
		},
		{
			name: "ListReleases_HTTPError",
			testFunc: func() error {
				client := forgejo.NewClient(newMockClient("", http.StatusInternalServerError))
				_, err := client.ListReleases(context.Background(), "owner", "repo")
				return err
			},
			shouldFail: true,
		},
		{
			name: "ListReleases_InvalidJSON",
			testFunc: func() error {
				client := forgejo.NewClient(newMockClient("not json", http.StatusOK))
				_, err := client.ListReleases(context.Background(), "owner", "repo")
				return err
			},
			shouldFail: true,
		},
		{
			name: "GetLatestRelease_InvalidURL",
			testFunc: func() error {
				client := forgejo.NewClient(newMockClient("", http.StatusOK))
				client.SetBaseURL("://invalid-url-%%")
				_, err := client.GetLatestRelease(context.Background(), "owner", "repo")
				return err
			},
			shouldFail: true,
		},
		{
			name: "GetLatestRelease_HTTPError",
			testFunc: func() error {
				client := forgejo.NewClient(newMockClient("", http.StatusInternalServerError))
				_, err := client.GetLatestRelease(context.Background(), "owner", "repo")
				return err
			},
			shouldFail: true,
		},
		{
			name: "GetLatestRelease_InvalidJSON",
			testFunc: func() error {
				client := forgejo.NewClient(newMockClient("not json", http.StatusOK))
				_, err := client.GetLatestRelease(context.Background(), "owner", "repo")
				return err
			},
			shouldFail: true,
		},
		{
			name: "GetRelease_InvalidURL",
			testFunc: func() error {
				client := forgejo.NewClient(newMockClient("", http.StatusOK))
				client.SetBaseURL("://invalid-url-%%")
				_, err := client.GetRelease(context.Background(), "owner", "repo", "v1.0.0")
				return err
			},
			shouldFail: true,
		},
		{
			name: "GetRelease_HTTPError",
			testFunc: func() error {
				client := forgejo.NewClient(newMockClient("", http.StatusInternalServerError))
				_, err := client.GetRelease(context.Background(), "owner", "repo", "v1.0.0")
				return err
			},
			shouldFail: true,
		},
		{
			name: "GetRelease_InvalidJSON",
			testFunc: func() error {
				client := forgejo.NewClient(newMockClient("not json", http.StatusOK))
				_, err := client.GetRelease(context.Background(), "owner", "repo", "v1.0.0")
				return err
			},
			shouldFail: true,
		},
		{
			name: "GetRelease_NotFound",
			testFunc: func() error {
				client := forgejo.NewClient(newMockClient("not found", http.StatusNotFound))
				_, err := client.GetRelease(context.Background(), "owner", "repo", "v9.9.9")
				return err
			},
			shouldFail: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.testFunc()
			if tc.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestListReleasesPagination(t *testing.T) {
	// Build a response with exactly 50 items (the page size) so the client
	// makes a second request, then return an empty page to stop iteration.
	makeReleases := func(count int, startID int) string {
		var sb strings.Builder
		sb.WriteString("[")
		for i := 0; i < count; i++ {
			if i > 0 {
				sb.WriteString(",")
			}
			id := startID + i
			sb.WriteString(fmt.Sprintf(`{
				"id": %d,
				"tag_name": "v1.%d.0",
				"name": "Release 1.%d.0",
				"body": "",
				"draft": false,
				"prerelease": false,
				"created_at": "2024-01-01T00:00:00Z",
				"published_at": "2024-01-01T00:00:00Z",
				"assets": []
			}`, id, i, i))
		}
		sb.WriteString("]")
		return sb.String()
	}

	page1 := makeReleases(50, 1)
	page2 := makeReleases(2, 51)

	callCount := 0
	mockTransport := roundTripperFunc(func(req *http.Request) *http.Response {
		callCount++
		body := page2
		if callCount == 1 {
			body = page1
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}
	})

	client := forgejo.NewClient(&http.Client{Transport: mockTransport})
	releases, err := client.ListReleases(context.Background(), "owner", "repo")

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount, "expected two HTTP requests for two pages")
	assert.Equal(t, 52, len(releases))
}

func TestUserAgentHeader(t *testing.T) {
	var capturedUA string

	mockTransport := roundTripperFunc(func(req *http.Request) *http.Response {
		capturedUA = req.Header.Get("User-Agent")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("[]")),
			Header:     make(http.Header),
		}
	})

	client := forgejo.NewClient(&http.Client{Transport: mockTransport})
	_, err := client.ListReleases(context.Background(), "owner", "repo")

	assert.NoError(t, err)
	assert.NotEmpty(t, capturedUA)
	assert.Contains(t, capturedUA, "distillery")
}
