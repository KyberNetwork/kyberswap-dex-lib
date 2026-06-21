package machima

import (
	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// Extra is pool extra data stored alongside the standard entity.Pool
type Extra struct {
	SqrtPriceX96 *uint256.Int `json:"sqrtPriceX96"`
	Tick         *int         `json:"tick"`
	Liquidity    *uint256.Int `json:"liquidity"`
	Ticks        []TickData   `json:"ticks"`
	TickSpacing  int          `json:"tickSpacing"`
	// Machima-specific: per-token tax config
	BuyTaxBps  uint16 `json:"buyTaxBps"`
	SellTaxBps uint16 `json:"sellTaxBps"`
	HasTax     bool   `json:"hasTax"`
	// Counter asset for this pool (WETH/USDC/XMA address)
	CounterAsset string `json:"counterAsset"`
	// Token address (the launched token)
	Token string `json:"token"`
	// Pool deployment time (unix seconds) — read from MachimaToken.poolDeploymentTime()
	PoolDeploymentTime uint64 `json:"poolDeploymentTime"`
	// XMA sell price floor — applied when the token being sold is XMA
	XmaSellSqrtPriceLimit *uint256.Int `json:"xmaSellSqrtPriceLimit,omitempty"`
}

// StaticExtra holds immutable pool metadata set at discovery time
type StaticExtra struct {
	CounterAsset  string `json:"counterAsset"`
	Token         string `json:"token"`
	RouterAddress string `json:"routerAddress"`
	// Global counter-asset set — mirrors on-chain _isCounterAsset
	WETH string `json:"weth"`
	USDC string `json:"usdc"`
	XMA  string `json:"xma"`
}

type TickData struct {
	Index          int          `json:"index"`
	LiquidityGross *uint256.Int `json:"liquidityGross"`
	LiquidityNet   *int256.Int  `json:"liquidityNet"`
}

// TaxConfig mirrors IClankNow.TaxConfig struct for ABI decoding.
// Field order must match the Solidity struct exactly.
type TaxConfig struct {
	BuyTaxBps          uint16
	SellTaxBps         uint16
	TradingTaxHandler  common.Address
	ProtocolTaxHandler common.Address
	ProtocolTaxBpsWeth uint16
	ProtocolTaxBpsUsdc uint16
	ProtocolTaxBpsXma  uint16
	HasTax             bool
}

// PoolMeta is returned by GetMetaInfo for the on-chain adapter encoding
type PoolMeta struct {
	Router   string `json:"router"`
	Deadline uint64 `json:"deadline"`
}
