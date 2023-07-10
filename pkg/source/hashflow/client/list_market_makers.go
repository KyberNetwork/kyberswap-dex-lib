package client

import (
	"net/url"
)

type (
	listMarketMakersQueryParams struct {
		Source    string
		NetworkId string
	}

	listMarketMakersResult struct {
		MarketMakers []string `json:"marketMakers"`
	}
)

func (p listMarketMakersQueryParams) toUrlValues() url.Values {
	return url.Values{
		"source":    []string{p.Source},
		"networkId": []string{p.NetworkId},
	}
}
