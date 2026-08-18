package parityprop

import (
	"errors"
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Token index convention: Tokens[0] is always base, Tokens[1] is always
// quote -- set by pool_list_updater.go (base()/quote() are read in that
// order) and relied on throughout this file.
const (
	baseIdx  = 0
	quoteIdx = 1
)

type PoolSimulator struct {
	pool.Pool

	staticExtra StaticExtra
	extra       Extra

	baseScale, quoteScale *uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	if len(ep.Tokens) != 2 {
		return nil, errors.New("parity-prop: pool must have exactly 2 tokens")
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if len(ep.Extra) > 0 {
		if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
			return nil, err
		}
	}

	baseScale, err := big256.NewUint256(staticExtra.BaseScale)
	if err != nil {
		return nil, err
	}
	quoteScale, err := big256.NewUint256(staticExtra.QuoteScale)
	if err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      []string{ep.Tokens[0].Address, ep.Tokens[1].Address},
			Reserves:    lo.Map(ep.Reserves, func(s string, _ int) *big.Int { return bignumber.NewBig(s) }),
			BlockNumber: ep.BlockNumber,
		}},
		staticExtra: staticExtra,
		extra:       extra,
		baseScale:   baseScale,
		quoteScale:  quoteScale,
	}, nil
}

// CalcAmountOut ports PmmPool.sol's _quote(): paused/zero-amount are checked
// here (they don't need oracle/param state), then Quote (math.go) replicates
// the oracle-sanity / staleness / spread / notional-cap / reserve-floor
// guards and the linear bid/ask pricing formula in the exact on-chain order.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.extra.Paused {
		return nil, ErrPoolPaused
	}

	tokenIn := params.TokenAmountIn.Token
	tokenOut := params.TokenOut
	indexIn := s.GetTokenIndex(tokenIn)
	indexOut := s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn == nil {
		return nil, ErrZeroAmount
	}
	if amountIn.IsZero() {
		return nil, ErrZeroAmount
	}

	tokenInIsBase := indexIn == baseIdx

	outIdx := quoteIdx
	outFloor := s.extra.MinQuoteReserve
	if !tokenInIsBase {
		outIdx = baseIdx
		outFloor = s.extra.MinBaseReserve
	}
	if len(s.Info.Reserves) <= outIdx || s.Info.Reserves[outIdx] == nil {
		return nil, ErrInsufficientReserve
	}
	outBalance, overflow := uint256.FromBig(s.Info.Reserves[outIdx])
	if overflow || outBalance == nil {
		return nil, ErrInsufficientReserve
	}

	// accumulated = (lastSwapBlock == block.number) ? blockNotional : 0.
	// The simulator's closest analog of "block.number" is the block its
	// state snapshot was taken at (Info.BlockNumber); UpdateBalance is
	// responsible for keeping LastSwapBlock/BlockNotional consistent with
	// that same reference block across chained swaps within one route.
	accumulated := new(uint256.Int)
	if s.extra.BlockNotional != nil && s.Info.BlockNumber != 0 && s.extra.LastSwapBlock == s.Info.BlockNumber {
		accumulated.Set(s.extra.BlockNotional)
	}

	result, err := Quote(QuoteParams{
		TokenInIsBase:            tokenInIsBase,
		AmountIn:                 amountIn,
		Bid:                      s.extra.OracleBid,
		Mid:                      s.extra.OracleMid,
		Ask:                      s.extra.OracleAsk,
		PriceUpdatedAt:           s.extra.OracleUpdatedAt,
		NowTimestamp:             uint64(time.Now().Unix()),
		MaxStaleness:             s.extra.MaxStaleness,
		MaxSpreadBps:             s.extra.MaxSpreadBps,
		FeeBps:                   s.extra.FeeBps,
		BaseScale:                s.baseScale,
		QuoteScale:               s.quoteScale,
		MaxSwapNotional:          s.extra.MaxSwapNotional,
		MaxBlockNotional:         s.extra.MaxBlockNotional,
		AccumulatedBlockNotional: accumulated,
		OutBalance:               outBalance,
		OutReserveFloor:          outFloor,
	})
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: result.AmountOut.ToBig()},
		// Fee is denominated in the output token -- PmmPool.sol takes feeBps
		// out of `gross` (the output side) before returning amountOut.
		Fee: &pool.TokenAmount{Token: tokenOut, Amount: result.Fee.ToBig()},
		Gas: defaultGas,
		SwapInfo: SwapInfo{
			Notional:      result.Notional,
			SwapBlock:     s.Info.BlockNumber,
			BlockNotional: new(uint256.Int).Add(accumulated, result.Notional),
		},
	}, nil
}

// UpdateBalance consumes the SwapInfo produced by CalcAmountOut to update
// reserves and the per-block notional accounting (LastSwapBlock/
// BlockNotional), mirroring PmmPool.swap()'s on-chain state transition.
// Every write reassigns a slice element / struct field to a freshly
// allocated value rather than mutating an existing pointer's contents in
// place -- see CloneState below for why that discipline is load-bearing.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return
	}

	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	if s.Info.Reserves[indexOut].Cmp(params.TokenAmountOut.Amount) >= 0 {
		s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
	} else {
		s.Info.Reserves[indexOut] = new(big.Int)
	}

	// CalcAmountOut already resolved the on-chain accumulate-or-reset branch
	// (lastSwapBlock == block.number) into SwapInfo.BlockNotional -- this is
	// a blind wholesale assign of that result, not a re-derivation. See the
	// `accumulated :=` comment in CalcAmountOut above.
	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		s.extra.LastSwapBlock = swapInfo.SwapBlock
		s.extra.BlockNotional = swapInfo.BlockNotional
	}
}

// CloneState deep-copies Info.Reserves (written by index above) into a new
// slice of new *big.Int. It deliberately does NOT deep-copy extra's pointer
// fields (BlockNotional, MaxSwapNotional, ...): UpdateBalance only ever
// reassigns s.extra.BlockNotional wholesale to a freshly allocated
// *uint256.Int, never mutates through the existing pointer, so the shallow
// struct copy `cloned := *s` already isolates clone from original. If
// UpdateBalance ever starts mutating extra.BlockNotional in place, this
// needs revisiting.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = lo.Map(s.Info.Reserves, func(r *big.Int, _ int) *big.Int {
		if r == nil {
			return nil
		}
		return new(big.Int).Set(r)
	})
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{BlockNumber: s.Info.BlockNumber}
}
