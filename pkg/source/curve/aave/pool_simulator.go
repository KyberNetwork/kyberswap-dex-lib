package aave

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type AavePool struct {
	pool.Pool
	Multipliers []*big.Int
	// extra fields
	InitialA            *big.Int
	FutureA             *big.Int
	InitialATime        int64
	FutureATime         int64
	AdminFee            *big.Int
	OffpegFeeMultiplier *big.Int
	gas                 Gas

	LpSupply *big.Int
}

type Gas struct {
	Exchange int64
}

func NewPoolSimulator(entityPool entity.Pool) (*AavePool, error) {
	var staticExtra curve.PoolAaveStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra curve.PoolAaveExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var numTokens = len(entityPool.Tokens)
	var tokens = make([]string, numTokens)
	var reserves = make([]*big.Int, numTokens)
	var multipliers = make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = staticExtra.UnderlyingTokens[i]
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		multipliers[i] = bignumber.NewBig10(staticExtra.PrecisionMultipliers[i])
	}

	lpSupply := bignumber.One
	if len(entityPool.Reserves) > numTokens {
		lpSupply = bignumber.NewBig10(entityPool.Reserves[numTokens])
	}

	return &AavePool{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    bignumber.NewBig10(extra.SwapFee),
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		Multipliers:         multipliers,
		InitialA:            bignumber.NewBig10(extra.InitialA),
		FutureA:             bignumber.NewBig10(extra.FutureA),
		InitialATime:        extra.InitialATime,
		FutureATime:         extra.FutureATime,
		AdminFee:            bignumber.NewBig10(extra.AdminFee),
		OffpegFeeMultiplier: bignumber.NewBig10(extra.OffpegFeeMultiplier),
		gas:                 DefaultGas,
		LpSupply:            lpSupply,
	}, nil
}

func (t *AavePool) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var tokenIndexFrom = t.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.GetTokenIndex(tokenOut)
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		amountOut, fee, err := GetDyUnderlying(
			t.Info.Reserves,
			t.Multipliers,
			t.FutureATime,
			t.FutureA,
			t.InitialATime,
			t.InitialA,
			t.Info.SwapFee,
			t.OffpegFeeMultiplier,
			tokenIndexFrom,
			tokenIndexTo,
			tokenAmountIn.Amount,
		)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		if err == nil && amountOut.Cmp(bignumber.ZeroBI) > 0 {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut,
				},
				Fee: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: fee,
				},
				Gas: t.gas.Exchange,
			}, nil
		}
	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenIndexFrom or tokenIndexTo is not correct: tokenIndexFrom: %v, tokenIndexTo: %v", tokenIndexFrom, tokenIndexTo)
}

func (t *AavePool) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = input.Amount
	var outputAmount = output.Amount
	// swap fee
	// output = output + output * swapFee * adminFee
	outputAmount = new(big.Int).Add(
		outputAmount,
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(new(big.Int).Mul(outputAmount, t.Info.SwapFee), FeeDenominator),
				t.AdminFee,
			),
			FeeDenominator,
		),
	)
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
		}
	}
}

func (t *AavePool) GetLpToken() string {
	return ""
}

func (t *AavePool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return curve.Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    true,
	}
}

func (t *AavePool) getDPrecision(xp []*big.Int, a *big.Int) (*big.Int, error) {
	var nCoins = len(xp)
	_xp := make([]*big.Int, nCoins)
	for i := 0; i < nCoins; i += 1 {
		_xp[i] = new(big.Int).Mul(xp[i], t.Multipliers[i])
	}
	return getD(_xp, a)
}

func (t *AavePool) AddLiquidity(amounts []*big.Int) (*big.Int, error) {
	var nCoins = len(amounts)
	var nCoinsBi = big.NewInt(int64(nCoins))
	var amp = _getAPrecise(t.FutureATime, t.FutureA, t.InitialATime, t.InitialA)
	var old_balances = make([]*big.Int, nCoins)
	for i := 0; i < nCoins; i += 1 {
		old_balances[i] = t.Info.Reserves[i]
	}
	D0, err := t.getDPrecision(old_balances, amp)
	if err != nil {
		return nil, err
	}
	var token_supply = t.LpSupply
	var new_balances = make([]*big.Int, nCoins)
	for i := 0; i < nCoins; i += 1 {
		new_balances[i] = new(big.Int).Add(old_balances[i], amounts[i])
	}
	D1, err := t.getDPrecision(new_balances, amp)
	if err != nil {
		return nil, err
	}
	if D1.Cmp(D0) <= 0 {
		return nil, ErrD1LowerThanD0
	}
	var mint_amount *big.Int
	if token_supply.Cmp(bignumber.ZeroBI) > 0 {
		ys := new(big.Int).Div(new(big.Int).Add(D0, D1), nCoinsBi)
		var _fee = new(big.Int).Div(new(big.Int).Mul(t.Info.SwapFee, nCoinsBi),
			new(big.Int).Mul(bignumber.Four, big.NewInt(int64(nCoins-1))))
		_feemul := t.OffpegFeeMultiplier
		for i := 0; i < nCoins; i += 1 {
			t.Info.Reserves[i] = new_balances[i] // cannot determine real amount transfered, so use this, close enough
			var ideal_balance = new(big.Int).Div(new(big.Int).Mul(D1, old_balances[i]), D0)
			var difference *big.Int
			if ideal_balance.Cmp(new_balances[i]) > 0 {
				difference = new(big.Int).Sub(ideal_balance, new_balances[i])
			} else {
				difference = new(big.Int).Sub(new_balances[i], ideal_balance)
			}
			xs := new(big.Int).Add(old_balances[i], new_balances[i])
			var fee = new(big.Int).Div(new(big.Int).Mul(_dynamicFee(xs, ys, _fee, _feemul), difference), FeeDenominator)
			new_balances[i] = new(big.Int).Sub(new_balances[i], fee)
		}
		D2, _ := t.getDPrecision(new_balances, amp)
		mint_amount = new(big.Int).Div(new(big.Int).Mul(token_supply, new(big.Int).Sub(D2, D0)), D0)
	} else {
		for i := 0; i < nCoins; i += 1 {
			t.Info.Reserves[i] = new_balances[i]
		}
		mint_amount = D1
	}
	t.LpSupply = new(big.Int).Add(t.LpSupply, mint_amount)
	return mint_amount, nil
}

