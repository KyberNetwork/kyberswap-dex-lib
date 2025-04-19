package twocryptong

import (
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	precisionMultipliers []uint256.Int
	Reserves             []uint256.Int // same as pool.Reserves but use uint256.Int

	NotAdjusted bool

	Extra       Extra
	StaticExtra StaticExtra

	gas int64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	sim := &PoolSimulator{}
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &sim.StaticExtra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.Extra), &sim.Extra); err != nil {
		return nil, err
	}

	var numTokens = len(entityPool.Tokens)
	if numTokens != NumTokens {
		return nil, ErrInvalidNumToken
	}

	if entityPool.Reserves == nil || len(entityPool.Reserves) != numTokens {
		return nil, ErrInvalidReserve
	}

	var tokens = make([]string, numTokens)
	var reservesBI = make([]*big.Int, numTokens)

	sim.Reserves = make([]uint256.Int, numTokens)
	sim.precisionMultipliers = make([]uint256.Int, numTokens)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address

		reservesBI[i] = bignumber.NewBig10(entityPool.Reserves[i])
		if err := sim.Reserves[i].SetFromDecimal(entityPool.Reserves[i]); err != nil {
			return nil, err
		}

		sim.precisionMultipliers[i].Exp(
			number.Number_10,
			uint256.NewInt(uint64(18-entityPool.Tokens[i].Decimals)),
		)
	}

	sim.Pool = pool.Pool{
		Info: pool.PoolInfo{
			Address:    strings.ToLower(entityPool.Address),
			ReserveUsd: entityPool.ReserveUsd,
			SwapFee:    bignumber.ZeroBI,
			Exchange:   entityPool.Exchange,
			Type:       entityPool.Type,
			Tokens:     tokens,
			Reserves:   reservesBI,
			Checked:    false,
		},
	}

	sim.gas = DefaultGas

	return sim, nil
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
	if tokenIndexFrom < 0 || tokenIndexTo < 0 {
		return nil, fmt.Errorf("tokenIndexFrom %v or tokenIndexTo %v is not correct",
			tokenIndexFrom, tokenIndexTo)
	}

	var amountOut, fee, amount uint256.Int
	amount.SetFromBig(tokenAmountIn.Amount)
	var swapInfo SwapInfo
	err := t.GetDy(
		tokenIndexFrom,
		tokenIndexTo,
		&amount,
		&amountOut, &fee, &swapInfo.K0, swapInfo.Xp[:],
	)
	if err != nil {
		return nil, err
	} else if amountOut.IsZero() {
		return nil, ErrZero
	}
	A, gamma := t._A_gamma()
	if err = t.tweak_price(A, gamma, swapInfo.Xp, nil, &swapInfo.K0,
		swapInfo.LastPrices[:], swapInfo.PriceScale[:], &swapInfo.XcpProfit, &swapInfo.D,
		&swapInfo.VirtualPrice); err != nil {
		return nil, errors.WithMessagef(ErrTweakPrice, "%v", err)
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: fee.ToBig(),
		},
		Gas:      t.gas,
		SwapInfo: swapInfo,
	}, nil
}

func (t *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn := param.TokenIn
	tokenAmountOut := param.TokenAmountOut

	var tokenIndexFrom = t.Info.GetTokenIndex(tokenIn)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenAmountOut.Token)
	if tokenIndexFrom < 0 || tokenIndexTo < 0 {
		return nil, fmt.Errorf("tokenIndexFrom %v or tokenIndexTo %v is not correct",
			tokenIndexFrom, tokenIndexTo)
	}

	var amountIn, feeDy, amountOut uint256.Int
	amountOut.SetFromBig(tokenAmountOut.Amount)

	var swapInfo SwapInfo
	err := t.GetDx(
		tokenIndexFrom,
		tokenIndexTo,
		&amountOut,
		&amountIn,
		&feeDy,
		&swapInfo.K0,
		swapInfo.Xp[:],
	)
	if err != nil {
		return nil, err
	} else if amountIn.IsZero() {
		return nil, ErrZero
	}
	A, gamma := t._A_gamma()
	if err = t.tweak_price(A, gamma, swapInfo.Xp, nil, &swapInfo.K0,
		swapInfo.LastPrices[:], swapInfo.PriceScale[:], &swapInfo.XcpProfit, &swapInfo.D,
		&swapInfo.VirtualPrice); err != nil {
		return nil, errors.WithMessagef(ErrTweakPrice, "%v", err)
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountOut.Token,
			Amount: feeDy.ToBig(),
		},
		Gas:      t.gas,
		SwapInfo: swapInfo,
	}, nil
}

func (t *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *t
	cloned.Info.Reserves = slices.Clone(t.Info.Reserves)
	cloned.Reserves = slices.Clone(t.Reserves)
	cloned.Extra.XcpProfit = t.Extra.XcpProfit
	cloned.Extra.D = t.Extra.D
	cloned.Extra.VirtualPrice = t.Extra.VirtualPrice
	return &cloned
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warnf("failed to UpdateBalance for curve-twocrypto-ng %v %v pool, wrong swapInfo type", t.Info.Address,
			t.Info.Exchange)
		return
	}

	input, output := params.TokenAmountIn, params.TokenAmountOut
	inputAmount, outputAmount := input.Amount, output.Amount
	inputIndex, outputIndex := t.GetTokenIndex(input.Token), t.GetTokenIndex(output.Token)

	t.Info.Reserves[inputIndex] = new(big.Int).Add(t.Info.Reserves[inputIndex], inputAmount)
	t.Reserves[inputIndex] = *new(uint256.Int).Add(&t.Reserves[inputIndex], number.SetFromBig(inputAmount))

	t.Info.Reserves[outputIndex] = new(big.Int).Sub(t.Info.Reserves[outputIndex], outputAmount)
	t.Reserves[outputIndex] = *new(uint256.Int).Sub(&t.Reserves[outputIndex], number.SetFromBig(outputAmount))

	t.Extra.LastPrices = swapInfo.LastPrices[:]
	t.Extra.PriceScale = swapInfo.PriceScale[:]
	t.Extra.XcpProfit = &swapInfo.XcpProfit
	t.Extra.D = &swapInfo.D
	t.Extra.VirtualPrice = &swapInfo.VirtualPrice
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	meta := curve.Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
	}
	if len(t.StaticExtra.IsNativeCoins) == len(t.Info.Tokens) {
		meta.TokenInIsNative = &t.StaticExtra.IsNativeCoins[fromId]
		meta.TokenOutIsNative = &t.StaticExtra.IsNativeCoins[toId]
	}
	return meta
}
