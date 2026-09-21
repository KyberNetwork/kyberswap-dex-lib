package everlongflamm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// StaticExtra is the listing's pinned identity of one pool: everything the tracker addresses its reads to and the
// simulator re-checks against the registries (hook_registry.go) on every quote (msgpack restores a simulator
// without NewPoolSimulator). Every field here is immutable on chain, or its change is a different wiring that has
// to be relisted once the registries admit it (a new implementation, a new hook set). What a curator can change on
// a live pool -- the Router venue set -- is in Extra instead, so a refresh can adopt it (Extra.Venues); pool-service
// is not assumed to overwrite a listed pool's StaticExtra.
type StaticExtra struct {
	ProfileVersion uint32              `json:"profileVersion"`
	ChainID        valueobject.ChainID `json:"chainId"`
	Factory        common.Address      `json:"factory"`
	Implementation common.Address      `json:"implementation"`
	Router         common.Address      `json:"router"`
	PriceFeed      common.Address      `json:"priceFeed"`
	Morpho         common.Address      `json:"morpho"`
	PoolAsset      common.Address      `json:"poolAsset"`
	LoanAsset      common.Address      `json:"loanAsset"`
	// Hooks is hooks() in field order: invariant, fee, recenter, controller, leverage, spread, loanSwap. The kind of
	// each is the registry's (hook_registry.go resolveHooks).
	Hooks [7]common.Address `json:"hooks"`
	// SequencerFeed and each token's aggregator and heartbeat are fixed by the PriceFeed (immutables and a config
	// with no setter).
	SequencerFeed common.Address    `json:"sequencerFeed"`
	Aggregators   [2]common.Address `json:"aggregators"` // pool asset, loan asset 0
	Heartbeats    [2]uint64         `json:"heartbeats"`
	ListedBlock   uint64            `json:"listedBlock"`
}

// StaticVenue is one Router venue: its financing account, Morpho market id and market params. The market params
// are immutable per id (Morpho creates a market once), the account and the id per venue index; the set itself is
// not, so it is published in Extra and re-read by every refresh.
type StaticVenue struct {
	Account  common.Address `json:"account"`
	MarketID common.Hash    `json:"marketId"`
	Oracle   common.Address `json:"oracle"`
	Irm      common.Address `json:"irm"`
	Lltv     uint256.Int    `json:"lltv"`
	// OracleFeeds are the price feeds Oracle is wired to, in MorphoChainlinkOracleV2 field order (baseFeed1,
	// baseFeed2, quoteFeed1, quoteFeed2; the unused ones are zero). They are immutables of the oracle, which is
	// itself immutable per market id, so they are read with the rest of the venue rather than every refresh -- but
	// each one is a Chainlink proxy, whose aggregator does move, so the tracker resolves them per refresh for the
	// dependency set (pool_tracker.go). An oracle that is not a MorphoChainlinkOracleV2 answers none and leaves
	// them zero, which costs the pool nothing but a dependency.
	OracleFeeds [4]common.Address `json:"oracleFeeds,omitempty"`
}

// Policy is the configured quoting policy the tracker stamps into Extra (config.go documents each margin).
type Policy struct {
	LeverRouting       bool   `json:"leverRouting,omitempty"`
	LeverMinEdgeBps    uint64 `json:"leverMinEdgeBps,omitempty"`
	PriceBandMarginBps uint64 `json:"priceBandMarginBps,omitempty"`
	PriceAgeMarginSec  uint64 `json:"priceAgeMarginSec,omitempty"`
	SpreadAgeMarginSec uint64 `json:"spreadAgeMarginSec,omitempty"`
	DebtDriftSec       uint64 `json:"debtDriftSec,omitempty"`
	MaxSnapshotAgeSec  uint64 `json:"maxSnapshotAgeSec,omitempty"`
	// QuoteDonatedVenues lifts the refusal of a venue position the Router does not fully manage (envelope).
	QuoteDonatedVenues bool  `json:"quoteDonatedVenues,omitempty"`
	GasSwapSell        int64 `json:"gasSwapSell,omitempty"`
	GasSwapSellCapEval int64 `json:"gasSwapSellCapEval,omitempty"`
	GasSwapSellPass    int64 `json:"gasSwapSellPass,omitempty"`
	GasSwapBuy         int64 `json:"gasSwapBuy,omitempty"`
	GasLeverUp         int64 `json:"gasLeverUp,omitempty"`
	GasLeverDown       int64 `json:"gasLeverDown,omitempty"`
}

func (p Policy) withDefaults() Policy {
	for _, g := range []struct {
		v   *int64
		def int64
	}{{&p.GasSwapSell, defaultGasSwapSell}, {&p.GasSwapSellCapEval, defaultGasSwapSellCapEval},
		{&p.GasSwapSellPass, defaultGasSwapSellPass}, {&p.GasSwapBuy, defaultGasSwapBuy},
		{&p.GasLeverUp, defaultGasLeverUp}, {&p.GasLeverDown, defaultGasLeverDown}} {
		if *g.v <= 0 {
			*g.v = g.def
		}
	}
	return p
}

