// Package ladder provides a reusable IPoolSimulator base for pool types that
// price swaps against a small set of on-chain-probed (amountIn, amountOut)
// points rather than a closed-form formula (e.g. RFQ-style prop AMMs
// wrapping a private/opaque pricing engine). Embed *ladder.PoolSimulator and
// override GetMetaInfo (and CloneState, to downcast) as needed — see
// pkg/liquidity-source/order-book for the same embed-and-override shape.
package ladder

import (
	"math"
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// MaxAge is the default staleness bound for embedders without a
// protocol-specific one, mirroring order-book's MaxAge.
const MaxAge = time.Minute

type PoolSimulator struct {
	pool.Pool

	Extra Extra
	Gas   int64

	Reserve0, Reserve1 *uint256.Int

	// splines are built once from Extra.Ladders (which never changes after
	// construction), so clones can safely share the same *Spline instances.
	splines [2]*Spline

	consumedIn  [2]float64
	consumedOut [2]float64
}

// NewPoolSimulator builds the shared ladder state from an entity.Pool, with
// no staleness enforcement. Embedders call this (or NewPoolSimulatorWith) for
// the common fields, then unmarshal their own StaticExtra and set Gas.
func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	return NewPoolSimulatorWith(ep, math.MaxInt64)
}

// NewPoolSimulatorWith is NewPoolSimulator with an explicit staleness bound:
// if the pool's state is older than maxAge, it returns ErrStale rather than
// risking a quote against a curve that's since drifted (these pools price
// against a live, continuously-updating feed -- see the "off by 10bps"
// investigation that motivated this check). Embedders that want this should
// register via pool.RegisterFactory and pass
// lo.Ternary(params.Opts.StaleCheck, MaxAge, math.MaxInt64) as maxAge (their
// own MaxAge, or this package's default), mirroring
// pkg/liquidity-source/order-book.
func NewPoolSimulatorWith(ep entity.Pool, maxAge time.Duration) (*PoolSimulator, error) {
	if time.Since(time.Unix(ep.Timestamp, 0)) > maxAge {
		return nil, ErrStale
	}
	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, ErrInvalidToken
	}

	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	r0, err := uint256.FromDecimal(ep.Reserves[0])
	if err != nil {
		return nil, err
	}
	r1, err := uint256.FromDecimal(ep.Reserves[1])
	if err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(s string, _ int) *big.Int { return bignum.NewBig(s) }),
			BlockNumber: ep.BlockNumber,
		},
		Extra:    extra,
		Reserve0: r0,
		Reserve1: r1,
		splines:  [2]*Spline{NewSpline(extra.Ladders[0]), NewSpline(extra.Ladders[1])},
	}, nil
}

