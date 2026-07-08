package capricornpamm

import (
	"errors"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	staticExtra StaticExtra
	extra       Extra

	reserve0, reserve1 *uint256.Int
	decimals0          uint8
	decimals1          uint8

	feeBpsU *uint256.Int

	consumedIn  [2]uint256.Int
	consumedOut [2]uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, errors.New("capricorn-pamm: pool must have exactly 2 tokens")
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
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
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(s string, _ int) *big.Int { return bignum.NewBig(s) }),
			BlockNumber: ep.BlockNumber,
		}},
		staticExtra: staticExtra,
		extra:       extra,
		reserve0:    r0,
		reserve1:    r1,
		decimals0:   ep.Tokens[0].Decimals,
		decimals1:   ep.Tokens[1].Decimals,
		feeBpsU:     uint256.NewInt(extra.FeeBps),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.extra.Paused {
		return nil, ErrPaused
	}
	if s.extra.Unquoteable {
		return nil, ErrPoolUnavailable
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

	ladder := s.extra.Ladder0
	reserveOut := s.reserve1
	dir := 0
	if indexIn == 1 {
		ladder = s.extra.Ladder1
		reserveOut = s.reserve0
		dir = 1
	}

	// Marginal quote: evaluate the cumulative cost curve at consumedIn+amountIn
	// and subtract what has already been paid out on this direction.
	totalIn := new(uint256.Int).Add(&s.consumedIn[dir], amountIn)
	totalOut, err := QuoteAmountOut(ladder, totalIn)
	if err != nil {
		return nil, err
	}
	if totalOut.Cmp(&s.consumedOut[dir]) < 0 {
		// Should be unreachable: QuoteAmountOut is monotone non-decreasing and
		// consumedOut[dir] == QuoteAmountOut(consumedIn[dir]) by invariant.
		return nil, ErrNoQuote
	}
	amountOut := new(uint256.Int).Sub(totalOut, &s.consumedOut[dir])
	if amountOut.IsZero() {
		return nil, ErrNoQuote
	}

	if amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrPoolUnavailable
	}

	feeAmount := big256.MulDiv(amountIn, s.feeBpsU, feeDenominatorU256)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenIn, Amount: feeAmount.ToBig()},
		Gas:            defaultGas,
		SwapInfo: SwapInfo{
			Reserve0: new(uint256.Int).Set(s.reserve0),
			Reserve1: new(uint256.Int).Set(s.reserve1),
		},
	}, nil
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
	s.consumedIn[indexIn].Add(&s.consumedIn[indexIn], inU)
	s.consumedOut[indexIn].Add(&s.consumedOut[indexIn], outU)

	if indexIn == 0 {
		s.reserve0.Add(s.reserve0, inU)
		if s.reserve1.Cmp(outU) >= 0 {
			s.reserve1.Sub(s.reserve1, outU)
		} else {
			s.reserve1.Clear()
		}
	} else {
		s.reserve1.Add(s.reserve1, inU)
		if s.reserve0.Cmp(outU) >= 0 {
			s.reserve0.Sub(s.reserve0, outU)
		} else {
			s.reserve0.Clear()
		}
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.reserve0 = new(uint256.Int).Set(s.reserve0)
	cloned.reserve1 = new(uint256.Int).Set(s.reserve1)
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{BlockNumber: s.Info.BlockNumber}
}
