package camelot

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type (
	Pool struct {
		poolpkg.Pool

		StableSwap           bool
		Token0FeePercent     *big.Int
		Token1FeePercent     *big.Int
		PrecisionMultiplier0 *big.Int
		PrecisionMultiplier1 *big.Int
		FeeDenominator       *big.Int

		Factory *Factory

		gas Gas
	}

	Factory struct {
		OwnerFeeShare *big.Int `json:"ownerFeeShare"`
		FeeTo         string   `json:"feeTo"`
	}

	Gas struct {
		Swap int64
	}

	Extra struct {
		StableSwap           bool     `json:"stableSwap"`
		Token0FeePercent     *big.Int `json:"token0FeePercent"`
		Token1FeePercent     *big.Int `json:"token1FeePercent"`
		PrecisionMultiplier0 *big.Int `json:"precisionMultiplier0"`
		PrecisionMultiplier1 *big.Int `json:"precisionMultiplier1"`

		Factory *Factory `json:"factory"`
	}

	StaticExtra struct {
		FeeDenominator *big.Int `json:"feeDenominator"`
	}

	Meta struct {
		SwapFee      uint32 `json:"swapFee"`
		FeePrecision uint32 `json:"feePrecision"`
	}
)

func NewPool(entityPool entity.Pool) (*Pool, error) {
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
		reserves = append(reserves, utils.NewBig10(reserve))
	}

	return &Pool{
		Pool: poolpkg.Pool{
			Info: poolpkg.PoolInfo{
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
func (p *Pool) CalcAmountOut(
	tokenAmountIn poolpkg.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	if strings.EqualFold(tokenAmountIn.Token, p.Info.Tokens[0]) {
		return p._swap0To1(tokenAmountIn, tokenOut)
	}

	return p._swap1To0(tokenAmountIn, tokenOut)
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	if strings.EqualFold(params.TokenAmountIn.Token, p.Info.Tokens[0]) {
		p.Info.Reserves[0] = new(big.Int).Sub(new(big.Int).Add(p.Info.Reserves[0], params.TokenAmountIn.Amount), params.Fee.Amount)
		p.Info.Reserves[1] = new(big.Int).Add(p.Info.Reserves[1], params.TokenAmountOut.Amount)
	} else {
		p.Info.Reserves[0] = new(big.Int).Add(p.Info.Reserves[0], params.TokenAmountOut.Amount)
		p.Info.Reserves[1] = new(big.Int).Sub(new(big.Int).Add(p.Info.Reserves[1], params.TokenAmountIn.Amount), params.Fee.Amount)
	}
}

func (p *Pool) GetLpToken() string {
	return ""
}

func (p *Pool) GetMidPrice(tokenIn string, _ string, base *big.Int) *big.Int {
	return p.getAmountOut(base, tokenIn)
}

func (p *Pool) CalcExactQuote(tokenIn string, _ string, base *big.Int) *big.Int {
	return p.getAmountOut(base, tokenIn)
}

func (p *Pool) GetMetaInfo(tokenIn string, _ string) interface{} {
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

func (p *Pool) _swap0To1(
	tokenAmountIn poolpkg.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	amountOut := p.getAmountOut(tokenAmountIn.Amount, tokenAmountIn.Token)

	if amountOut.Cmp(constant.Zero) <= 0 {
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
			new(big.Int).Exp(p.FeeDenominator, constant.Three, nil),
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
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &poolpkg.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: fee,
		},
		Gas: p.gas.Swap,
	}, nil
}

func (p *Pool) _swap1To0(
	tokenAmountIn poolpkg.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	amountOut := p.getAmountOut(tokenAmountIn.Amount, tokenAmountIn.Token)

	if amountOut.Cmp(constant.Zero) <= 0 {
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
			new(big.Int).Exp(p.FeeDenominator, constant.Three, nil),
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
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &poolpkg.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: fee,
		},
		Gas: p.gas.Swap,
	}, nil
}

func (p *Pool) getAmountOut(amountIn *big.Int, tokenIn string) *big.Int {
	var feePercent *big.Int

	if strings.EqualFold(tokenIn, p.Info.Tokens[0]) {
		feePercent = p.Token0FeePercent
	} else {
		feePercent = p.Token1FeePercent
	}

	return p._getAmountOut(amountIn, tokenIn, p.Info.Reserves[0], p.Info.Reserves[1], feePercent)
}

func (p *Pool) _getAmountOut(
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
		_reserve0 = new(big.Int).Div(new(big.Int).Mul(_reserve0, constant.BONE), p.PrecisionMultiplier0)
		_reserve1 = new(big.Int).Div(new(big.Int).Mul(_reserve1, constant.BONE), p.PrecisionMultiplier1)

		var (
			reserveA       = _reserve0
			reserveB       = _reserve1
			isSwapFrom0To1 = strings.EqualFold(tokenIn, p.Info.Tokens[0])
		)

		if isSwapFrom0To1 {
			reserveA = _reserve0
			reserveB = _reserve1
			amountIn = new(big.Int).Div(new(big.Int).Mul(amountIn, constant.BONE), p.PrecisionMultiplier0)
		} else {
			reserveA = _reserve1
			reserveB = _reserve0
			amountIn = new(big.Int).Div(new(big.Int).Mul(amountIn, constant.BONE), p.PrecisionMultiplier1)
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
			return new(big.Int).Div(new(big.Int).Mul(y, p.PrecisionMultiplier1), constant.BONE)
		} else {
			return new(big.Int).Div(new(big.Int).Mul(y, p.PrecisionMultiplier0), constant.BONE)
		}
	}

	var (
		reserveA = _reserve0
		reserveB = _reserve1
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

func (p *Pool) _k(balance0 *big.Int, balance1 *big.Int) *big.Int {
	if p.StableSwap {
		_x := new(big.Int).Div(new(big.Int).Mul(balance0, constant.BONE), p.PrecisionMultiplier0)
		_y := new(big.Int).Div(new(big.Int).Mul(balance1, constant.BONE), p.PrecisionMultiplier1)
		_a := new(big.Int).Div(new(big.Int).Mul(_x, _y), constant.BONE)
		_b := new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(_x, _x), constant.BONE),
			new(big.Int).Div(new(big.Int).Mul(_y, _y), constant.BONE),
		)

		return new(big.Int).Div(new(big.Int).Mul(_a, _b), constant.BONE)
	}

	return new(big.Int).Mul(balance0, balance1)
}

func (p *Pool) _getY(x0 *big.Int, xy *big.Int, y *big.Int) *big.Int {
	for i := 0; i < 255; i++ {
		yPrev := y
		k := _f(x0, y)

		if k.Cmp(xy) < 0 {
			// dy = (xy - k) * 1e18 / _d(x0, y)
			dy := new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(xy, k), constant.BONE), _d(x0, y))
			y = new(big.Int).Add(y, dy)
		} else {
			// dy = (k - xy) * 1e18 / _d(x0, y);
			dy := new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(k, xy), constant.BONE), _d(x0, y))
			y = new(big.Int).Sub(y, dy)
		}

		if y.Cmp(yPrev) > 0 {
			if new(big.Int).Sub(y, yPrev).Cmp(constant.One) <= 0 {
				return y
			}
		} else {
			if new(big.Int).Sub(yPrev, y).Cmp(constant.One) <= 0 {
				return y
			}
		}
	}

	return y
}
