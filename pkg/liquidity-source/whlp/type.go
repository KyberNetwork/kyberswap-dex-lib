package whlp

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type StaticExtra struct {
	Accountant common.Address `json:"accountant"`
	Depositor  common.Address `json:"depositor"`
	QuoteAsset common.Address `json:"quoteAsset"`
}

type Extra struct {
	RateInQuote        *big.Int `json:"rateInQuote"`
	IsAccountantPaused bool     `json:"isAccountantPaused"`
}

type PoolMeta struct {
	Depositor     common.Address `json:"depositor"`
	Accountant    common.Address `json:"accountant"`
	CommunityCode string         `json:"communityCode"`
}
