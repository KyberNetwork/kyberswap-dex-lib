package woofiv2

import (
	"github.com/holiman/uint256"
)

type (
	Extra struct {
		QuoteToken string               `json:"quoteToken"`
		TokenInfos map[string]TokenInfo `json:"tokenInfos"`
		Wooracle   Wooracle             `json:"wooracle"`
	}

	Wooracle struct {
		Address  string           `json:"address"`
		States   map[string]State `json:"states"`
		Decimals map[string]uint8 `json:"decimals"`
	}

	TokenInfo struct {
		Reserve *uint256.Int `json:"reserve"`
		FeeRate uint16       `json:"feeRate"`
	}

	State struct {
		Price      *uint256.Int `json:"price"`
		Spread     uint64       `json:"spread"`
		Coeff      uint64       `json:"coeff"`
		WoFeasible bool         `json:"woFeasible"`
	}
)
