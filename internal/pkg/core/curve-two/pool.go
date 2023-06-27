package curveTwo

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	curveBase "github.com/KyberNetwork/router-service/internal/pkg/core/curve-base"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type Pool struct {
	pool.Pool
	Precisions        []*big.Int
	A                 *big.Int
	Gamma             *big.Int
	D                 *big.Int
	FeeGamma          *big.Int
	MidFee            *big.Int
	OutFee            *big.Int
	FutureAGammaTime  int64
	FutureAGamma      *big.Int
	InitialAGammaTime int64
	InitialAGamma     *big.Int

	LastPricesTimestamp int64
	PriceScalePacked    *big.Int
	PriceOraclePacked   *big.Int
	LastPricesPacked    *big.Int

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

func NewPool(entityPool entity.Pool) (*Pool, error) {
	var staticExtra curveBase.PoolStaticExtra
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

	packedPrice := utils.NewBig10(extraStr.PriceScale)
	lastPricesPacked := utils.NewBig10(extraStr.LastPrices)
	priceOraclePacked := utils.NewBig10(extraStr.PriceOracle)

	return &Pool{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    constant.Zero,
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

		PriceScalePacked:  packedPrice,
		LastPricesPacked:  lastPricesPacked,
		PriceOraclePacked: priceOraclePacked,

		FutureAGammaTime:  extraStr.FutureAGammaTime,
		FutureAGamma:      utils.NewBig10(extraStr.FutureAGamma),
		InitialAGammaTime: extraStr.InitialAGammaTime,
		InitialAGamma:     utils.NewBig10(extraStr.InitialAGamma),

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

func (t *Pool) CalcAmountOut(
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
		if amountOut.Cmp(constant.Zero) > 0 {
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

func (t *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	_, _, _, _ = t.Swap(input, output.Token)
}

func (t *Pool) Swap(
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
			Amount: constant.Zero,
		}, t.gas.Exchange, nil
}

func (t *Pool) GetLpToken() string {
	return t.LpToken
}

func (t *Pool) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	tokenInIndex := t.GetTokenIndex(tokenIn)
	tokenOutIndex := t.GetTokenIndex(tokenOut)
	reserveIn := t.LpSupply
	reserveOut := t.LpSupply
	if tokenIn != t.LpToken {
		reserveIn = t.Info.Reserves[tokenInIndex]
	}
	if tokenOut != t.LpToken {
		reserveOut = t.Info.Reserves[tokenOutIndex]
	}
	var ret = new(big.Int).Mul(base, reserveOut)
	ret = new(big.Int).Div(ret, reserveIn)
	return ret
}

func (t *Pool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	tokenInIndex := t.GetTokenIndex(tokenIn)
	tokenOutIndex := t.GetTokenIndex(tokenOut)
	reserveIn := t.LpSupply
	reserveOut := t.LpSupply
	if tokenIn != t.LpToken {
		reserveIn = t.Info.Reserves[tokenInIndex]
	}
	if tokenOut != t.LpToken {
		reserveOut = t.Info.Reserves[tokenOutIndex]
	}
	var exactQuote = new(big.Int).Mul(base, reserveOut)
	exactQuote = new(big.Int).Div(exactQuote, reserveIn)
	return exactQuote
}

func (t *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
	}
}
