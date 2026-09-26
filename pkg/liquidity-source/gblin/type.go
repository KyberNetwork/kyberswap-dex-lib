package gblin

import (
	"github.com/holiman/uint256"
)

// StaticExtra holds values that only change through a timelocked governance call.
type StaticExtra struct {
	Lens string `json:"lens"`
	// SequencerFeed is the vault's sequencer uptime feed (a sentinel in front of Chainlink's). Empty when the
	// vault has none, in which case the sequencer check is off, as on-chain.
	SequencerFeed string `json:"sequencerFeed,omitempty"`
}

// Extra is the state a mint is priced from, read at one block.
//
// GBLIN.buyGBLINWithWeth prices a mint as
//
//	protocolFee  = amount * protocolFeeBps / 10_000
//	stabilityFee = amount * stabilityFeeBps / 10_000
//	supply'      = supply + supply * managementFeeBps * (now - lastAccrual) / (10_000 * 365 days)
//	out          = (amount - protocolFee - stabilityFee) * (supply' + 1e6) / (navEth + 1)
//	feeShares    = protocolFee * (supply' + 1e6) / (navEth + 1)
//
// with navEth = totalEthValue(0) + the vault's native ETH, which the mint wraps before pricing. The management
// fee is accrued to Timestamp, the block the state was read at: accruing to an earlier time under-states
// supply' and therefore under-quotes, so the vault never pays out less than quoted for that reason.
type Extra struct {
	Supply           *uint256.Int `json:"supply,omitempty"`
	NavEth           *uint256.Int `json:"navEth,omitempty"`
	LastAccrual      uint64       `json:"lastAccrual"`
	ManagementFeeBps uint64       `json:"managementFeeBps"`
	ProtocolFeeBps   uint64       `json:"protocolFeeBps"`
	StabilityFeeBps  uint64       `json:"stabilityFeeBps"`
	MinDeposit       *uint256.Int `json:"minDeposit,omitempty"`
	// NavReliable is false while a price feed is stale, a basket token does not answer or an auction fill is
	// open; the vault refuses mints then.
	NavReliable bool `json:"navReliable"`
	// SequencerUp is false while the sequencer is reported down or less than an hour after it came back up.
	SequencerUp bool   `json:"sequencerUp"`
	Timestamp   uint64 `json:"timestamp"`
}

// SwapInfo carries what a mint adds to the vault's supply, so UpdateBalance does not recompute it.
type SwapInfo struct {
	AccruedShares *uint256.Int `json:"-"`
	SharesOut     *uint256.Int `json:"-"`
	FeeShares     *uint256.Int `json:"-"`
}

// PoolMeta is passed to the executor-side encoder. The only entry point is
// buyGBLINWithWeth(amount, minOut, receiver) on the vault, which pulls WETH from the caller.
type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}
