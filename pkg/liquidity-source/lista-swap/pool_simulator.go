package listaswap

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
)

var (
	ErrInvalidValue    = errors.New("invalid value")
	ErrPriceDiffToken0 = errors.New("price difference for token0 exceeds threshold")
	ErrPriceDiffToken1 = errors.New("price difference for token1 exceeds threshold")
)

type PoolSimulator struct {
	pool.Pool
	baseSim            *base.PoolSimulator
	decimals           []uint8
	isNativeCoins      []bool
	oraclePrices       [2]*big.Int
	priceDiffThreshold [2]*big.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	curveBaseSimulator, err := base.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	decimals := make([]uint8, len(entityPool.Tokens))
	for i, token := range entityPool.Tokens {
		decimals[i] = token.Decimals
	}

	return &PoolSimulator{
		Pool:               curveBaseSimulator.Pool,
		baseSim:            curveBaseSimulator,
		decimals:           decimals,
		isNativeCoins:      staticExtra.IsNativeCoins,
		oraclePrices:       extra.OraclePrices,
		priceDiffThreshold: extra.PriceDiffThreshold,
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	res, err := t.baseSim.CalcAmountOut(param)
	if err != nil {
		return nil, err
	}

	// Now we check if the oracle prices are still satisfied
	// after the swap (checkPriceDiff() in the contract).
	// Prepare to update balance (without actually modifying the simulator state).
	updatedBalances := t.prepareUpdateBalance(param, res)

	if err := t.checkPriceDiff(updatedBalances); err != nil {
		return nil, err
	}

	return res, nil
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	t.baseSim.UpdateBalance(params)
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	var fromId = t.baseSim.GetTokenIndex(tokenIn)
	var toId = t.baseSim.GetTokenIndex(tokenOut)

	meta := Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
	}
	if len(t.isNativeCoins) == len(t.Info.Tokens) {
		meta.TokenInIsNative = t.isNativeCoins[fromId]
		meta.TokenOutIsNative = t.isNativeCoins[toId]
	}

	return meta
}

// prepareUpdateBalance has the same logic as UpdateBalance,
// but returns the updated balances value instead of modifying the simulator state.
func (t *PoolSimulator) prepareUpdateBalance(
	param pool.CalcAmountOutParams,
	res *pool.CalcAmountOutResult,
) []*big.Int {
	var inputAmount = param.TokenAmountIn.Amount
	var outputAmount = res.TokenAmountOut.Amount
	// swap fee
	// output = output + output * swapFee * adminFee
	outputAmount = new(big.Int).Add(
		outputAmount,
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(new(big.Int).Mul(outputAmount, t.Info.SwapFee), FeeDenominator),
				t.baseSim.AdminFee,
			),
			FeeDenominator,
		),
	)
	updatedBalances := make([]*big.Int, len(t.Info.Reserves))
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == param.TokenAmountIn.Token {
			updatedBalances[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
		}
		if t.Info.Tokens[i] == param.TokenOut {
			updatedBalances[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
		}
	}

	return updatedBalances
}

func (t *PoolSimulator) checkPriceDiff(updatedBalances []*big.Int) error {
	if len(updatedBalances) != 2 ||
		len(t.decimals) != 2 ||
		len(t.baseSim.Multipliers) != 2 {
		return ErrInvalidValue
	}

	value := bignumber.NewBig10("100000000000000000000") // (100 * 1e18)
	dsp0 := t.decimals[0]
	dsp1 := t.decimals[1]

	dx0 := new(big.Int).Div(
		new(big.Int).Mul(
			value,
			bignumber.TenPowInt(dsp0),
		),
		t.oraclePrices[0],
	)

	dx1 := new(big.Int).Div(
		new(big.Int).Mul(
			value,
			bignumber.TenPowInt(dsp1),
		),
		t.oraclePrices[1],
	)

	dy1, err := t.getDyWithoutFee(updatedBalances, 0, 1, dx0)
	if err != nil {
		return err
	}
	dy0, err := t.getDyWithoutFee(updatedBalances, 1, 0, dx1)
	if err != nil {
		return err
	}

	price0 := new(big.Int).Div(
		new(big.Int).Mul(
			new(big.Int).Mul(dx1, t.baseSim.Multipliers[1]),
			t.oraclePrices[1],
		),
		new(big.Int).Mul(dy0, t.baseSim.Multipliers[0]),
	)
	price1 := new(big.Int).Div(
		new(big.Int).Mul(
			new(big.Int).Mul(dx0, t.baseSim.Multipliers[0]),
			t.oraclePrices[0],
		),
		new(big.Int).Mul(dy1, t.baseSim.Multipliers[1]),
	)

	priceDiff0 := lo.Ternary(
		price0.Cmp(t.oraclePrices[0]) > 0,
		new(big.Int).Sub(price0, t.oraclePrices[0]),
		new(big.Int).Sub(t.oraclePrices[0], price0),
	)

	priceDiff1 := lo.Ternary(
		price1.Cmp(t.oraclePrices[1]) > 0,
		new(big.Int).Sub(price1, t.oraclePrices[1]),
		new(big.Int).Sub(t.oraclePrices[1], price1),
	)

	priceDiff0.Mul(priceDiff0, bignumber.TenPowInt(18))
	if priceDiff0.Cmp(new(big.Int).Mul(t.oraclePrices[0], t.priceDiffThreshold[0])) > 0 {
		return ErrPriceDiffToken0
	}

	priceDiff1.Mul(priceDiff1, bignumber.TenPowInt(18))
	if priceDiff1.Cmp(new(big.Int).Mul(t.oraclePrices[1], t.priceDiffThreshold[1])) > 0 {
		return ErrPriceDiffToken1
	}

	return nil
}
