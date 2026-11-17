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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nightly-prop/math"
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

	extra, err := newExtra(state)
	if err != nil {
		return p, err
	}
	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = extra.reserves()
	p.BlockNumber = state.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
}

// newExtra converts the router's PoolStateV1 to the pool state that the math walks. A position
// fraction above uint104 is an error, as the contract's type cannot hold it.
func newExtra(state rpcPoolStateV1) (Extra, error) {
	fraction, err := math.FractionFromRaw(uint256.MustFromBig(state.PositionFraction))
	if err != nil {
		return Extra{}, fmt.Errorf("nightly-prop: position fraction: %w", err)
	}
	levels := make([]math.Level, len(state.Levels))
	for i, level := range state.Levels {
		levels[i] = math.Level{
			Config: math.PriceLevelConfig{
				LowerOffset:    math.SignedSpread(level.LowerOffset),
				UpperOffset:    math.SignedSpread(level.UpperOffset),
				BidExtraSpread: math.Spread(level.BidExtraSpread),
				AskExtraSpread: math.Spread(level.AskExtraSpread),
			},
			Liquidity: math.Liquidity{
				BaseAmount:  *uint256.MustFromBig(level.BaseAmount),
				QuoteAmount: *uint256.MustFromBig(level.QuoteAmount),
			},
		}
	}
	return Extra{
		Routed:         state.Routed,
		RouterPaused:   state.RouterPaused,
		PoolPaused:     state.PoolPaused,
		PriceAvailable: state.PriceAvailable,
		BlockTimestamp: state.BlockTimestamp.Uint64(),
		Price: math.OraclePrice{
			MidPrice:    state.MidPrice,
			BidSpread:   math.Spread(state.BidSpread),
			AskSpread:   math.Spread(state.AskSpread),
			TimestampMs: state.PriceTimestampMs.Uint64(),
		},
		Pool: math.Pool{
			BaseDecimals:  state.BaseDecimals,
			QuoteDecimals: state.QuoteDecimals,
			MaxPriceAgeMs: state.MaxPriceAgeMs,
			LowestLevelID: state.LowestLevelID,
			Levels:        levels,
			Position:      math.Position{LevelID: state.PositionLevelID, Fraction: fraction},
		},
	}, nil
}

// swappable reports whether a partner swap can run on the pool now: the router routes its pair
// through it, neither the router nor the pool is paused, and the pool has a price.
func (e *Extra) swappable() bool {
	return e.Routed && !e.RouterPaused && !e.PoolPaused && e.PriceAvailable
}

// reserves is levelBalances for a swappable pool, in the record's token order (base, then quote).
// It is zero for a pool that is not swappable, so the route finder skips it.
func (e *Extra) reserves() entity.PoolReserves {
	if !e.swappable() {
		return entity.PoolReserves{"0", "0"}
	}
	base, quote := e.levelBalances()
	return entity.PoolReserves{base.Dec(), quote.Dec()}
}

// levelBalances sums the level balances per token.
func (e *Extra) levelBalances() (base, quote *uint256.Int) {
	base, quote = new(uint256.Int), new(uint256.Int)
	for _, level := range e.Pool.Levels {
		base.Add(base, &level.Liquidity.BaseAmount)
		quote.Add(quote, &level.Liquidity.QuoteAmount)
	}
	return base, quote
}
