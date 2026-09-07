package spireprop

import (
	"errors"

	"github.com/holiman/uint256"
)

const DexType = "spire-prop"
const defaultGas int64 = 200000

var (
	ErrInvalidState = errors.New("spire-prop: invalid state")
	ErrInvalidToken = errors.New("spire-prop: invalid token")
	ErrAmount       = errors.New("spire-prop: invalid amount")
	ErrExpired      = errors.New("spire-prop: curve expired")
	ErrDepth        = errors.New("spire-prop: beyond depth")
	ErrOverflow     = errors.New("spire-prop: arithmetic overflow")
)

type Config struct {
	DexID      string `json:"dexID"`
	Entrypoint string `json:"entrypoint"`
	// The protocol has no enumerable factory. List configured bases; verify the
	// immutable quote/curve/custodian on chain before emitting each pair.
	Bases []string `json:"bases"`
}

type StaticExtra struct {
	Entrypoint string `json:"entrypoint"`
	CurveBook  string `json:"curveBook"`
	Custodian  string `json:"custodian"`
}

type Knot struct {
	Q     uint256.Int `json:"q"`
	Extra uint256.Int `json:"extra"`
}

type Side struct {
	SpreadBps int16       `json:"spreadBps"`
	DepthBps  uint16      `json:"depthBps"`
	Filled    uint256.Int `json:"filled"`
	Knots     []Knot      `json:"knots"`
}

type Extra struct {
	Seq            uint64      `json:"seq"`
	FillSeq        uint64      `json:"fillSeq"`
	LastUpdateAt   uint64      `json:"lastUpdateAt"`
	ValidUntil     uint64      `json:"validUntil"`
	TTL            uint64      `json:"ttl"`
	BlockTimestamp uint64      `json:"blockTimestamp"`
	Mid            uint256.Int `json:"mid"`
	QUnit          uint256.Int `json:"qUnit"`
	CUnit          uint256.Int `json:"cUnit"`
	Ask            Side        `json:"ask"`
	Bid            Side        `json:"bid"`
}

type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	Entrypoint      string `json:"entrypoint"`
	Base            string `json:"base"`
	ApprovalAddress string `json:"approvalAddress"`
	ValidUntil      uint64 `json:"validUntil"`
}

type SwapInfo struct {
	IndexIn   int
	FillSeq   uint64
	Cursor    uint256.Int
	AmountIn  uint256.Int
	AmountOut uint256.Int
}
