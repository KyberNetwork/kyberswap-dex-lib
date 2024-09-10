package woofiv21

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type (
	Extra struct {
		QuoteToken string               `json:"quoteToken"`
		TokenInfos map[string]TokenInfo `json:"tokenInfos"`
		Wooracle   Wooracle             `json:"wooracle"`
		Cloracle   map[string]Cloracle  `json:"cloracle"`
	}

	Wooracle struct {
		Address       string           `json:"address"`
		States        map[string]State `json:"states"`
		Decimals      map[string]uint8 `json:"decimals"`
		Timestamp     int64            `json:"timestamp"`
		StaleDuration int64            `json:"staleDuration"`
		Bound         uint64           `json:"bound"`
	}

	Cloracle struct {
		OracleAddress common.Address `json:"oracleAddress"`
		Answer        *uint256.Int   `json:"answer"`
		UpdatedAt     *uint256.Int   `json:"updatedAt"`
		CloPreferred  bool           `json:"cloPreferred"`
	}

	TokenInfo struct {
		Reserve         *uint256.Int `json:"reserve"`
		FeeRate         uint16       `json:"feeRate"`
		MaxGamma        *uint256.Int `json:"maxGamma"`
		MaxNotionalSwap *uint256.Int `json:"maxNotionalSwap"`
	}

	State struct {
		Price      *uint256.Int `json:"price"`
		Spread     uint64       `json:"spread"`
		Coeff      uint64       `json:"coeff"`
		WoFeasible bool         `json:"woFeasible"`
	}
)
