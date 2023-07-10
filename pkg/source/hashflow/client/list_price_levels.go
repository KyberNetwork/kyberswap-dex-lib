package client

import (
	"net/url"
)

type (
	listPriceLevelsQueryParams struct {
		Source        string
		NetworkId     string
		MarketMarkers []string
	}

	listPriceLevelsResult struct {
		Status    string                                 `json:"status"`
		NetworkID int                                    `json:"networkId"`
		Levels    map[string][]listPriceLevelsResultPair `json:"levels"`
	}

	listPriceLevelsResultPair struct {
		Pair   listPriceLevelsResultPairDescription `json:"pair"`
		Levels []listPriceLevelResultPriceLevel     `json:"levels"`
	}

	listPriceLevelsResultPairDescription struct {
		BaseTokenName      string `json:"baseTokenName"`
		QuoteTokenName     string `json:"quoteTokenName"`
		BaseToken          string `json:"baseToken"`
		QuoteToken         string `json:"quoteToken"`
		BaseTokenDecimals  uint8  `json:"baseTokenDecimals"`
		QuoteTokenDecimals uint8  `json:"quoteTokenDecimals"`
	}

	listPriceLevelResultPriceLevel struct {
		Level string `json:"level"`
		Price string `json:"price"`
	}
)

func (p listPriceLevelsQueryParams) toUrlValues() url.Values {
	return url.Values{
		"source":       []string{p.Source},
		"networkId":    []string{p.NetworkId},
		"marketMakers": p.MarketMarkers,
	}
}
