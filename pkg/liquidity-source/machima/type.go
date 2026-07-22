package machima

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

// Extra is the mutable pool state refreshed by PoolTracker on every cycle.
//
// The UniV3 half is the embedded uniswapv3.Extra verbatim rather than a re-declaration, because
// the delegated UniV3 tracker reads this same Extra back (FetchPoolTicks unmarshals it into a
// uniswapv3.Extra). Re-declaring the fields with uint256/int256 broke exactly that: uint256
// marshals to a *quoted* decimal string and math/big refuses to unmarshal one. Embedding keeps the
// shared half byte-identical in both directions.
type Extra struct {
	uniswapv3.Extra
	ProtocolState
}

// ProtocolState is the half of Extra that UniV3 knows nothing about.
//
// It is a struct rather than loose fields so the tracker can carry it across the UniV3 calls that
// rewrite Extra in one assignment — and so a field added here is carried automatically instead of
// being silently dropped. Dropping one is not a decode error: it just leaves hasTax false and the
// pool quotes with no tax.
type ProtocolState struct {
	// Per-token trading tax, from ClankNow.getTokenTax(token).
	BuyTaxBps  uint16 `json:"buyTaxBps"`
	SellTaxBps uint16 `json:"sellTaxBps"`
	HasTax     bool   `json:"hasTax"`

	// PoolDeploymentTime is the unix second the pool was deployed, from
	// MachimaToken.poolDeploymentTime(). Swaps revert until it + AntiSniperWindowSeconds.
	PoolDeploymentTime uint64 `json:"poolDeploymentTime"`

	// XmaSellSqrtPriceLimit is the launch-tick price floor the Machima swap adapter passes to the
	// pool as sqrtPriceLimitX96 when XMA is the token being sold. Nil means no floor.
	XmaSellSqrtPriceLimit *big.Int `json:"xmaSellSqrtPriceLimit,omitempty"`
}

// StaticExtra holds immutable pool metadata written once at discovery time, by either
// PoolsListUpdater (subgraph backfill) or PoolFactory (PoolCreated logs).
type StaticExtra struct {
	// Token is the launched token of this pool, i.e. the side that is not the counter asset. The
	// tracker reads the tax config for it. Swap direction is not derived from it — that uses the
	// global counter-asset set below, mirroring the router's _classifyPair.
	Token string `json:"token"`

	// RouterAddress is the MachimaAggregatorRouter the executor calls; it is the only field the
	// encoding layer needs.
	RouterAddress string `json:"routerAddress"`

	// Global counter-asset set, mirroring the router's _isCounterAsset mapping.
	WETH string `json:"weth"`
	USDC string `json:"usdc"`
	XMA  string `json:"xma"`
}

// SwapInfo carries the underlying V3 state transition plus the amounts that actually moved through
// the V3 pool. Tax is charged outside the pool, so pool reserves must be updated with the pre/post
// tax amounts rather than the user-facing ones.
type SwapInfo struct {
	V3 uniswapv3.SwapInfo `json:"v3"`

	// PoolAmountIn is the input reaching the pool: amountIn minus buy tax on a buy, amountIn on a sell.
	PoolAmountIn *big.Int `json:"poolAmountIn"`
	// PoolAmountOut is the output leaving the pool, before sell tax is deducted.
	PoolAmountOut *big.Int `json:"poolAmountOut"`
}

// TaxConfig mirrors IClankNow.TaxConfig for ABI decoding. Field order must match Solidity.
//
// Only BuyTaxBps/SellTaxBps/HasTax affect the swap output. The protocolTaxBps* fields split that
// same trading tax between the protocol and trading tax handlers rather than stacking on top of it.
//
// Verified on Base against MachimaAggregatorQuoter.quote() for XMA, whose config is
// buy/sell = 100 bps with protocolTaxBpsWeth/Usdc/Xma = 3400/3400/1500: the quoter reports
// taxBps = 100 and taxAmount = exactly 1% of amountIn. If the protocol bps were an extra deduction
// the effective rate would have been far higher.
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

// PoolMeta is consumed by aggregator-encoding's PackMachima. The executor calldata is a single word
// — the router address — since the contract derives the deadline from block.timestamp itself.
//
// ApprovalAddress is the same router and is not redundant: the Machima router *pulls* tokenIn from
// the executor with transferFrom, so the executor must have approved it or the swap reverts.
// executeMachima only approves when the SHOULD_APPROVE_MAX flag is set, and router-service decides
// that by reading pool.ApprovalInfo out of this struct — not from the simulator's
// GetApprovalAddress. Dropping this field silently breaks every Machima swap.
type PoolMeta struct {
	Router          string `json:"router"`
	ApprovalAddress string `json:"approvalAddress"`
}

// Metadata is the pool-list checkpoint. The field name is not free: the ticks-based bootstrap
// persists its own poolMetadata over whatever the lister returns, and feeds that same shape back
// into GetNewPools. Using any other name means the cursor silently never loads, so every bootstrap
// re-scans from zero and the failure-retry rewind (bootstrap rewinds to the earliest pool that
// failed) is ignored.
type Metadata struct {
	LastCreatedAtTimestamp *big.Int `json:"lastCreatedAtTimestamp"`
}

// SubgraphPool matches the Machima subgraph Pool entity. Token symbol and decimals are
// deliberately not requested — a downstream job fills those in after listing.
type SubgraphPool struct {
	ID          string        `json:"id"`
	Token0      SubgraphToken `json:"token0"`
	Token1      SubgraphToken `json:"token1"`
	TradedToken string        `json:"tradedToken"`
	CreatedAt   string        `json:"createdAt"`
}

type SubgraphToken struct {
	Address string `json:"id"`
}
