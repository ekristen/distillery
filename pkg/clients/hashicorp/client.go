package hashicorp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ekristen/distillery/pkg/common"
)

// Client is a client for interacting with the HashiCorp Releases API
type Client struct {
	client *http.Client
}

// NewClient creates a new client for interacting with the HashiCorp Releases API
func NewClient(client *http.Client) *Client {
	if client == nil {
		client = &http.Client{}
	}

	return &Client{
		client: client,
	}
}

// ListProducts returns a list of products available from the HashiCorp Releases API
func (c *Client) ListProducts(ctx context.Context) (Products, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, "https://api.releases.hashicorp.com/v1/products", http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", common.NAME, common.AppVersion))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data Products
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}

// ListReleasesOptions are options for listing releases
type ListReleasesOptions struct {
	PreReleases  bool
	LicenseClass string
}

// ListReleases returns a list of releases for a product from the HashiCorp Releases API
func (c *Client) ListReleases(ctx context.Context, product string, opts *ListReleasesOptions) ([]*Release, error) {
	if opts == nil {
		opts = &ListReleasesOptions{
			LicenseClass: "oss",
		}
	}

	var licenseClass string
	if opts.LicenseClass != "all" {
		licenseClass = fmt.Sprintf("license_class=%s", opts.LicenseClass)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("https://api.releases.hashicorp.com/v1/releases/%s?%s", product, licenseClass), http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", common.NAME, common.AppVersion))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []*Release
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if !opts.PreReleases {
		for i, release := range data {
			if release.IsPrerelease {
				if i < len(data)-1 {
					data = append(data[:i], data[i+1:]...)
				} else {
					data = data[:i]
				}
			}
		}
	}

	return data, nil
}

// GetVersion returns a specific release for a product from the HashiCorp Releases API
func (c *Client) GetVersion(ctx context.Context, product, version string) (*Release, error) {
	licenseClass := "license_class=oss"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("https://api.releases.hashicorp.com/v1/releases/%s/%s?%s",
			product, version, licenseClass), http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", common.NAME, common.AppVersion))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data *Release
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}
