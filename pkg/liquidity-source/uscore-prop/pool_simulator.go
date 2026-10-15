package uscoreprop

import (
	"math"
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	ladders  [2][]Point
	slopes   [2][][2]uint256.Int
	reserves [2]uint256.Int
	// consumedIn/Out accumulate swaps per direction so a split route walks the same curve.
	consumedIn, consumedOut [2]uint256.Int
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	return NewPoolSimulatorWith(params.EntityPool, lo.Ternary(params.Opts.StaleCheck, MaxAge, math.MaxInt64))
}

func NewPoolSimulatorWith(ep entity.Pool, maxAge time.Duration) (*PoolSimulator, error) {
	if time.Since(time.Unix(ep.Timestamp, 0)) > maxAge {
		return nil, ladder.ErrStale
	}
	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, ErrInvalidState
	}
	s := &PoolSimulator{Pool: pool.Pool{Info: pool.PoolInfo{
		Address:     ep.Address,
		Exchange:    ep.Exchange,
		Type:        ep.Type,
		Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
		Reserves:    make([]*big.Int, 2),
		BlockNumber: ep.BlockNumber,
	}}}
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	s.ladders = extra.Ladders
	for i, r := range ep.Reserves {
		if err := s.reserves[i].SetFromDecimal(r); err != nil {
			return nil, err
		}
		s.Info.Reserves[i] = s.reserves[i].ToBig()
	}
	for dir, points := range s.ladders {
		for i := range points {
			if p := &points[i]; p[0].IsZero() || p[1].IsZero() || p[0].Gt(maxPoint) || p[1].Gt(maxPoint) ||
				i > 0 && (!p[0].Gt(&points[i-1][0]) || p[1].Lt(&points[i-1][1])) {
				return nil, ErrInvalidState
			}
		}
		s.slopes[dir] = segmentSlopes(points)
	}
	return s, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = []*big.Int{new(big.Int).Set(s.Info.Reserves[0]), new(big.Int).Set(s.Info.Reserves[1])}
	return &cloned
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	in, out := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if in < 0 || out < 0 || in == out {
		return nil, ladder.ErrInvalidToken
	}
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ladder.ErrZeroAmountIn
	}
	var total, amountOut uint256.Int
	if total.SetFromBig(params.TokenAmountIn.Amount) {
		return nil, ladder.ErrAmountInTooLarge
	}
	if _, overflow := total.AddOverflow(&total, &s.consumedIn[in]); overflow {
		return nil, ladder.ErrAmountInTooLarge
	}
	if err := interpolate(s.ladders[in], s.slopes[in], &total, &amountOut); err != nil {
		return nil, err
	}
	if !amountOut.Gt(&s.consumedOut[in]) {
		return nil, ladder.ErrNoQuote
	}
	if amountOut.Sub(&amountOut, &s.consumedOut[in]).Gt(&s.reserves[out]) {
		return nil, ladder.ErrInsufficientLiquidity
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: bignum.ZeroBI},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	in, out := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if in < 0 || out < 0 || in == out || params.TokenAmountIn.Amount == nil || params.TokenAmountOut.Amount == nil ||
		params.TokenAmountIn.Amount.Sign() <= 0 || params.TokenAmountOut.Amount.Sign() <= 0 {
		return
	}
	var amountIn, amountOut, reserveIn, consumedIn uint256.Int
	if amountIn.SetFromBig(params.TokenAmountIn.Amount) || amountOut.SetFromBig(params.TokenAmountOut.Amount) ||
		amountOut.Gt(&s.reserves[out]) {
		return
	}
	if _, overflow := reserveIn.AddOverflow(&s.reserves[in], &amountIn); overflow {
		return
	}
	if _, overflow := consumedIn.AddOverflow(&s.consumedIn[in], &amountIn); overflow {
		return
	}
	s.consumedIn[in] = consumedIn
	s.consumedOut[in].Add(&s.consumedOut[in], &amountOut)
	s.reserves[in] = reserveIn
	s.reserves[out].Sub(&s.reserves[out], &amountOut)
	s.Info.Reserves[in], s.Info.Reserves[out] = s.reserves[in].ToBig(), s.reserves[out].ToBig()
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return ladder.PoolMeta{BlockNumber: s.Info.BlockNumber}
}
