package camelot

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		pool.Pool

		StableSwap           bool
		Token0FeePercent     *big.Int
		Token1FeePercent     *big.Int
		PrecisionMultiplier0 *big.Int
		PrecisionMultiplier1 *big.Int
		FeeDenominator       *big.Int

		Factory *Factory

		gas Gas
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	tokens := make([]string, 0, len(entityPool.Tokens))
	for _, token := range entityPool.Tokens {
		tokens = append(tokens, token.Address)
	}

	reserves := make([]*big.Int, 0, len(entityPool.Reserves))
	for _, reserve := range entityPool.Reserves {
		reserves = append(reserves, bignumber.NewBig10(reserve))
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		StableSwap:           extra.StableSwap,
		Token0FeePercent:     extra.Token0FeePercent,
		Token1FeePercent:     extra.Token1FeePercent,
		PrecisionMultiplier0: extra.PrecisionMultiplier0,
		PrecisionMultiplier1: extra.PrecisionMultiplier1,
		FeeDenominator:       staticExtra.FeeDenominator,
		Factory:              extra.Factory,
		gas:                  DefaultGas,
	}, nil
}

// CalcAmountOut simulates swapping through the pool and returns amountOut, fee and gas
// Swapping between token0 and token1 has the same logic but using different configs, such as Token0FeePercent or Token1FeePercent
// ,so I implemented two different functions to reduce if/else statements
// https://arbiscan.deth.net/address/0x84652bb2539513BAf36e225c930Fdd8eaa63CE27
func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	if strings.EqualFold(tokenAmountIn.Token, p.Info.Tokens[0]) {
		return p._swap0To1(tokenAmountIn, tokenOut)
	}

	return p._swap1To0(tokenAmountIn, tokenOut)
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if strings.EqualFold(params.TokenAmountIn.Token, p.Info.Tokens[0]) {
		p.Info.Reserves[0] = new(big.Int).Add(p.Info.Reserves[0], params.TokenAmountOut.Amount)
		p.Info.Reserves[1] = new(big.Int).Sub(new(big.Int).Add(p.Info.Reserves[1], params.TokenAmountIn.Amount), params.Fee.Amount)
	} else {
		p.Info.Reserves[0] = new(big.Int).Sub(new(big.Int).Add(p.Info.Reserves[0], params.TokenAmountIn.Amount), params.Fee.Amount)
		p.Info.Reserves[1] = new(big.Int).Add(p.Info.Reserves[1], params.TokenAmountOut.Amount)
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, _ string) interface{} {
	var swapFee uint32
	if strings.EqualFold(tokenIn, p.Info.Tokens[0]) {
		swapFee = uint32(p.Token0FeePercent.Uint64())
	} else {
		swapFee = uint32(p.Token1FeePercent.Uint64())
	}

	return Meta{
		SwapFee:      swapFee,
		FeePrecision: uint32(p.FeeDenominator.Int64()),
	}
}

func (p *PoolSimulator) _swap0To1(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	amountOut := p.getAmountOut(tokenAmountIn.Amount, tokenAmountIn.Token)

	if amountOut.Cmp(bignumber.ZeroBI) <= 0 {
		return &pool.CalcAmountOutResult{}, ErrInsufficientOutputAmount
	}

	if amountOut.Cmp(p.Info.Reserves[1]) >= 0 {
		return &pool.CalcAmountOutResult{}, ErrInsufficientLiquidity
	}

	fee := new(big.Int).Div(
		new(big.Int).Mul(tokenAmountIn.Amount, p.Token0FeePercent),
		p.FeeDenominator,
	)

	if p.StableSwap && p.Factory != nil && len(p.Factory.FeeTo) > 0 {
		ownerFeeShare := new(big.Int).Mul(p.FeeDenominator, p.Factory.OwnerFeeShare)
		ownerFee := new(big.Int).Div(
			new(big.Int).Mul(new(big.Int).Mul(tokenAmountIn.Amount, ownerFeeShare), p.Token0FeePercent),
			new(big.Int).Exp(p.FeeDenominator, bignumber.Three, nil),
		)

		fee = new(big.Int).Sub(fee, ownerFee)
	}

	balance0Adjusted := new(big.Int).Add(new(big.Int).Sub(p.Info.Reserves[0], fee), tokenAmountIn.Amount)
	balance1Adjusted := new(big.Int).Sub(p.Info.Reserves[1], amountOut)

	kBefore := p._k(p.Info.Reserves[0], p.Info.Reserves[1])
	kAfter := p._k(balance0Adjusted, balance1Adjusted)

	if kAfter.Cmp(kBefore) < 0 {
		return &pool.CalcAmountOutResult{}, ErrInvalidK
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: fee,
		},
		Gas: p.gas.Swap,
	}, nil
}

func (p *PoolSimulator) _swap1To0(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	amountOut := p.getAmountOut(tokenAmountIn.Amount, tokenAmountIn.Token)

	if amountOut.Cmp(bignumber.ZeroBI) <= 0 {
		return &pool.CalcAmountOutResult{}, ErrInsufficientOutputAmount
	}

	if amountOut.Cmp(p.Info.Reserves[0]) >= 0 {
		return &pool.CalcAmountOutResult{}, ErrInsufficientLiquidity
	}

	fee := new(big.Int).Div(
		new(big.Int).Mul(tokenAmountIn.Amount, p.Token1FeePercent),
		p.FeeDenominator,
	)

	if p.StableSwap && p.Factory != nil && len(p.Factory.FeeTo) > 0 {
		ownerFeeShare := new(big.Int).Mul(p.FeeDenominator, p.Factory.OwnerFeeShare)
		ownerFee := new(big.Int).Div(
			new(big.Int).Mul(new(big.Int).Mul(tokenAmountIn.Amount, ownerFeeShare), p.Token1FeePercent),
			new(big.Int).Exp(p.FeeDenominator, bignumber.Three, nil),
		)

		fee = new(big.Int).Sub(fee, ownerFee)
	}

	balance0Adjusted := new(big.Int).Sub(p.Info.Reserves[0], amountOut)
	balance1Adjusted := new(big.Int).Add(new(big.Int).Sub(p.Info.Reserves[1], fee), tokenAmountIn.Amount)

	kBefore := p._k(p.Info.Reserves[0], p.Info.Reserves[1])
	kAfter := p._k(balance0Adjusted, balance1Adjusted)

	if kAfter.Cmp(kBefore) < 0 {
		return &pool.CalcAmountOutResult{}, ErrInvalidK
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: fee,
		},
		Gas: p.gas.Swap,
	}, nil
}

