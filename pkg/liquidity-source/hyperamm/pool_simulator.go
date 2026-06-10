package hyperamm

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// scale18 is 1e18, the fixed-point denominator used by fairPriceInScale18.
var scale18 = uint256.MustFromDecimal(scale18Str)

// bps is 10 000 – the denominator for basis-point fee arithmetic.
var bps = uint256.MustFromDecimal(bpsStr)

// PoolSimulator is the in-memory swap simulator for HyperAMM.
//
// HyperAMM is an oracle-priced AMM on Hyperliquid EVM.  Swaps are quoted from
// HyperCore oracle prices (via HyperAMMSwapFeeModule.fairPriceInScale18) with a
// dynamic fee that includes a base fee, an imbalance component, a market-impact
// component, and a premium adjustment.
//
// The tracker stores a snapshot of the fair price and the effective reference
// fee (from previewSwapFeeInBips with a 1-unit input).  The simulator applies
// the stored fee directly – this is accurate for amounts close to the reference
// and an approximation for larger amounts (the market-impact component scales
// with size).
type PoolSimulator struct {
	pool.Pool
	extra       Extra
	staticExtra StaticExtra
	reserve0    *uint256.Int
	reserve1    *uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	tokens := lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address })
	reserves := lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignum.NewBig(r) })

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: ep.BlockNumber,
		}},
		extra:       extra,
		staticExtra: staticExtra,
		reserve0:    uint256.MustFromDecimal(ep.Reserves[0]),
		reserve1:    uint256.MustFromDecimal(ep.Reserves[1]),
	}, nil
}

// CalcAmountOut computes the output amount for an exact-input swap.
//
// Formula (mirrors the on-chain HyperAMM swap path):
//
//	amountInAfterFee = amountIn × (10000 − feeBps) / 10000
//	amountOut        = amountInAfterFee × fairPrice / 1e18
//
// where fairPrice is `SwapFeeModule.fairPriceInScale18(isZeroToOne)`.  The
// premium adjustment is already folded into the fair price by the fee module.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.extra.IsPaused {
		return nil, ErrPoolPaused
	}

	tokenIn := params.TokenAmountIn.Token
	tokenOut := params.TokenOut

	indexIn := s.GetTokenIndex(tokenIn)
	indexOut := s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	isZeroToOne := indexIn == 0

	// Select the stored fair price and reference fee for this direction.
	var fairPrice *uint256.Int
	var feeBps uint64
	if isZeroToOne {
		fairPrice = s.extra.FairPrice0To1
		feeBps = s.extra.RefFee0To1
	} else {
		fairPrice = s.extra.FairPrice1To0
		feeBps = s.extra.RefFee1To0
	}

	if fairPrice == nil || fairPrice.IsZero() {
		return nil, ErrZeroFairPrice
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.IsZero() {
		return nil, ErrZeroAmountIn
	}

	// Deduct the fee from the input amount.
	// amountInAfterFee = amountIn * (bps - feeBps) / bps
	feeBpsUint := new(uint256.Int).SetUint64(feeBps)
	if feeBpsUint.Cmp(bps) >= 0 {
		// Fee >= 100 % – refuse to quote.
		return nil, ErrZeroAmountOut
	}
	bpsMinusFee := new(uint256.Int).Sub(bps, feeBpsUint)

	amountInAfterFee, overflow := new(uint256.Int).MulDivOverflow(amountIn, bpsMinusFee, bps)
	if overflow {
		return nil, ErrOverflow
	}

	// Apply the oracle price.
	// amountOut = amountInAfterFee * fairPrice / 1e18
	amountOut, overflow := new(uint256.Int).MulDivOverflow(amountInAfterFee, fairPrice, scale18)
	if overflow {
		return nil, ErrOverflow
	}

	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	// Check that the pool has enough reserve to cover the output.
	reserveOut := lo.Ternary(isZeroToOne, s.reserve1, s.reserve0)
	if amountOut.Gt(reserveOut) {
		return nil, ErrInsufficientReserve
	}

	feeAmount := new(uint256.Int).Sub(amountIn, amountInAfterFee)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: feeAmount.ToBig(),
		},
		Gas:      defaultGas,
		SwapInfo: SwapInfo{IsZeroToOne: isZeroToOne},
	}, nil
}

// UpdateBalance adjusts the in-memory reserves after a swap is included in a
// route plan.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return
	}
	amountOut, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow {
		return
	}

	if swapInfo.IsZeroToOne {
		s.reserve0.Add(s.reserve0, amountIn)
		if amountOut.Gt(s.reserve1) {
			s.reserve1.Clear()
		} else {
			s.reserve1.Sub(s.reserve1, amountOut)
		}
	} else {
		s.reserve1.Add(s.reserve1, amountIn)
		if amountOut.Gt(s.reserve0) {
			s.reserve0.Clear()
		} else {
			s.reserve0.Sub(s.reserve0, amountOut)
		}
	}
}

// CloneState returns a deep copy of the simulator.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.reserve0 = new(uint256.Int).Set(s.reserve0)
	cloned.reserve1 = new(uint256.Int).Set(s.reserve1)
	cloned.extra = Extra{
		FairPrice0To1: new(uint256.Int).Set(s.extra.FairPrice0To1),
		FairPrice1To0: new(uint256.Int).Set(s.extra.FairPrice1To0),
		BaseFeeBps:    s.extra.BaseFeeBps,
		RefFee0To1:    s.extra.RefFee0To1,
		RefFee1To0:    s.extra.RefFee1To0,
		IsPaused:      s.extra.IsPaused,
	}
	return &cloned
}

// GetMetaInfo returns the direction flag needed by the transaction builder.
func (s *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return MetaInfo{
		BlockNumber: s.Info.BlockNumber,
		IsZeroToOne: s.GetTokenIndex(tokenIn) == 0,
	}
}
