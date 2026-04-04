package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeClient_SameHostRedirectPreservesAuth(t *testing.T) {
	// Server that redirects from /start to /dest on the same host
	mux := http.NewServeMux()
	var destAuth, destPrivateToken string

	mux.HandleFunc("/dest", func(w http.ResponseWriter, r *http.Request) {
		destAuth = r.Header.Get("Authorization")
		destPrivateToken = r.Header.Get("PRIVATE-TOKEN")
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dest", http.StatusFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewSafeClient()
	req, err := http.NewRequestWithContext(context.Background(), "GET", srv.URL+"/start", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("PRIVATE-TOKEN", "gitlab-secret")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "Bearer secret-token", destAuth, "Authorization header should be preserved on same-host redirect")
	assert.Equal(t, "gitlab-secret", destPrivateToken, "PRIVATE-TOKEN header should be preserved on same-host redirect")
}

func TestSafeClient_CrossHostRedirectStripsAuth(t *testing.T) {
	// Destination server (different host) that captures headers
	var destAuth, destPrivateToken string
	dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		destAuth = r.Header.Get("Authorization")
		destPrivateToken = r.Header.Get("PRIVATE-TOKEN")
		w.WriteHeader(http.StatusOK)
	}))
	defer dest.Close()

	// Origin server that redirects to the destination
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, dest.URL+"/resource", http.StatusFound)
	}))
	defer origin.Close()

	client := NewSafeClient()
	req, err := http.NewRequestWithContext(context.Background(), "GET", origin.URL+"/start", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("PRIVATE-TOKEN", "gitlab-secret")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, destAuth, "Authorization header should be stripped on cross-host redirect")
	assert.Empty(t, destPrivateToken, "PRIVATE-TOKEN header should be stripped on cross-host redirect")
}

func TestSafeClient_TooManyRedirects(t *testing.T) {
	redirectCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		http.Redirect(w, r, fmt.Sprintf("/redir-%d", redirectCount), http.StatusFound)
	}))
	defer srv.Close()

	client := NewSafeClient()
	req, err := http.NewRequestWithContext(context.Background(), "GET", srv.URL+"/start", http.NoBody)
	require.NoError(t, err)

	resp, err := client.Do(req)
	// After 10 redirects, the client should stop following
	// http.ErrUseLastResponse causes the client to return the last response
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.LessOrEqual(t, redirectCount, 11) // initial + up to 10 redirects
}

func TestSafeClient_NoRedirect(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewSafeClient()
	req, err := http.NewRequestWithContext(context.Background(), "GET", srv.URL+"/resource", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer my-token")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "Bearer my-token", receivedAuth, "Authorization header should be sent when there is no redirect")
}