func (t *AavePool) CalculateTokenAmount(amounts []*big.Int, deposit bool) (*big.Int, error) {
	return calculateTokenAmount(
		t.Info.Reserves,
		t.Multipliers,
		t.FutureATime, t.FutureA,
		t.InitialATime, t.InitialA,
		bignumber.ZeroBI, // withdraw fee not used in deposit case
		t.LpSupply,
		amounts,
		true,
	)
}

func (t *AavePool) CalculateWithdrawOneCoin(tokenAmount *big.Int, i int) (*big.Int, *big.Int, error) {
	return calculateWithdrawOneTokenDy(
		t.Info.Reserves,
		t.Multipliers,
		t.FutureATime, t.FutureA,
		t.InitialATime, t.InitialA,
		t.Info.SwapFee,
		t.LpSupply,
		i,
		tokenAmount,
	)
}

func (t *AavePool) RemoveLiquidityOneCoin(tokenAmount *big.Int, i int) (*big.Int, error) {
	var dy, dy_fee, err = t.CalculateWithdrawOneCoin(tokenAmount, i)
	if err != nil {
		return nil, err
	}
	t.Info.Reserves[i] = new(big.Int).Sub(
		t.Info.Reserves[i],
		new(big.Int).Add(dy, new(big.Int).Div(new(big.Int).Mul(dy_fee, t.AdminFee), FeeDenominator)),
	)
	t.LpSupply = new(big.Int).Sub(t.LpSupply, tokenAmount)
	return dy, nil
}

func (t *AavePool) GetDy(i int, j int, dx *big.Int) (*big.Int, *big.Int, error) {
	var nTokens = len(t.Info.Tokens)
	xp := make([]*big.Int, nTokens)
	for _i := 0; _i < nTokens; _i += 1 {
		xp[_i] = new(big.Int).Mul(t.Multipliers[_i], t.Info.Reserves[_i])
	}

	// x: uint256 = xp[i] + dx * precisions[i]
	var x = new(big.Int).Add(xp[i], new(big.Int).Mul(dx, t.Multipliers[i]))

	// y: uint256 = self.get_y(i, j, x, xp)
	var y, err = getY(t.FutureATime, t.FutureA, t.InitialATime, t.InitialA, i, j, x, xp)
	if err != nil {
		return nil, nil, err
	}

	// dy: uint256 = (xp[j] - y) / precisions[j]
	var dy = new(big.Int).Div(new(big.Int).Sub(xp[j], y), t.Multipliers[j])

	// _fee: uint256 = self._dynamic_fee(
	// 		(xp[i] + x) / 2, (xp[j] + y) / 2, self.fee, self.offpeg_fee_multiplier
	// ) * dy / FEE_DENOMINATOR
	var fee = _dynamicFee(
		new(big.Int).Div(new(big.Int).Add(xp[i], x), bignumber.Two),
		new(big.Int).Div(new(big.Int).Add(xp[j], y), bignumber.Two),
		t.Info.SwapFee,
		t.OffpegFeeMultiplier,
	)
	fee = new(big.Int).Div(new(big.Int).Mul(fee, dy), FeeDenominator)

	// return dy - _fee
	dy = new(big.Int).Sub(dy, fee)
	return dy, fee, nil
}

func (t *AavePool) GetVirtualPrice() (*big.Int, error) {
	var A = _getAPrecise(t.FutureATime, t.FutureA, t.InitialATime, t.InitialA)
	D, err := t.getDPrecision(t.Info.Reserves, A)
	if err != nil {
		return nil, err
	}
	if t.LpSupply.Cmp(bignumber.ZeroBI) == 0 {
		return nil, ErrDenominatorZero
	}
	return new(big.Int).Div(new(big.Int).Mul(D, Precision), t.LpSupply), nil
}
