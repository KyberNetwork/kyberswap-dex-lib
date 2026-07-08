package poe

import (
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	reserveX *uint256.Int
	reserveY *uint256.Int
	price    *uint256.Int
	feeHbps  *uint256.Int
	alpha    *uint256.Int
	expiry   uint64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     ep.Address,
				Exchange:    ep.Exchange,
				Type:        ep.Type,
				Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
				Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
				BlockNumber: ep.BlockNumber,
			},
		},
		reserveX: uint256.MustFromDecimal(ep.Reserves[0]),
		reserveY: uint256.MustFromDecimal(ep.Reserves[1]),
		price:    extra.Price,
		feeHbps:  extra.FeeHbps,
		alpha:    extra.Alpha,
		expiry:   extra.Expiry,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if err := s.validate(); err != nil {
		return nil, err
	}

	isXtoY := indexIn == 0

	q, err := getQuote(u256.ToBig(s.reserveX), u256.ToBig(s.reserveY), tokenAmountIn.Amount, isXtoY,
		u256.ToBig(s.price), u256.ToBig(s.feeHbps), u256.ToBig(s.alpha))
	if err != nil {
		return nil, err
	}

	remaining := new(big.Int).Sub(tokenAmountIn.Amount, q.actualIn)

	fee := q.feeOut
	if !isXtoY {
		fee = q.feeIn
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: q.amountOut},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[1], Amount: fee},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: remaining,
		},
		Gas: defaultGas,
	}, nil
}

// CalcAmountIn estimates the input required for a desired output. The
// on-chain pool has no exact-out entrypoint (getQuote/swap only take
// amountIn), so this binary-searches getQuote — which is monotonic in
// amountIn — for the smallest input whose quoted output meets the target.
func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut := param.TokenAmountOut
	tokenIn := param.TokenIn

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if tokenAmountOut.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}

	if err := s.validate(); err != nil {
		return nil, err
	}

	isXtoY := indexIn == 0
	realReserveOut := u256.ToBig(lo.Ternary(isXtoY, s.reserveY, s.reserveX))
	if tokenAmountOut.Amount.Cmp(realReserveOut) >= 0 {
		return nil, ErrInsufficientLiquidity
	}

	reserveX, reserveY := u256.ToBig(s.reserveX), u256.ToBig(s.reserveY)
	price, fee, alpha := u256.ToBig(s.price), u256.ToBig(s.feeHbps), u256.ToBig(s.alpha)

	quoteOut := func(amountIn *big.Int) *big.Int {
		q, err := getQuote(reserveX, reserveY, amountIn, isXtoY, price, fee, alpha)
		if err != nil {
			return new(big.Int)
		}
		return q.amountOut
	}

	loAmt, hiAmt := new(big.Int), big.NewInt(1)
	for quoteOut(hiAmt).Cmp(tokenAmountOut.Amount) < 0 {
		loAmt.Set(hiAmt)
		hiAmt = new(big.Int).Lsh(hiAmt, 1)
		if hiAmt.Cmp(realReserveOut) >= 0 {
			hiAmt = new(big.Int).Set(realReserveOut)
			break
		}
	}

	if quoteOut(hiAmt).Cmp(tokenAmountOut.Amount) < 0 {
		return nil, ErrInsufficientLiquidity
	}

	for i := 0; i < 256; i++ {
		diff := new(big.Int).Sub(hiAmt, loAmt)
		if diff.Cmp(big.NewInt(1)) <= 0 {
			break
		}
		mid := new(big.Int).Add(loAmt, diff)
		mid.Rsh(mid, 1)

		if quoteOut(mid).Cmp(tokenAmountOut.Amount) >= 0 {
			hiAmt = mid
		} else {
			loAmt = mid
		}
	}

	q, err := getQuote(reserveX, reserveY, hiAmt, isXtoY, price, fee, alpha)
	if err != nil {
		return nil, err
	}

	feeAmt := q.feeOut
	if !isXtoY {
		feeAmt = q.feeIn
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: q.actualIn},
		Fee:           &pool.TokenAmount{Token: s.Info.Tokens[1], Amount: feeAmt},
		Gas:           defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)

	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)
	amountOut := uint256.MustFromBig(params.TokenAmountOut.Amount)

	isXtoY := indexIn == 0
	if isXtoY {
		s.reserveX.Add(s.reserveX, amountIn)
		s.reserveY.Sub(s.reserveY, amountOut)
	} else {
		s.reserveY.Add(s.reserveY, amountIn)
		s.reserveX.Sub(s.reserveX, amountOut)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.reserveX = new(uint256.Int).Set(s.reserveX)
	cloned.reserveY = new(uint256.Int).Set(s.reserveY)

	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return PoolMeta{
		BlockNumber: s.Info.BlockNumber,
		IsXtoY:      s.GetTokenIndex(tokenIn) == 0,
	}
}

func (s *PoolSimulator) validate() error {
	if s.alpha.Cmp(uBps) <= 0 {
		return ErrInvalidAlpha
	}

	if s.price.IsZero() {
		return ErrZeroReserve
	}

	now := uint64(time.Now().Unix())
	if now > s.expiry {
		return ErrExpiredOracle
	}

	return nil
}
