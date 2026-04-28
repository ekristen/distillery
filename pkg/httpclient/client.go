package httpclient

import (
	"net/http"
)

// authHeaders are headers that must be stripped when following a redirect
// to a different host, to prevent leaking credentials.
var authHeaders = []string{"Authorization", "PRIVATE-TOKEN"}

// NewSafeClient returns an http.Client that strips authentication headers
// when following redirects to a different host. This prevents credential
// leakage if a download URL redirects to an attacker-controlled server.
func NewSafeClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			if len(via) > 0 && req.URL.Host != via[0].URL.Host {
				for _, h := range authHeaders {
					req.Header.Del(h)
				}
			}
			return nil
		},
	}
}
