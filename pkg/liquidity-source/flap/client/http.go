package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	// graduatingHotBoardPath is the only board selector that returns curve-stage (not yet migrated
	// to DEX) tokens: listed=false, progress < 100 for every item. The plain /v3/board endpoint (and
	// every other selector: trending, worldcup, cat, tag=<name>) only ever returns already-graduated
	// (progress=100, listed=true) tokens sorted by volume, since fresh curve tokens rank far below
	// them by that metric — verified empirically, not documented.
	graduatingHotBoardPath = "/v3/board/graduatinghot"

	apiKeyHeader = "trust-wallet-by-pass"

	defaultTimeout = 10 * time.Second
)

var ErrBoardRequestFailed = errors.New("flap: board request failed")

type Client struct {
	client *resty.Client
}

// NewClient builds a client for the flap.sh board API. baseURL is per-chain, e.g.
// https://bnb.taxed.fun. apiKey is sent as the trust-wallet-by-pass header and must only ever come
// from server-side config.
func NewClient(baseURL, apiKey string) *Client {
	c := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(defaultTimeout).
		SetHeader(apiKeyHeader, apiKey)

	return &Client{client: c}
}

// GetGraduatingHotBoard fetches one page of curve-stage (not yet graduated) tokens, optionally
// continuing from a previous cursor. The API returns at most 20 items per page; NextCursor is empty on
// the last page.
func (c *Client) GetGraduatingHotBoard(ctx context.Context, cursor string) (BoardResponse, error) {
	req := c.client.R().SetContext(ctx)
	if cursor != "" {
		req.SetQueryParam("cursor", cursor)
	}

	var result BoardResponse
	resp, err := req.SetResult(&result).Get(graduatingHotBoardPath)
	if err != nil {
		return BoardResponse{}, err
	}
	if resp.IsError() || resp.StatusCode() != http.StatusOK {
		return BoardResponse{}, ErrBoardRequestFailed
	}

	return result, nil
}
