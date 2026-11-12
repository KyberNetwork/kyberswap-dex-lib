package everlongflamm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// StaticExtra is fixed at listing: the Factory the pool was found at, whose `isPool` every refresh re-reads.
type StaticExtra struct {
	Factory string `json:"factory"`
}

// Point is one probe of a direction: the input the pool used and the net output it pays, in base units, exact
// unless the ladder's rule lowered it (settle).
type Point [2]*uint256.Int

// Ladder is one direction's probes at the refresh block, settled: strictly ascending in input, the output never
// falling, at most maxLadderPoints points.
type Ladder struct {
	In     uint8   `json:"i"`
	Out    uint8   `json:"j"`
	Points []Point `json:"p"`
	// Flat has bit k set where span k, from point k-1 to point k, quotes point k-1's output: a chord there could pay
	// more than the pool, as across a change of route. A direction with a flat span takes no second fill.
	Flat uint64 `json:"f,omitempty"`
	// Capped is set only on a clip of the curve's route at the top: the last probe used less than it was asked, the
	// pool's input ceiling for the direction, where a larger input clips at the same point or fills whole at a better
	// output. A larger input then fills at that point and returns the rest.
	Capped bool `json:"c,omitempty"`

	// The charge (gas.go). Cells are the paths a fill of the direction may take, the table's for the pricer's build,
	// the chain, each live venue kind on the pair and the side, each at its least: a ladder without them is not
	// quoted. The reads that narrow them at each fill, all at the refresh block: Held, the output's custody
	// (tokenConfig.held); Freed, the output the pool frees before a draw, its held, supplied and posted (the
	// request's legs); OwedOut and OwedIn, each token's debt (owedUnknown, owedNone, owedSome); FirstOwing, a borrow
	// of the output may be a first one while it owes, as a second live venue lends it; PostedIn, the input has
	// collateral posted (its posted leg), so a borrow of the output while it owes nothing may take an owing borrow's
	// cells; Repay, the input owes and its repay is certain. Nil or zero: not read. Second is the charge of a fill
	// after another fill of the route moved the pool, in the same transaction: the least second-fill charge of the
	// pair's live kinds. A ladder without it is not quoted either. After are the same paths at the gas of the pool's
	// first fill after another pool of this source filled earlier in the same transaction, never above their cells,
	// where the table measured such fills: a fill is charged them where the route's swap limit says another pool
	// filled (pool_simulator.go). A ladder without them charges its cells then.
	Cells      []Cell       `json:"cs"`
	Second     int64        `json:"sc"`
	After      []Cell       `json:"ac,omitempty"`
	Held       *uint256.Int `json:"ho,omitempty"`
	Freed      *uint256.Int `json:"fr,omitempty"`
	OwedOut    uint8        `json:"oo,omitempty"`
	OwedIn     uint8        `json:"oi,omitempty"`
	FirstOwing bool         `json:"fw,omitempty"`
	PostedIn   bool         `json:"pi,omitempty"`
	Repay      bool         `json:"rp,omitempty"`
}

// Extra is the refresh: a ladder per direction that trades and is routed now, and the haircut taken off every
// amount interpolated between two probes. A paused pool, a closed pair or a refused hook has no ladder.
type Extra struct {
	Ladders    []Ladder `json:"l"`
	HaircutBps uint64   `json:"h"`
	// LiveAt is the time of a refresh that routes something. pool-service saves only a state that changed, and a quiet
	// pool's ladders can repeat to the byte, so without it such a pool's Timestamp would stop and MaxAge would take it
	// out of routing, refreshed or not.
	LiveAt int64 `json:"lv,omitempty"`
}

// SwapInfo keeps the shape the encoder reads: venue 0, the adapter's swap kind, for every pair of every pool.
type SwapInfo struct {
	Venue        uint8  `json:"venue"`
	PoolAssetIn  bool   `json:"poolAssetIn"`
	AmountInUsed string `json:"amountInUsed"`
	AmountOut    string `json:"amountOut"`
}

// PoolMeta names the contract to approve, the pool, which pulls the input in `swap`, and the refresh block.
type PoolMeta struct {
	ApprovalAddress string `json:"approvalAddress"`
	BlockNumber     uint64 `json:"blockNumber"`
}

// Metadata is the lister's cursor: how many tokens each listed pool had, so a token added later re-lists it.
type Metadata struct {
	Pools map[string]int `json:"pools"`
}

// ABI result shapes. A function returning one tuple is decoded into a struct wrapping it: go-ethereum copies a
// single output into the target's first field.

type hookSet struct {
	Hook       common.Address `abi:"hook"`
	Controller common.Address `abi:"controller"`
}

type hooksResult struct {
	Set hookSet
}

type tokenConfig struct {
	Token            common.Address `abi:"token"`
	Decimals         uint8          `abi:"decimals"`
	SwapPriceBandWad uint64         `abi:"swapPriceBandWad"`
	FeeFloorWad      uint64         `abi:"feeFloorWad"`
	MaxSwapNotional  *big.Int       `abi:"maxSwapNotional"`
	ReserveTarget    *big.Int       `abi:"reserveTarget"`
	MinSupplyBatch   *big.Int       `abi:"minSupplyBatch"`
	Held             *big.Int       `abi:"held"`
	Partner          uint8          `abi:"partner"`
	PairId           uint8          `abi:"pairId"`
	DebtFree         bool           `abi:"debtFree"`
}

type tokenConfigResult struct {
	Config tokenConfig
}

type tradableResult struct {
	Tradable      *big.Int `abi:"tradable"`
	HookPairs     *big.Int `abi:"hookPairs"`
	OperatorPairs *big.Int `abi:"operatorPairs"`
	PendingPairs  *big.Int `abi:"pendingPairs"`
	ExecutableAt  *big.Int `abi:"executableAt"`
}

type pairHookResult struct {
	Hook common.Address `abi:"hook"`
	Mask *big.Int       `abi:"mask"`
}

type laneResult struct {
	LaneWord   *big.Int `abi:"laneWord"`
	LaneConfig *big.Int `abi:"laneConfig"`
	Pending    *big.Int `abi:"pending"`
}

// venueView is the Router's record of one of a pool's lending venues (IMMRouter.VenueView): the source reads its
// kind, the tokens of its credit pair and its status, for the kind gate and the charge of the pairs it may fund.
type venueView struct {
	Account          common.Address `abi:"account"`
	Id               [32]byte       `abi:"id"`
	Kind             uint8          `abi:"kind"`
	Pair             uint8          `abi:"pair"`
	Coll             uint8          `abi:"coll"`
	Debt             uint8          `abi:"debt"`
	LltvWad          uint64         `abi:"lltvWad"`
	BorrowEnabled    bool           `abi:"borrowEnabled"`
	SupplyEnabled    bool           `abi:"supplyEnabled"`
	Status           uint8          `abi:"status"`
	DebtCap          *big.Int       `abi:"debtCap"`
	SupplyCap        *big.Int       `abi:"supplyCap"`
	MaxBorrowRateWad uint64         `abi:"maxBorrowRateWad"`
}

type venueResult struct {
	View venueView
}
