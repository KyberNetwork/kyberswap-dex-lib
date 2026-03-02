package launchpadv2

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type StaticExtra struct {
	Bonding string `json:"bA"`
	Router  string `json:"r"`
}

type Extra struct {
	Trading                    bool     `json:"t"`
	LaunchExecuted             bool     `json:"lE"`
	GradThreshold              *big.Int `json:"gT"`
	KLast                      *big.Int `json:"kL"`
	BuyTax                     *big.Int `json:"bTx"`
	SellTax                    *big.Int `json:"sTx"`
	ReserveA                   *big.Int `json:"rA"`
	ReserveB                   *big.Int `json:"rB"`
	AntiSniperBuyTaxStartValue *big.Int `json:"aV"`
	TaxStartTime               *big.Int `json:"tST"`
	StartTime                  *big.Int `json:"sT"`
	Graduated                  bool     `json:"g,omitempty"`
}

type SwapInfo struct {
	Bonding     string `json:"bondingAddress"`
	isBuy       bool
	newReserveA *uint256.Int
	newReserveB *uint256.Int
	newBalanceA *uint256.Int
	newBalanceB *uint256.Int
}

type PoolMeta struct {
	BlockNumber     uint64 `json:"bN"`
	ApprovalAddress string `json:"approvalAddress"`
}

type BondingTokenInfoData struct {
	Token        common.Address
	Name         string
	InternalName string
	Ticker       string
	Supply       *big.Int
	Price        *big.Int
	MarketCap    *big.Int
	Liquidity    *big.Int
	Volume       *big.Int
	Volume24H    *big.Int
	PrevPrice    *big.Int
	LastUpdated  *big.Int
}

type BondingTokenInfo struct {
	Creator          common.Address
	Token            common.Address
	Pair             common.Address
	AgentToken       common.Address
	Data             BondingTokenInfoData
	Description      string
	Image            string
	Twitter          string
	Telegram         string
	Youtube          string
	Website          string
	Trading          bool
	TradingOnUniswap bool
	ApplicationId    *big.Int
	InitialPurchase  *big.Int
	VirtualId        *big.Int
	LaunchExecuted   bool
}
