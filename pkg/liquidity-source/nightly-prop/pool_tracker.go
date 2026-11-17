package nightlyprop

import (
	"context"
	"fmt"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

// NewPoolTracker takes no setting from cfg: each record names its router in StaticExtra.
func NewPoolTracker(_ *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{ethrpcClient: ethrpcClient}
}

// GetNewPoolState reads the pool's whole state with one getPoolStateV1 call to the record's
// router, so every value comes from one block, and the record's BlockNumber is the router's
// number for that block. A pool that the router cannot swap now, for example a paused one or one
// whose price is stale, gets zero reserves, and Extra keeps the flags that tell why.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, fmt.Errorf("nightly-prop: decode static extra: %w", err)
	}
	if len(p.Tokens) != 2 {
		return p, fmt.Errorf("%w: %d tokens", ErrInvalidPoolRecord, len(p.Tokens))
	}

	var out tuple[rpcPoolStateV1]
	// getPoolStateV1 accepts only a simulation, whose origin is the zero address.
	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetFrom(common.Address{}).
		AddCall(&ethrpc.Call{ABI: routerABI, Target: staticExtra.RouterAddress, Method: routerMethodGetPoolStateV1,
			Params: []any{common.HexToAddress(p.Address)}}, []any{&out}).
		Call(); err != nil {
		return p, fmt.Errorf("nightly-prop: getPoolStateV1: %w", err)
	}
	state := out.Value
	if state.Base != common.HexToAddress(p.Tokens[0].Address) || state.Quote != common.HexToAddress(p.Tokens[1].Address) {
		return p, fmt.Errorf("%w: the pool's tokens are %s and %s", ErrInvalidPoolRecord, state.Base, state.Quote)
	}

	extra := newExtra(state)
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = extra.reserves()
	p.BlockNumber = state.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
}

func newExtra(state rpcPoolStateV1) Extra {
	levels := make([]Level, len(state.Levels))
	for i, level := range state.Levels {
		levels[i] = Level{
			LowerOffset:    level.LowerOffset,
			UpperOffset:    level.UpperOffset,
			BidExtraSpread: level.BidExtraSpread,
			AskExtraSpread: level.AskExtraSpread,
			BaseAmount:     uint256.MustFromBig(level.BaseAmount),
			QuoteAmount:    uint256.MustFromBig(level.QuoteAmount),
		}
	}
	return Extra{
		Routed:         state.Routed,
		RouterPaused:   state.RouterPaused,
		PoolPaused:     state.PoolPaused,
		PriceAvailable: state.PriceAvailable,
		BlockTimestamp: state.BlockTimestamp.Uint64(),
		BaseDecimals:   state.BaseDecimals,
		QuoteDecimals:  state.QuoteDecimals,
		Price: Price{
			MidPrice:    state.MidPrice,
			BidSpread:   state.BidSpread,
			AskSpread:   state.AskSpread,
			TimestampMs: state.PriceTimestampMs.Uint64(),
		},
		MaxPriceAgeMs: state.MaxPriceAgeMs,
		Position: Position{
			LevelID:  state.PositionLevelID,
			Fraction: uint256.MustFromBig(state.PositionFraction),
		},
		LowestLevelID: state.LowestLevelID,
		Levels:        levels,
	}
}

// swappable reports whether a partner swap can run on the pool now: the router routes its pair
// through it, neither the router nor the pool is paused, and the pool has a price.
func (e *Extra) swappable() bool {
	return e.Routed && !e.RouterPaused && !e.PoolPaused && e.PriceAvailable
}

// reserves sums the level balances per token, in the record's token order (base, then quote). It
// is zero for a pool that is not swappable, so the route finder skips it.
func (e *Extra) reserves() entity.PoolReserves {
	if !e.swappable() {
		return entity.PoolReserves{"0", "0"}
	}
	base, quote := new(uint256.Int), new(uint256.Int)
	for _, level := range e.Levels {
		base.Add(base, level.BaseAmount)
		quote.Add(quote, level.QuoteAmount)
	}
	return entity.PoolReserves{base.Dec(), quote.Dec()}
}
