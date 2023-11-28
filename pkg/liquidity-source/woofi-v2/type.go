package woofiv2

import "github.com/holiman/uint256"

// DecimalInfo
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L58
type (
	DecimalInfo struct {
		priceDec *uint256.Int // 10**(price_decimal)
		quoteDec *uint256.Int // 10**(quote_decimal)
		baseDec  *uint256.Int // 10**(base_decimal)
	}

	Extra struct {
		QuoteToken string
		TokenInfos map[string]TokenInfo
		Wooracle   Wooracle
	}

	Wooracle struct {
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

	woofiV2SwapInfo struct {
		newPrice      *uint256.Int
		newBase1Price *uint256.Int
		newBase2Price *uint256.Int
	}

	Gas struct {
		Swap int64
	}
)
