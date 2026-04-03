package woofiv2

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type (
	Gas struct {
		Swap int64
	}

	// DecimalInfo
	// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L58
	DecimalInfo struct {
		priceDec *uint256.Int // 10**(price_decimal)
		quoteDec *uint256.Int // 10**(quote_decimal)
		baseDec  *uint256.Int // 10**(base_decimal)
	}

	woofiV2SwapInfo struct {
		newPrice      *uint256.Int
		newBase1Price *uint256.Int
		newBase2Price *uint256.Int
	}

	Extra struct {
		QuoteToken string               `json:"quoteToken"`
		TokenInfos map[string]TokenInfo `json:"tokenInfos"`
		Wooracle   Wooracle             `json:"wooracle"`
		Cloracle   map[string]Cloracle  `json:"cloracle"`
		IsPaused   bool                 `json:"isPaused"`
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