// CalcAmtOut is the native uint256 quote. It shares quote with CalcAmountOut and
// differs only in reading the swap limit as uint256.
func (s *PoolSimulator) CalcAmtOut(params pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	amountOut, err := s.quote(params.TokenAmountIn.Token, params.TokenOut, params.TokenAmountIn.Amount.Float64())
	if err != nil {
		return nil, err
	}

	if limit := params.Limit; limit != nil {
		if l := limit.GetLimit(params.TokenOut); l == nil || amountOut.Gt(l) {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	return &pool.CalcAmtOutResult{
		TokenAmountOut: &pool.TokenAmt{Token: params.TokenOut, Amount: *amountOut},
		Fee:            &pool.TokenAmt{Token: params.TokenAmountIn.Token},
		Gas:            s.Gas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	amountInF, _ := params.TokenAmountIn.Amount.Float64()
	amountOut, err := s.quote(params.TokenAmountIn.Token, params.TokenOut, amountInF)
	if err != nil {
		return nil, err
	}

	if limit := params.Limit; limit != nil {
		if l := limit.GetLimit(params.TokenOut); l == nil || amountOut.ToBig().Cmp(l) > 0 {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: bignum.ZeroBI},
		Gas:            s.Gas,
	}, nil
}

// quote returns the ladder amount out for amountInF of tokenIn, net of the amount already consumed in that
// direction. It does not check the swap limit; the callers do.
func (s *PoolSimulator) quote(tokenIn, tokenOut string, amountInF float64) (*uint256.Int, error) {
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}

	if amountInF <= 0 {
		return nil, ErrZeroAmountIn
	}

	spline, reserveOut := s.splines[0], s.Reserve1
	if indexIn == 1 {
		spline, reserveOut = s.splines[1], s.Reserve0
	}

	totalIn := s.consumedIn[indexIn] + amountInF
	totalOut, err := spline.QuoteAmountOut(totalIn)
	if err != nil {
		return nil, err
	} else if totalOut < s.consumedOut[indexIn] {
		return nil, ErrNoQuote
	}
	amountOutF := totalOut - s.consumedOut[indexIn]
	if amountOutF <= 0 {
		return nil, ErrNoQuote
	}

	amountOut, ok := u256FromFloat(amountOutF)
	if !ok {
		return nil, ErrInsufficientLiquidity
	} else if amountOut.IsZero() {
		return nil, ErrNoQuote
	} else if reserveOut != nil && amountOut.Gt(reserveOut) {
		return nil, ErrInsufficientLiquidity
	}
	return amountOut, nil
}

// u256FromFloat truncates a positive finite float64 to uint256, matching big.Float.Int. It reports false when the
// value does not fit 256 bits.
func u256FromFloat(f float64) (*uint256.Int, bool) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil, false
	}
	f = math.Trunc(f)
	if f < 1<<64 {
		return new(uint256.Int).SetUint64(uint64(f)), true
	}
	// Frexp gives f = frac * 2^exp with a 53-bit exact mantissa, so the shift is exact.
	frac, exp := math.Frexp(f)
	if exp > 256 {
		return nil, false
	}
	z := new(uint256.Int).SetUint64(uint64(math.Ldexp(frac, 53)))
	return z.Lsh(z, uint(exp-53)), true
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	if indexIn < 0 || indexIn > 1 {
		return
	}
	inU, inOverflow := uint256.FromBig(params.TokenAmountIn.Amount)
	outU, outOverflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if inOverflow || outOverflow || inU == nil || outU == nil {
		return
	}

	inF, _ := params.TokenAmountIn.Amount.Float64()
	outF, _ := params.TokenAmountOut.Amount.Float64()
	s.consumedIn[indexIn] += inF
	s.consumedOut[indexIn] += outF

	if indexIn == 0 {
		s.Reserve0 = new(uint256.Int).Add(s.Reserve0, inU)
		s.Reserve1 = new(uint256.Int).Sub(s.Reserve1, outU)
	} else {
		s.Reserve1 = new(uint256.Int).Add(s.Reserve1, inU)
		s.Reserve0 = new(uint256.Int).Sub(s.Reserve0, outU)
	}
	s.Info.Reserves[0] = s.Reserve0.ToBig()
	s.Info.Reserves[1] = s.Reserve1.ToBig()

	if limit := params.SwapLimitBig; limit != nil {
		_, _, _ = limit.UpdateLimit(
			params.TokenAmountOut.Token,
			params.TokenAmountIn.Token,
			params.TokenAmountOut.Amount,
			params.TokenAmountIn.Amount,
		)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = []*big.Int{new(big.Int).Set(s.Info.Reserves[0]), new(big.Int).Set(s.Info.Reserves[1])}
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{BlockNumber: s.Info.BlockNumber}
}

// CalculateLimit reports each token's remaining ladder-side reserve. Callers
// that share inventory across multiple pools (e.g. one vault backing many
// pairs) should also call pool.RegisterUseSwapLimit for their DexType.
func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	tokens := s.GetTokens()
	return map[string]*big.Int{
		tokens[0]: s.Reserve0.ToBig(),
		tokens[1]: s.Reserve1.ToBig(),
	}
}
