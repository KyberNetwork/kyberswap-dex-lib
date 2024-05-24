package syncswapv2aqua

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswapv2"
	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	vaultAddress     string
	swapFeesMin      []*big.Int
	swapFeesMax      []*big.Int
	swapFeesGamma    []*big.Int
	Precisions       []*big.Int
	A                *big.Int
	Gamma            *big.Int
	gas              syncswap.Gas
	PriceScalePacked *big.Int
	D                *big.Int
	LastPricesPacked *big.Int
	FutureTime       int64

	PriceOraclePacked   *big.Int
	LastPricesTimestamp int64
	LpSupply            *big.Int
	XcpProfit           *big.Int
	VirtualPrice        *big.Int
	NotAdjusted         bool
	AllowedExtraProfit  *big.Int
	AdjustmentStep      *big.Int
	MaHalfTime          *big.Int

	InitialTime  int64
	InitialA     int64
	FutureA      int64
	InitialGamma int64
	FutureGamma  int64
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra syncswapv2.ExtraAquaPool
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = utils.NewBig10(entityPool.Reserves[i])
	}
	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    constant.ZeroBI,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		vaultAddress:        extra.VaultAddress,
		swapFeesMin:         []*big.Int{extra.SwapFee0To1Min, extra.SwapFee1To0Min},
		swapFeesMax:         []*big.Int{extra.SwapFee0To1Max, extra.SwapFee1To0Max},
		swapFeesGamma:       []*big.Int{extra.SwapFee0To1Gamma, extra.SwapFee1To0Gamma},
		PriceScalePacked:    extra.PriceScale,
		D:                   extra.D,
		A:                   extra.A,
		Gamma:               extra.Gamma,
		Precisions:          []*big.Int{extra.Token0PrecisionMultiplier, extra.Token1PrecisionMultiplier},
		LastPricesPacked:    extra.LastPrices,
		FutureTime:          extra.FutureTime,
		PriceOraclePacked:   extra.PriceOracle,
		LastPricesTimestamp: extra.LastPricesTimestamp,
		LpSupply:            extra.LpSupply,
		XcpProfit:           extra.XcpProfit,
		VirtualPrice:        extra.VirtualPrice,
		NotAdjusted:         false,
		AllowedExtraProfit:  extra.AllowedExtraProfit,
		AdjustmentStep:      extra.AdjustmentStep,
		MaHalfTime:          extra.MaHalfTime,

		InitialTime:  extra.InitialTime,
		InitialA:     extra.InitialA,
		FutureA:      extra.FutureA,
		InitialGamma: extra.InitialGamma,
		FutureGamma:  extra.FutureGamma,
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	// swap from token to token
	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
	amountOut, fee, err := t.GetDy(
		tokenIndexFrom,
		tokenIndexTo,
		tokenAmountIn.Amount,
	)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}
	if amountOut.Cmp(constant.ZeroBI) > 0 {
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: amountOut,
			},
			Fee: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: fee,
			},
			Gas: t.gas.Swap,
		}, nil

	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf(
		"tokenIndexFrom %v or tokenIndexTo %v is not correct", tokenIndexFrom, tokenIndexTo,
	)
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	_, _, _, _ = t.Swap(input, output.Token)
}

func (t *PoolSimulator) Swap(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.TokenAmount, *pool.TokenAmount, int64, error) {
	var inputAmount = tokenAmountIn.Amount
	var inputIndex = t.GetTokenIndex(tokenAmountIn.Token)
	var outputIndex = t.GetTokenIndex(tokenOut)
	amountOut, err := t.Exchange(inputIndex, outputIndex, inputAmount)
	if err != nil {
		return nil, nil, 0, err
	}
	return &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		}, &pool.TokenAmount{
			Token:  tokenOut,
			Amount: constant.ZeroBI,
		}, t.gas.Swap, nil
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return syncswap.Meta{
		VaultAddress: p.vaultAddress,
	}
}
