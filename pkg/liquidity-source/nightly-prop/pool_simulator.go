package nightlyprop

import (
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nightly-prop/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator quotes exact-input swaps before any fee selected by the executor.
// Token 0 is the base token and token 1 is the quote token.
type PoolSimulator struct {
	pool.Pool
	Extra       Extra
	StaticExtra StaticExtra
	StaleCheck  bool
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

// NewPoolSimulator reads a record that the tracker refreshed. A record without state, as
// GetNewPools lists it, has an empty Extra and fails here.
func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	ep := params.EntityPool
	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, fmt.Errorf("%w: %d tokens and %d reserves", ErrInvalidPoolRecord, len(ep.Tokens),
			len(ep.Reserves))
	}
	s := &PoolSimulator{StaleCheck: params.Opts.StaleCheck}
	if err := json.Unmarshal([]byte(ep.Extra), &s.Extra); err != nil {
		return nil, fmt.Errorf("nightly-prop: decode extra: %w", err)
	}
	if err := json.Unmarshal([]byte(ep.StaticExtra), &s.StaticExtra); err != nil {
		return nil, fmt.Errorf("nightly-prop: decode static extra: %w", err)
	}
	s.Info = pool.PoolInfo{
		Address:     strings.ToLower(ep.Address),
		Exchange:    ep.Exchange,
		Type:        ep.Type,
		Tokens:      []string{ep.Tokens[0].Address, ep.Tokens[1].Address},
		Reserves:    []*big.Int{bignumber.NewBig(ep.Reserves[0]), bignumber.NewBig(ep.Reserves[1])},
		BlockNumber: ep.BlockNumber,
	}
	return s, nil
}

// CalcAmountOut runs the pool's exact-in walk on the record's state, as swapExactIn does. Its
// SwapInfo is the walk's *math.Swap, which UpdateBalance applies.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	in, out := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if in < 0 || out < 0 || in == out {
		return nil, ErrInvalidToken
	}
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() < 0 {
		return nil, ErrInvalidAmount
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmount
	}
	if err := s.Extra.swapError(); err != nil {
		return nil, err
	}

	walk := s.Extra.Pool.SellExactBase
	if in == 1 {
		walk = s.Extra.Pool.BuyWithExactQuote
	}
	swap, err := walk(amountIn, s.Extra.Price, s.blockTimestamp())
	if err != nil {
		return nil, err
	}
	// The walk visits consecutive levels. A level without output liquidity has no write.
	start, end := s.Extra.Pool.Position.LevelID, swap.Quote.EndPosition.LevelID
	visited := int64(max(start, end)) - int64(min(start, end)) + 1
	written := int64(len(swap.Writes))
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[out], Amount: swap.Quote.AmountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[in], Amount: new(big.Int)},
		Gas:            singleLevelGas + (written-1)*extraWrittenLevelGas + (visited-written)*skippedLevelGas,
		SwapInfo:       &swap,
	}, nil
}

// AmountInRange reports the maximum remaining input in native token units. The lower bound is
// open because the pool has no configured minimum. Small inputs can round to zero output.
func (s *PoolSimulator) AmountInRange(tokenIn, tokenOut string, _ pool.SwapLimit) (*big.Int, *big.Int, error) {
	in, out := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if in < 0 || out < 0 || in == out {
		return nil, nil, ErrInvalidToken
	}
	if s.Extra.swapError() != nil {
		return nil, new(big.Int), nil
	}
	maxIn, err := s.Extra.Pool.MaxAmountIn(in == 1, s.Extra.Price, s.blockTimestamp())
	if errors.Is(err, math.ErrStaleOraclePrice) {
		return nil, new(big.Int), nil
	}
	if err != nil {
		// A smaller swap may still succeed before it reaches the level that failed.
		return nil, nil, fmt.Errorf("%w: %w", pool.ErrUnsupported, err)
	}
	return nil, maxIn.ToBig(), nil
}

// blockTimestamp is the block time, in seconds, for the walk's price age check: the quote's time.
// Without StaleCheck it is zero, for which no price is too old.
func (s *PoolSimulator) blockTimestamp() uint64 {
	if !s.StaleCheck {
		return 0
	}
	return uint64(time.Now().Unix())
}

// UpdateBalance applies the swap of CalcAmountOut's SwapInfo, the level balances and the end
// position that the swap writes, and sets the reserves that follow from the levels.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swap, ok := params.SwapInfo.(*math.Swap)
	if !ok || s.Extra.Pool.Apply(*swap) != nil {
		return
	}
	base, quote := s.Extra.levelBalances()
	s.Info.Reserves = []*big.Int{base.ToBig(), quote.ToBig()}
}

// CloneState copies the levels, which UpdateBalance writes by index. The position is a value, and
// UpdateBalance replaces the reserves slice whole.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Extra.Pool.Levels = slices.Clone(s.Extra.Pool.Levels)
	return &cloned
}

// GetMetaInfo names the router, which pulls the input from the caller.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: s.StaticExtra.RouterAddress, BlockNumber: s.Info.BlockNumber}
}

// GetApprovalAddress is the router, which pulls the input from the caller.
func (s *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return s.StaticExtra.RouterAddress
}
