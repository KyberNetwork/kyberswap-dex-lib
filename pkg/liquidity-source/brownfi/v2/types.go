package brownfiv2

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type GetReservesResult struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}

type Extra struct {
	Fee             uint64          `json:"f,omitempty"`
	Lambda          uint64          `json:"l,omitempty"`
	Kappa           *uint256.Int    `json:"k,omitempty"`
	OPrices         [2]*uint256.Int `json:"p,omitempty"`
	PriceUpdateData []byte          `json:"u,omitempty"`
}

type StaticExtra struct {
	PriceFeedIds [2]string `json:"f"`
}

type SwapInfo struct {
	PriceUpdateData []byte `json:"u"`
}

type PoolMeta struct {
	pool.ApprovalInfo
	Fee uint64 `json:"fee"`
}

type PythUpdateData struct {
	Binary struct {
		Data []string `json:"data"`
	} `json:"binary"`
	Parsed []struct {
		Price struct {
			Price string `json:"price"`
			Expo  int    `json:"expo"`
		} `json:"price"`
	} `json:"parsed"`
}
