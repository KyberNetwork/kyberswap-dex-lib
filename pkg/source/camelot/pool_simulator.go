package camelot

import (
	"math/big"
	"slices"
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

var _ = pool.RegisterFactory0(DexTypeCamelot, NewPoolSimulator)

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
				Address:  strings.ToLower(entityPool.Address),
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: reserves,
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

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Info.Reserves = slices.Clone(p.Info.Reserves)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if strings.EqualFold(params.TokenAmountIn.Token, p.Info.Tokens[0]) {
		newAmtIn := new(big.Int).Add(p.Info.Reserves[0], params.TokenAmountIn.Amount)
		p.Info.Reserves[0] = newAmtIn.Sub(newAmtIn, params.Fee.Amount)
		p.Info.Reserves[1] = new(big.Int).Sub(p.Info.Reserves[1], params.TokenAmountOut.Amount)
	} else {
		p.Info.Reserves[0] = new(big.Int).Sub(p.Info.Reserves[0], params.TokenAmountOut.Amount)
		newAmtIn := new(big.Int).Add(p.Info.Reserves[1], params.TokenAmountIn.Amount)
		p.Info.Reserves[1] = newAmtIn.Sub(newAmtIn, params.Fee.Amount)
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

	if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutputAmount
	} else if amountOut.Cmp(p.Info.Reserves[1]) >= 0 {
		return nil, ErrInsufficientLiquidity
	}

	var fee big.Int
	fee.Div(
		fee.Mul(tokenAmountIn.Amount, p.Token0FeePercent),
		p.FeeDenominator,
	)

	if p.StableSwap && p.Factory != nil && p.Factory.FeeTo != ZeroAddress {
		var ownerFeeShare, denom big.Int
		ownerFeeShare.Mul(p.FeeDenominator, p.Factory.OwnerFeeShare)
		ownerFee := ownerFeeShare.Div(
			ownerFeeShare.Mul(ownerFeeShare.Mul(tokenAmountIn.Amount, &ownerFeeShare), p.Token0FeePercent),
			denom.Exp(p.FeeDenominator, bignumber.Three, nil),
		)

		fee.Sub(&fee, ownerFee)
	}

	var balance0Adjusted, balance1Adjusted big.Int
	balance0Adjusted.Add(balance0Adjusted.Sub(p.Info.Reserves[0], &fee), tokenAmountIn.Amount)
	balance1Adjusted.Sub(p.Info.Reserves[1], amountOut)

	kBefore := p._k(p.Info.Reserves[0], p.Info.Reserves[1])
	kAfter := p._k(&balance0Adjusted, &balance1Adjusted)
	if kAfter.Cmp(kBefore) < 0 {
		return nil, ErrInvalidK
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: &fee,
		},
		Gas: p.gas.Swap,
	}, nil
}

