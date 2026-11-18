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

// GetNewPoolState reads the pool's whole state for the record's partner with one getPoolStateV1
// call to the record's router, so every value comes from one block, and the record's BlockNumber
// is the router's number for that block. A pool whose swaps by the partner revert now, for example
// a paused one or one with a missing role, gets zero reserves, and Extra keeps the flags that tell
// why. A stale price keeps the reserves: the simulator checks the price's age when it quotes.
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
	if !common.IsHexAddress(staticExtra.PartnerAddress) {
		return p, fmt.Errorf("%w: partner address %q", ErrInvalidPoolRecord, staticExtra.PartnerAddress)
	}

	var out tuple[rpcPoolStateV1]
	// getPoolStateV1 accepts only a simulation, whose origin is the zero address.
	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetFrom(common.Address{}).
		AddCall(&ethrpc.Call{ABI: routerABI, Target: staticExtra.RouterAddress, Method: routerMethodGetPoolStateV1,
			Params: []any{common.HexToAddress(p.Address), common.HexToAddress(staticExtra.PartnerAddress)}},
			[]any{&out}).
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
		Routed:                 state.Routed,
		RouterPaused:           state.RouterPaused,
		PoolPaused:             state.PoolPaused,
		RouterSwapper:          state.RouterSwapper,
		PoolPriceReader:        state.PoolPriceReader,
		PartnerRole:            state.PartnerRole,
		PartnerFeeRecipientSet: state.PartnerFeeRecipientSet,
		PriceAvailable:         state.PriceAvailable,
		BlockTimestamp:         state.BlockTimestamp.Uint64(),
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

// swapError is the reason that the partner's swaps on the pool revert now, apart from the walk and
// the price's age, or nil: the router must route the pair through the pool, neither the router
// nor the pool may be paused, the pool must let the router swap and the oracle must let the pool
// read prices, the partner needs its role and a fee recipient, and the pool needs a price.
func (e *Extra) swapError() error {
	switch {
	case !e.Routed:
		return ErrNoRoute
	case e.RouterPaused || e.PoolPaused:
		return ErrPaused
	case !e.RouterSwapper || !e.PoolPriceReader:
		return ErrMissingRole
	case !e.PartnerRole || !e.PartnerFeeRecipientSet:
		return ErrPartnerNotEnabled
	case !e.PriceAvailable:
		return ErrPriceUnavailable
	}
	return nil
}

// reserves is levelBalances in the record's token order (base, then quote) when swapError is nil.
// Otherwise it is zero, so the route finder skips the pool.
func (e *Extra) reserves() entity.PoolReserves {
	if e.swapError() != nil {
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
