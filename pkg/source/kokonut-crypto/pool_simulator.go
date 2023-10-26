package kokonutcrypto

import (
	"encoding/json"
	"fmt"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	Precisions                     []*big.Int
	A                              *big.Int
	Gamma                          *big.Int
	D                              *big.Int
	FeeGamma                       *big.Int
	MidFee                         *big.Int
	OutFee                         *big.Int
	FutureAGammaTime               int64
	FutureA                        *big.Int
	FutureGamma                    *big.Int
	InitialAGammaTime              int64
	InitialA                       *big.Int
	InitialGamma                   *big.Int
	MinRemainingPostRebalanceRatio *big.Int

	LastPricesTimestamp int64
	PriceScale          *big.Int
	PriceOracle         *big.Int
	LastPrices          *big.Int

	LpToken            string
	LpSupply           *big.Int
	XcpProfit          *big.Int
	VirtualPrice       *big.Int
	AllowedExtraProfit *big.Int
	AdjustmentStep     *big.Int
	MaHalfTime         *big.Int
	NotAdjusted        bool
	gas                Gas
}

type Gas struct {
	Exchange int64
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extraStr Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extraStr); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	precisions := make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = utils.NewBig10(entityPool.Reserves[i])
		precisions[i] = utils.NewBig10(staticExtra.PrecisionMultipliers[i])
	}

	priceScale := utils.NewBig10(extraStr.PriceScale)
	lastPrices := utils.NewBig10(extraStr.LastPrices)
	priceOracle := utils.NewBig10(extraStr.PriceOracle)

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
		Precisions: precisions,
		A:          utils.NewBig10(extraStr.A),
		D:          utils.NewBig10(extraStr.D),
		Gamma:      utils.NewBig10(extraStr.Gamma),
		FeeGamma:   utils.NewBig10(extraStr.FeeGamma),
		MidFee:     utils.NewBig10(extraStr.MidFee),
		OutFee:     utils.NewBig10(extraStr.OutFee),

		PriceScale:  priceScale,
		LastPrices:  lastPrices,
		PriceOracle: priceOracle,

		FutureAGammaTime:               extraStr.FutureAGammaTime,
		FutureA:                        utils.NewBig10(extraStr.FutureA),
		FutureGamma:                    utils.NewBig10(extraStr.FutureGamma),
		InitialAGammaTime:              extraStr.InitialAGammaTime,
		InitialA:                       utils.NewBig10(extraStr.InitialA),
		InitialGamma:                   utils.NewBig10(extraStr.InitialGamma),
		MinRemainingPostRebalanceRatio: utils.NewBig10(extraStr.MinRemainingPostRebalanceRatio),

		LastPricesTimestamp: extraStr.LastPricesTimestamp,
		LpToken:             staticExtra.LpToken,
		LpSupply:            utils.NewBig10(extraStr.LpSupply),
		XcpProfit:           utils.NewBig10(extraStr.XcpProfit),
		VirtualPrice:        utils.NewBig10(extraStr.VirtualPrice),
		AllowedExtraProfit:  utils.NewBig10(extraStr.AllowedExtraProfit),
		AdjustmentStep:      utils.NewBig10(extraStr.AdjustmentStep),
		MaHalfTime:          utils.NewBig10(extraStr.MaHalfTime),
		NotAdjusted:         false,
		gas:                 DefaultGas,
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	// swap from token to token
	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
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
				Gas: t.gas.Exchange,
			}, nil

		}
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
		}, t.gas.Exchange, nil
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return curve.Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
	}
}