func (p *PoolSimulator) _swap1To0(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	amountOut := p.getAmountOut(tokenAmountIn.Amount, tokenAmountIn.Token)

	if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutputAmount
	} else if amountOut.Cmp(p.Info.Reserves[0]) >= 0 {
		return nil, ErrInsufficientLiquidity
	}

	var fee big.Int
	fee.Div(
		fee.Mul(tokenAmountIn.Amount, p.Token1FeePercent),
		p.FeeDenominator,
	)

	if p.StableSwap && p.Factory != nil && p.Factory.FeeTo != ZeroAddress {
		var ownerFeeShare, denom big.Int
		ownerFeeShare.Mul(p.FeeDenominator, p.Factory.OwnerFeeShare)
		ownerFee := ownerFeeShare.Div(
			ownerFeeShare.Mul(ownerFeeShare.Mul(tokenAmountIn.Amount, &ownerFeeShare), p.Token1FeePercent),
			denom.Exp(p.FeeDenominator, bignumber.Three, nil),
		)

		fee.Sub(&fee, ownerFee)
	}

	var balance0Adjusted, balance1Adjusted big.Int
	balance0Adjusted.Sub(p.Info.Reserves[0], amountOut)
	balance1Adjusted.Add(balance1Adjusted.Sub(p.Info.Reserves[1], &fee), tokenAmountIn.Amount)

	kBefore := p._k(p.Info.Reserves[0], p.Info.Reserves[1])
	kAfter := p._k(&balance0Adjusted, &balance1Adjusted)
	if kAfter.Cmp(kBefore) < 0 {
		return nil, ErrInvalidK
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: &fee,
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
	var tmp, tmp1, tmp2 big.Int
	if p.StableSwap {
		amountIn = tmp.Sub(
			amountIn,
			tmp.Div(tmp.Mul(amountIn, feePercent), p.FeeDenominator),
		)

		xy := p._k(_reserve0, _reserve1)
		_reserve0 = tmp1.Div(tmp1.Mul(_reserve0, bignumber.BONE), p.PrecisionMultiplier0)
		_reserve1 = tmp2.Div(tmp2.Mul(_reserve1, bignumber.BONE), p.PrecisionMultiplier1)

		var (
			reserveA       *big.Int
			reserveB       *big.Int
			isSwapFrom0To1 = strings.EqualFold(tokenIn, p.Info.Tokens[0])
		)

		if isSwapFrom0To1 {
			reserveA = _reserve0
			reserveB = _reserve1
			amountIn = amountIn.Div(amountIn.Mul(amountIn, bignumber.BONE), p.PrecisionMultiplier0)
		} else {
			reserveA = _reserve1
			reserveB = _reserve0
			amountIn = amountIn.Div(amountIn.Mul(amountIn, bignumber.BONE), p.PrecisionMultiplier1)
		}

		y := tmp.Sub(
			reserveB,
			p._getY(
				amountIn.Add(amountIn, reserveA),
				xy,
				reserveB,
			),
		)

		if isSwapFrom0To1 {
			return tmp.Div(tmp.Mul(y, p.PrecisionMultiplier1), bignumber.BONE)
		} else {
			return tmp.Div(tmp.Mul(y, p.PrecisionMultiplier0), bignumber.BONE)
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

	amountIn = tmp.Mul(amountIn, tmp.Sub(p.FeeDenominator, feePercent))
	return tmp.Div(
		tmp1.Mul(amountIn, reserveB),
		tmp2.Add(
			tmp2.Mul(reserveA, p.FeeDenominator),
			amountIn,
		),
	)
}

func (p *PoolSimulator) _k(balance0 *big.Int, balance1 *big.Int) *big.Int {
	if p.StableSwap {
		var _x, _y, _a big.Int
		_x.Div(_x.Mul(balance0, bignumber.BONE), p.PrecisionMultiplier0)
		_y.Div(_y.Mul(balance1, bignumber.BONE), p.PrecisionMultiplier1)
		_a.Div(_a.Mul(&_x, &_y), bignumber.BONE)
		_b := _x.Add(
			_x.Div(_x.Mul(&_x, &_x), bignumber.BONE),
			_y.Div(_y.Mul(&_y, &_y), bignumber.BONE),
		)

		return _a.Div(_a.Mul(&_a, _b), bignumber.BONE)
	}

	return new(big.Int).Mul(balance0, balance1)
}

func (p *PoolSimulator) _getY(x0 *big.Int, xy *big.Int, y *big.Int) *big.Int {
	var yPrev, _y, dy big.Int
	_y.Set(y)
	yPrev.Set(&_y)
	for range 255 {
		k := _f(x0, &_y)

		// if k < xy { dy = (xy - k) * 1e18 / _d(x0, y); y+=dy; }
		// else      { dy = (k - xy) * 1e18 / _d(x0, y); y-=dy; }
		kGtXy := dy.Sub(xy, k).Sign() > 0
		dy.Div(dy.Mul(dy.Abs(&dy), bignumber.BONE), _d(x0, &_y))
		if kGtXy {
			_y.Add(&_y, &dy)
		} else {
			_y.Sub(&_y, &dy)
		}

		if yPrev.Abs(yPrev.Sub(&yPrev, &_y)).Cmp(bignumber.One) <= 0 {
			return &_y
		}
		yPrev.Set(&_y)
	}

	return &_y
}
