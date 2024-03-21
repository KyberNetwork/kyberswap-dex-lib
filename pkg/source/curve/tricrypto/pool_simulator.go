package tricrypto

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
	Gas                Gas
}

type Gas struct {
	Exchange int64
}

func NewPoolSimulator(entityPool entity.Pool) (*Pool, error) {
	var staticExtra curve.PoolTricryptoStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extraStr curve.PoolTricryptoExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extraStr); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	precisions := make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		precisions[i] = bignumber.NewBig10(staticExtra.PrecisionMultipliers[i])
	}

	packedPrice := bignumber.ZeroBI
	lastPricesPacked := bignumber.ZeroBI
	priceOraclePacked := bignumber.ZeroBI
	for i := numTokens - 2; i >= 0; i -= 1 {
		var priceScale = bignumber.NewBig10(extraStr.PriceScale[i])
		packedPrice = new(big.Int).Or(new(big.Int).Lsh(packedPrice, PriceSize), priceScale)
		var lastPrice = bignumber.NewBig10(extraStr.LastPrices[i])
		lastPricesPacked = new(big.Int).Or(new(big.Int).Lsh(lastPricesPacked, PriceSize), lastPrice)
		var priceOracle = bignumber.NewBig10(extraStr.PriceOracle[i])
		priceOraclePacked = new(big.Int).Or(new(big.Int).Lsh(priceOraclePacked, PriceSize), priceOracle)
	}

	return &Pool{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    bignumber.ZeroBI,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		Precisions: precisions,
		A:          bignumber.NewBig10(extraStr.A),
		D:          bignumber.NewBig10(extraStr.D),
		Gamma:      bignumber.NewBig10(extraStr.Gamma),
		FeeGamma:   bignumber.NewBig10(extraStr.FeeGamma),
		MidFee:     bignumber.NewBig10(extraStr.MidFee),
		OutFee:     bignumber.NewBig10(extraStr.OutFee),

		PriceScalePacked:  packedPrice,
		LastPricesPacked:  lastPricesPacked,
		PriceOraclePacked: priceOraclePacked,

		FutureAGammaTime:  extraStr.FutureAGammaTime,
		FutureAGamma:      bignumber.NewBig10(extraStr.FutureAGamma),
		InitialAGammaTime: extraStr.InitialAGammaTime,
		InitialAGamma:     bignumber.NewBig10(extraStr.InitialAGamma),

		LastPricesTimestamp: extraStr.LastPricesTimestamp,
		LpToken:             staticExtra.LpToken,
		LpSupply:            bignumber.NewBig10(extraStr.LpSupply),
		XcpProfit:           bignumber.NewBig10(extraStr.XcpProfit),
		VirtualPrice:        bignumber.NewBig10(extraStr.VirtualPrice),
		AllowedExtraProfit:  bignumber.NewBig10(extraStr.AllowedExtraProfit),
		AdjustmentStep:      bignumber.NewBig10(extraStr.AdjustmentStep),
		MaHalfTime:          bignumber.NewBig10(extraStr.MaHalfTime),
		NotAdjusted:         false,
		Gas:                 DefaultGas,
	}, nil
}

func (t *Pool) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
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
		if amountOut.Cmp(bignumber.ZeroBI) > 0 {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut,
				},
				Fee: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: fee,
				},
				Gas: t.Gas.Exchange,
			}, nil

		}
	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenIndexFrom %v or tokenIndexTo %v is not correct", tokenIndexFrom, tokenIndexTo)
}

func (t *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = input.Amount
	var inputIndex = t.GetTokenIndex(input.Token)
	var outputIndex = t.GetTokenIndex(output.Token)
	_, _ = t.Exchange(inputIndex, outputIndex, inputAmount)
}

func (t *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return curve.Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
	}
}
