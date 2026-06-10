package whlp

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type StaticExtra struct {
	Accountant    common.Address `json:"accountant"`
	Depositor     common.Address `json:"depositor"`
	CommunityCode string         `json:"communityCode"`
}

type Extra struct {
	RateInQuote *big.Int `json:"rateInQuote"`
}

type MetaInfo struct {
	Depositor       common.Address `json:"depositor"`
	Accountant      common.Address `json:"accountant"`
	CommunityCode   string         `json:"communityCode"`
	ApprovalAddress common.Address `json:"approvalAddress"`
}