// Extra is one refresh: every word the state is built from (state_reads.go), all read at the entity's
// BlockNumber, and the verdict of the refresh. The simulator quotes only an Attested snapshot without drift:
// the tracker re-ran the pool's own previewSwap / previewLever / fundingCeiling at that block and the port
// reproduced every answer and every revert.
type Extra struct {
	Reads *flammReads `json:"reads,omitempty"`
	// Venues is the Router venue set the refresh read at BlockNumber, with the Morpho market params behind each
	// id: MMRouter.venue(pool, i) for i < venueCount(pool). The curator may add one to a live pool
	// (MMRouter.sol addVenue), which is why it is refreshed here rather than pinned at listing; every entry is
	// re-checked against the registry before a quote (validVenues).
	Venues []StaticVenue `json:"venues,omitempty"`
	// OracleAhead is what each venue's Morpho market oracle answers at the far end of the snapshot window
	// (MorphoBlueAccount.oraclePrice(id) at the refresh block with the block timestamp moved to BlockNumber's
	// timestamp + Policy.MaxSnapshotAgeSec; empty when the policy declares no window). It is the one input of the
	// state that time alone moves: the BTC/USD feed behind the market oracle is a Chainlink SVR DualAggregator,
	// which withholds each primary round for a fixed delay, so price() answers a different round as the clock
	// passes it with no transaction in between. The simulator re-runs every fill at these answers and refuses it
	// unless it settles identically (pool_simulator.go oracleShifted).
	OracleAhead []OracleAnswer `json:"oracleAhead,omitempty"`
	// Dependencies are the non-pool addresses whose logs should also trigger a refresh
	// (pool.IPoolTrackerWithDependencies): the aggregators the PriceFeed's two proxies and the sequencer proxy
	// resolve to, the aggregators behind each venue's market oracle feeds, the hooks whose kind emits events of its
	// own that move a quote (the swap hook and the leverage spread hook) and the Router. DependenciesStored is
	// pool-service's own flag, which the refresh clears whenever the resolved set changes (a Chainlink phase rotation
	// emits nothing at the proxy).
	Dependencies       []common.Address `json:"dependencies,omitempty"`
	DependenciesStored bool             `json:"dependenciesStored,omitempty"`
	Attested           bool             `json:"attested"`
	Probes             int              `json:"probes,omitempty"`
	AttestFailure      string           `json:"attestFailure,omitempty"`
	ProfileDrift       string           `json:"profileDrift,omitempty"`
	// ScheduledChangeAt is the earliest timestamp at which a scheduled change to what the port mirrors can execute
	// (0: none scheduled): the factory's pending implementation, which anyone may execute for every pool at once
	// (FLAMMFactory.sol:149), the pool's pending hook set (FLAMMOpsLib.sol:258), a pending venue admission or a
	// pending loan asset (tracker_reads.go scheduledChangeAt).
	ScheduledChangeAt uint64 `json:"scheduledChangeAt,omitempty"`
	Policy            Policy `json:"policy"`
}

// OracleAnswer is one MorphoBlueAccount.oraclePrice(id) answer: whether the oracle answered non-zero inside the
// account's gas cap, and the price it answered (1e36-scaled).
type OracleAnswer struct {
	Ok    bool        `json:"ok"`
	Price uint256.Int `json:"price"`
}

// PoolMeta is what the executor needs: the pool is the approval target (it pulls exactly amountInUsed) and the
// snapshot block the quote was priced at. The venue is per fill (SwapInfo.Venue, adapter data word 1).
type PoolMeta struct {
	ApprovalAddress string `json:"approvalAddress"`
	BlockNumber     uint64 `json:"blockNumber"`
}

// SwapInfo carries one quoted fill to UpdateBalance: the venue the adapter must be told (data word 1), the
// amounts, and the complete post-state of the settlement CalcAmountOut ran, which UpdateBalance adopts verbatim.
// AmountInUsed is what the pool pulls: a swap's used input, a lever-up's poolAsset, a lever-down's payNative.
type SwapInfo struct {
	Venue        uint8       `json:"venue"`
	PoolAssetIn  bool        `json:"poolAssetIn"`
	AmountInUsed uint256.Int `json:"amountInUsed"`
	AmountOut    uint256.Int `json:"amountOut"`
	SpreadPpm    uint256.Int `json:"spreadPpm,omitempty"`

	// lineage is the token of the adopted state the fill was quoted on (PoolSimulator.lineage), and amountIn the
	// input the fill was quoted for: a lever-down is sized on the whole input, not on AmountInUsed
	// (FLAMMLeverLib.sol:124-141), so the input is part of what identifies the post-state.
	lineage  uint64
	amountIn uint256.Int
	seq      uint64
	next     *flammState
}
