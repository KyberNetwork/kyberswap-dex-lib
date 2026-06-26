package hyperamm

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

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
	Extra
	StaticExtra
	reserves  [2]*uint256.Int
	precision *uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	} else if extra.IsPaused {
		return nil, ErrPoolPaused
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
		Extra:       extra,
		StaticExtra: staticExtra,
		reserves:    [2]*uint256.Int{uint256.MustFromDecimal(ep.Reserves[0]), uint256.MustFromDecimal(ep.Reserves[1])},
		precision:   big256.TenPow(18 + ep.Tokens[1].Decimals - ep.Tokens[0].Decimals),
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
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenOut
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	// Select the stored fair price and reference fee for this direction.
	fairPrice, feeBps := s.FairPriceFrom[indexIn], s.RefFeeFrom[indexIn]
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
	if feeBpsUint.Cmp(big256.UBasisPoint) >= 0 {
		// Fee >= 100 % – refuse to quote.
		return nil, ErrZeroAmountOut
	}
	bpsMinusFee := feeBpsUint.Sub(big256.UBasisPoint, feeBpsUint)
	amountInAfterFee := big256.MulDivDown(amountIn, amountIn, bpsMinusFee, big256.UBasisPoint)

	// Apply the oracle price.
	var amountOut *uint256.Int
	if indexIn == 0 {
		// amountOut = amountInAfterFee * pricePrec / fairPrice
		amountOut = big256.MulDivDown(amountInAfterFee, amountInAfterFee, s.precision, fairPrice)
	} else {
		// amountOut = amountInAfterFee * fairPrice / pricePrec
		amountOut = big256.MulDivDown(amountInAfterFee, amountInAfterFee, fairPrice, s.precision)
	}
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	// Check that the pool has enough reserve to cover the output.
	if reserveOut := s.reserves[indexOut]; amountOut.Gt(reserveOut) {
		return nil, ErrInsufficientReserve
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenIn, Amount: bignum.ZeroBI},
		Gas:            defaultGas,
	}, nil
}

// CloneState returns a deep copy of the simulator.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

// UpdateBalance adjusts the in-memory reserves after a swap is included in a
// route plan.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	amountIn, _ := uint256.FromBig(params.TokenAmountIn.Amount)
	amountOut, _ := uint256.FromBig(params.TokenAmountOut.Amount)
	s.reserves[indexIn] = amountIn.Add(s.reserves[indexIn], amountIn)
	s.reserves[indexOut] = amountOut.Sub(s.reserves[indexOut], amountOut)
}

// GetMetaInfo returns the direction flag needed by the transaction builder.
func (s *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return MetaInfo{
		ApprovalAddress: Router,
		BlockNumber:     s.Info.BlockNumber,
		IsZeroToOne:     tokenIn == s.GetTokens()[0],
	}
}
