package forgejo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rs/zerolog/log"

	"github.com/ekristen/distillery/pkg/common"
)

const baseURL = "https://codeberg.org/api/v1"

func NewClient(client *http.Client) *Client {
	return &Client{
		client:  client,
		baseURL: baseURL,
	}
}

type Client struct {
	client  *http.Client
	baseURL string
	token   string
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

func (c *Client) GetToken() string {
	return c.token
}

func (c *Client) GetClient() *http.Client {
	return c.client
}

func (c *Client) doRequest(ctx context.Context, reqURL string) (*http.Response, error) {
	log.Trace().Msgf("GET %s", reqURL)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", common.NAME, common.AppVersion))
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, reqURL)
	}

	return resp, nil
}

func (c *Client) ListReleases(ctx context.Context, owner, repo string) ([]*Release, error) {
	var all []*Release

	const pageSize = 50
	for page := 1; ; page++ {
		reqURL := fmt.Sprintf("%s/repos/%s/%s/releases?limit=%d&page=%d",
			c.baseURL, url.PathEscape(owner), url.PathEscape(repo), pageSize, page)

		resp, err := c.doRequest(ctx, reqURL)
		if err != nil {
			return nil, err
		}

		var releases []*Release
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			_ = resp.Body.Close()
			return nil, err
		}
		_ = resp.Body.Close()

		all = append(all, releases...)

		if len(releases) < pageSize {
			break
		}
	}

	return all, nil
}

func (c *Client) GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	reqURL := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.baseURL, url.PathEscape(owner), url.PathEscape(repo))

	resp, err := c.doRequest(ctx, reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *Client) GetRelease(ctx context.Context, owner, repo, tag string) (*Release, error) {
	reqURL := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", c.baseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(tag))

	resp, err := c.doRequest(ctx, reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}
