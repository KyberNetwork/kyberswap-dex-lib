package liquid

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type StaticExtra struct {
	LiquidRefer common.Address `json:"liquidRefer"`
	Teller      common.Address `json:"teller"`
	// SimpleTeller: see PoolInfo.
	SimpleTeller  bool           `json:"simpleTeller,omitempty"`
	Depositor     common.Address `json:"depositor,omitempty"`
	Accountant    common.Address `json:"accountant,omitempty"`
	CommunityCode string         `json:"communityCode,omitempty"`
}

type Config struct {
	DexID string `json:"dexID"`
	// ChainID selects the built-in pools; zero means Ethereum.
	ChainID valueobject.ChainID `json:"chainId"`
}

type Extra struct {
	IsTellerPaused  bool       `json:"tellerIsPaused"`
	ShareLockPeriod uint64     `json:"shareLockPeriod,omitempty"`
	AssetData       []Asset    `json:"assetData"`
	RateInQuote     []*big.Int `json:"rateInQuote"`
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
	Depositor       common.Address `json:"depositor"`
	Accountant      common.Address `json:"accountant"`
	CommunityCode   string         `json:"communityCode"`
	BlockNumber     uint64         `json:"blockNumber"`
}
