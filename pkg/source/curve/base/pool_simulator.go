package base

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
)

type PoolBaseSimulator struct {
	pool.Pool
	Multipliers []uint256.Int
	Rates       []uint256.Int
	Reserves    []uint256.Int // same as pool.Reserves but use uint256.Int
	// extra fields
	InitialA     uint256.Int
	FutureA      uint256.Int
	InitialATime int64
	FutureATime  int64
	AdminFee     uint256.Int
	SwapFee      uint256.Int
	LpToken      string
	LpSupply     uint256.Int
	APrecision   uint256.Int
	gas          Gas
	numTokensBI  uint256.Int
}

type Gas struct {
	Exchange int64
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolBaseSimulator, error) {
	var staticExtra curve.PoolBaseStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra curve.PoolBaseExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var numTokens = len(entityPool.Tokens)
	if entityPool.Reserves == nil || len(entityPool.Reserves) < numTokens {
		return nil, errors.New("empty reserve")
	}

	if numTokens > MaxTokenCount {
		return nil, errors.New("exceed max number of tokens")
	}

	var tokens = make([]string, numTokens)
	var reservesBI = make([]*big.Int, numTokens)
	var reserves = make([]uint256.Int, numTokens)
	var multipliers = make([]uint256.Int, numTokens)
	var rates = make([]uint256.Int, numTokens)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reservesBI[i] = bignumber.NewBig10(entityPool.Reserves[i])
		if err := multipliers[i].SetFromDecimal(staticExtra.PrecisionMultipliers[i]); err != nil {
			return nil, err
		}
		if err := rates[i].SetFromDecimal(staticExtra.Rates[i]); err != nil {
			return nil, err
		}
		if err := reserves[i].SetFromDecimal(entityPool.Reserves[i]); err != nil {
			return nil, err
		}
	}

	sim := &PoolBaseSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    bignumber.NewBig10(extra.SwapFee),
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reservesBI,
				Checked:    false,
			},
		},
		Multipliers:  multipliers,
		Rates:        rates,
		Reserves:     reserves,
		InitialATime: extra.InitialATime,
		FutureATime:  extra.FutureATime,
		LpToken:      staticExtra.LpToken,
		gas:          DefaultGas,
	}
	if err := sim.InitialA.SetFromDecimal(extra.InitialA); err != nil {
		return nil, err
	}
	if err := sim.FutureA.SetFromDecimal(extra.FutureA); err != nil {
		return nil, err
	}
	if err := sim.AdminFee.SetFromDecimal(extra.AdminFee); err != nil {
		return nil, err
	}
	if err := sim.SwapFee.SetFromDecimal(extra.SwapFee); err != nil {
		return nil, err
	}
	if err := sim.LpSupply.SetFromDecimal(entityPool.Reserves[numTokens]); err != nil {
		return nil, err
	}

	if len(staticExtra.APrecision) > 0 {
		if err := sim.APrecision.SetFromDecimal(staticExtra.APrecision); err != nil {
			return nil, err
		}
	} else {
		sim.APrecision.SetUint64(1)
	}
	sim.numTokensBI.SetUint64(uint64(numTokens))
	return sim, nil
}

func (t *PoolBaseSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	// swap from token to token
	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		var amountOut, fee, amount uint256.Int
		amount.SetFromBig(tokenAmountIn.Amount)
		err := t.GetDyU256(
			tokenIndexFrom,
			tokenIndexTo,
			&amount,
			nil,
			&amountOut, &fee,
		)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		if amountOut.Sign() > 0 {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut.ToBig(),
				},
				Fee: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: fee.ToBig(),
				},
				Gas: t.gas.Exchange,
			}, nil
		}
	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", tokenIndexFrom, tokenIndexTo)
}

func (t *PoolBaseSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = input.Amount
	var outputAmount = output.Amount
	// swap fee
	// output = output + output * swapFee * adminFee
	outputAmount = new(big.Int).Add(
		outputAmount,
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(new(big.Int).Mul(outputAmount, t.Info.SwapFee), FeeDenominator.ToBig()),
				t.AdminFee.ToBig(),
			),
			FeeDenominator.ToBig(),
		),
	)
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
			t.Reserves[i].Add(&t.Reserves[i], number.SetFromBig(inputAmount))
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
			t.Reserves[i].Sub(&t.Reserves[i], number.SetFromBig(outputAmount))
		}
	}
}

func (t *PoolBaseSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return curve.Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
	}
}
