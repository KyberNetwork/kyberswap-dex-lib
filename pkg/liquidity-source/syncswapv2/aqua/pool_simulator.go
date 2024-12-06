package syncswapv2aqua

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	bignumber "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	AMultiplier = uint256.NewInt(10000)
	Precision   = big256.BONE
)

type PoolSimulator struct {
	pool.Pool
	vaultAddress     string
	swapFeesMin      []*uint256.Int
	swapFeesMax      []*uint256.Int
	swapFeesGamma    []*uint256.Int
	Precisions       []*uint256.Int
	A                *uint256.Int
	Gamma            *uint256.Int
	gas              syncswap.Gas
	PriceScalePacked *uint256.Int
	D                *uint256.Int
	LastPricesPacked *uint256.Int
	FutureTime       int64

	PriceOraclePacked   *uint256.Int
	LastPricesTimestamp int64
	LpSupply            *uint256.Int
	XcpProfit           *uint256.Int
	VirtualPrice        *uint256.Int
	NotAdjusted         bool
	AllowedExtraProfit  *uint256.Int
	AdjustmentStep      *uint256.Int
	MaHalfTime          *uint256.Int

	InitialTime  int64
	InitialA     int64
	FutureA      int64
	InitialGamma int64
	FutureGamma  int64
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra ExtraAquaPool
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
	}
	return &PoolSimulator{
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
		vaultAddress:        extra.VaultAddress,
		swapFeesMin:         []*uint256.Int{extra.SwapFee0To1Min, extra.SwapFee1To0Min},
		swapFeesMax:         []*uint256.Int{extra.SwapFee0To1Max, extra.SwapFee1To0Max},
		swapFeesGamma:       []*uint256.Int{extra.SwapFee0To1Gamma, extra.SwapFee1To0Gamma},
		PriceScalePacked:    extra.PriceScale,
		D:                   extra.D,
		A:                   extra.A,
		Gamma:               extra.Gamma,
		Precisions:          []*uint256.Int{extra.Token0PrecisionMultiplier, extra.Token1PrecisionMultiplier},
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
		uint256.MustFromBig(tokenAmountIn.Amount),
	)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}
	if amountOut.Cmp(big256.ZeroBI) > 0 {
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: amountOut.ToBig(),
			},
			Fee: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: fee.ToBig(),
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
	amountOut, err := t.Exchange(inputIndex, outputIndex, uint256.MustFromBig(inputAmount))
	if err != nil {
		return nil, nil, 0, err
	}
	return &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		}, &pool.TokenAmount{
			Token:  tokenOut,
			Amount: bignumber.ZeroBI,
		}, t.gas.Swap, nil
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return syncswap.Meta{
		VaultAddress: addressZero,
	}
}
