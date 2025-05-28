package etherfiebtc

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type InitialPool struct {
	Teller     string             `json:"teller"`
	Accountant string             `json:"accountant"`
	Base       string             `json:"base"`
	Decimals   uint8              `json:"decimals"`
	Tokens     []entity.PoolToken `json:"tokens"`
}

type Gas struct {
	Deposit int64
}

type Extra struct {
	IsTellerPaused  bool                        `json:"isTellerPaused"`
	ShareLockPeriod uint64                      `json:"shareLockPeriod"`
	Assets          map[string]Asset            `json:"assets"`
	AccountantState AccountantState             `json:"accountantState"`
	RateProviders   map[string]RateProviderData `json:"rateProviders"`
}

type StaticExtra struct {
	Accountant string `json:"accountant"`
	Base       string `json:"base"`
	Decimals   uint8  `json:"decimals"`
}

type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress"`
}

type Asset struct {
	AllowDeposits  bool   `json:"allowDeposits"`
	AllowWithdraws bool   `json:"allowWithdraws"`
	SharePremium   uint16 `json:"sharePremium"`
}

type AccountantState struct {
	PayoutAddress                  common.Address `json:"-"`
	HighwaterMark                  *big.Int       `json:"-"`
	FeesOwedInBase                 *big.Int       `json:"-"`
	TotalSharesLastUpdate          *big.Int       `json:"-"`
	ExchangeRate                   *big.Int       `json:"exchangeRate"`
	AllowedExchangeRateChangeUpper uint16         `json:"-"`
	AllowedExchangeRateChangeLower uint16         `json:"-"`
	LastUpdateTimestamp            uint64         `json:"lastUpdateTimestamp"`
	IsPaused                       bool           `json:"isPaused"`
	MinimumUpdateDelayInSeconds    *big.Int       `json:"-"`
	ManagementFee                  uint16         `json:"-"`
	PerformanceFee                 uint16         `json:"-"`
}

type RateProviderData struct {
	IsPeggedToBase bool           `json:"isPeggedToBase"`
	RateProvider   common.Address `json:"rateProvider"`
}
