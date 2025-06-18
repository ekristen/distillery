package homebrew

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/ekristen/distillery/pkg/common"
)

func NewClient(client *http.Client) *Client {
	if client == nil {
		client = http.DefaultClient
	}

	return &Client{
		client: client,
	}
}

type Client struct {
	client *http.Client
}

func (h *Client) GetFormula(ctx context.Context, formula string) (*Formula, error) {
	url := fmt.Sprintf("https://formulae.brew.sh/api/formula/%s.json", formula)

	log.Debug().Msgf("fetching formula: %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", common.NAME, common.AppVersion))

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var f *Formula
	if err := json.NewDecoder(resp.Body).Decode(&f); err != nil {
		return nil, err
	}

	return f, nil
}