func (p *PoolSimulator) getAmountOut(amountIn *big.Int, tokenIn string) *big.Int {
	var feePercent *big.Int

	if strings.EqualFold(tokenIn, p.Info.Tokens[0]) {
		feePercent = p.Token0FeePercent
	} else {
		feePercent = p.Token1FeePercent
	}

	return p._getAmountOut(amountIn, tokenIn, p.Info.Reserves[0], p.Info.Reserves[1], feePercent)
}

func (p *PoolSimulator) _getAmountOut(
	amountIn *big.Int,
	tokenIn string,
	_reserve0 *big.Int,
	_reserve1 *big.Int,
	feePercent *big.Int,
) *big.Int {
	if p.StableSwap {
		amountIn = new(big.Int).Sub(
			amountIn,
			new(big.Int).Div(new(big.Int).Mul(amountIn, feePercent), p.FeeDenominator),
		)

		xy := p._k(_reserve0, _reserve1)
		_reserve0 = new(big.Int).Div(new(big.Int).Mul(_reserve0, bignumber.BONE), p.PrecisionMultiplier0)
		_reserve1 = new(big.Int).Div(new(big.Int).Mul(_reserve1, bignumber.BONE), p.PrecisionMultiplier1)

		var (
			reserveA       *big.Int
			reserveB       *big.Int
			isSwapFrom0To1 = strings.EqualFold(tokenIn, p.Info.Tokens[0])
		)

		if isSwapFrom0To1 {
			reserveA = _reserve0
			reserveB = _reserve1
			amountIn = new(big.Int).Div(new(big.Int).Mul(amountIn, bignumber.BONE), p.PrecisionMultiplier0)
		} else {
			reserveA = _reserve1
			reserveB = _reserve0
			amountIn = new(big.Int).Div(new(big.Int).Mul(amountIn, bignumber.BONE), p.PrecisionMultiplier1)
		}

		y := new(big.Int).Sub(
			reserveB,
			p._getY(
				new(big.Int).Add(amountIn, reserveA),
				xy,
				reserveB,
			),
		)

		if isSwapFrom0To1 {
			return new(big.Int).Div(new(big.Int).Mul(y, p.PrecisionMultiplier1), bignumber.BONE)
		} else {
			return new(big.Int).Div(new(big.Int).Mul(y, p.PrecisionMultiplier0), bignumber.BONE)
		}
	}

	var (
		reserveA *big.Int
		reserveB *big.Int
	)

	if strings.EqualFold(tokenIn, p.Info.Tokens[0]) {
		reserveA = _reserve0
		reserveB = _reserve1
	} else {
		reserveA = _reserve1
		reserveB = _reserve0
	}

	amountIn = new(big.Int).Mul(amountIn, new(big.Int).Sub(p.FeeDenominator, feePercent))
	return new(big.Int).Div(
		new(big.Int).Mul(amountIn, reserveB),
		new(big.Int).Add(
			new(big.Int).Mul(reserveA, p.FeeDenominator),
			amountIn,
		),
	)
}

func (p *PoolSimulator) _k(balance0 *big.Int, balance1 *big.Int) *big.Int {
	if p.StableSwap {
		_x := new(big.Int).Div(new(big.Int).Mul(balance0, bignumber.BONE), p.PrecisionMultiplier0)
		_y := new(big.Int).Div(new(big.Int).Mul(balance1, bignumber.BONE), p.PrecisionMultiplier1)
		_a := new(big.Int).Div(new(big.Int).Mul(_x, _y), bignumber.BONE)
		_b := new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(_x, _x), bignumber.BONE),
			new(big.Int).Div(new(big.Int).Mul(_y, _y), bignumber.BONE),
		)

		return new(big.Int).Div(new(big.Int).Mul(_a, _b), bignumber.BONE)
	}

	return new(big.Int).Mul(balance0, balance1)
}

func (p *PoolSimulator) _getY(x0 *big.Int, xy *big.Int, y *big.Int) *big.Int {
	for i := 0; i < 255; i++ {
		yPrev := y
		k := _f(x0, y)

		if k.Cmp(xy) < 0 {
			// dy = (xy - k) * 1e18 / _d(x0, y)
			dy := new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(xy, k), bignumber.BONE), _d(x0, y))
			y = new(big.Int).Add(y, dy)
		} else {
			// dy = (k - xy) * 1e18 / _d(x0, y);
			dy := new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(k, xy), bignumber.BONE), _d(x0, y))
			y = new(big.Int).Sub(y, dy)
		}

		if y.Cmp(yPrev) > 0 {
			if new(big.Int).Sub(y, yPrev).Cmp(bignumber.One) <= 0 {
				return y
			}
		} else {
			if new(big.Int).Sub(yPrev, y).Cmp(bignumber.One) <= 0 {
				return y
			}
		}
	}

	return y
}
