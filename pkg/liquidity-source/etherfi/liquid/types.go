package liquid

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type StaticExtra struct {
	LiquidRefer common.Address `json:"liquidRefer"`
	Teller      common.Address `json:"teller"`
}

type Extra struct {
	IsTellerPaused bool       `json:"tellerIsPaused"`
	AssetData      []Asset    `json:"assetData"`
	RateInQuote    []*big.Int `json:"rateInQuote"`
}

type Asset struct {
	AllowDeposits  bool   `json:"allowDeposits"`
	AllowWithdraws bool   `json:"allowWithdraws"`
	SharePremium   uint16 `json:"sharePremium"`
}

type MetaInfo struct {
	LiquidRefer     common.Address `json:"liquidRefer"`
	Teller          common.Address `json:"teller"`
	ApprovalAddress common.Address `json:"approvalAddress"`
}
