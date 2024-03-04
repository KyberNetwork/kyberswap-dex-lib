package stableng

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
)

type PoolSimulator struct {
	pool.Pool

	precisionMultipliers []uint256.Int
	reserves             []uint256.Int // same as pool.Reserves but use uint256.Int

	LpSupply uint256.Int
	gas      Gas

	numTokens     int
	numTokensU256 uint256.Int

	extra       Extra
	staticExtra StaticExtra
}

type Gas struct {
	Exchange int64
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	sim := &PoolSimulator{}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &sim.staticExtra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.Extra), &sim.extra); err != nil {
		return nil, err
	}

	var numTokens = len(entityPool.Tokens)
	// Reserves: N tokens & lpSupply
	if entityPool.Reserves == nil || len(entityPool.Reserves) != numTokens+1 {
		return nil, ErrInvalidReserve
	}

	if numTokens > shared.MaxTokenCount {
		return nil, ErrInvalidNumToken
	}

	var tokens = make([]string, numTokens)
	var reservesBI = make([]*big.Int, numTokens)

	sim.reserves = make([]uint256.Int, numTokens)
	sim.precisionMultipliers = make([]uint256.Int, numTokens)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address

		reservesBI[i] = bignumber.NewBig10(entityPool.Reserves[i])
		if err := sim.reserves[i].SetFromDecimal(entityPool.Reserves[i]); err != nil {
			return nil, err
		}

		sim.precisionMultipliers[i].Exp(
			uint256.NewInt(10),
			uint256.NewInt(uint64(18-entityPool.Tokens[i].Decimals)),
		)
	}

	sim.Pool = pool.Pool{
		Info: pool.PoolInfo{
			Address:    strings.ToLower(entityPool.Address),
			ReserveUsd: entityPool.ReserveUsd,
			SwapFee:    sim.extra.SwapFee.ToBig(),
			Exchange:   entityPool.Exchange,
			Type:       entityPool.Type,
			Tokens:     tokens,
			Reserves:   reservesBI,
			Checked:    false,
		},
	}

	sim.gas = DefaultGas

	if err := sim.LpSupply.SetFromDecimal(entityPool.Reserves[numTokens]); err != nil {
		return nil, err
	}

	sim.numTokens = numTokens
	sim.numTokensU256.SetUint64(uint64(numTokens))
	return sim, nil
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	// swap from token to token
	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		var amountOut, adminFee, amount uint256.Int
		amount.SetFromBig(tokenAmountIn.Amount)
		err := t.GetDy(
			tokenIndexFrom,
			tokenIndexTo,
			&amount,
			nil,
			&amountOut, &adminFee,
		)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		if !amountOut.IsZero() {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut.ToBig(),
				},
				Fee: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: adminFee.ToBig(),
				},
				Gas: t.gas.Exchange,
			}, nil
		}
	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", tokenIndexFrom, tokenIndexTo)
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = input.Amount
	var outputAmount = output.Amount
	// output = output + adminFee
	outputAmount = new(big.Int).Add(
		outputAmount,
		params.Fee.Amount,
	)
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
			t.reserves[i].Add(&t.reserves[i], number.SetFromBig(inputAmount))
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
			t.reserves[i].Sub(&t.reserves[i], number.SetFromBig(outputAmount))
		}
	}
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	meta := curve.Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
	}
	if len(t.staticExtra.IsNativeCoins) == t.numTokens {
		meta.TokenInIsNative = &t.staticExtra.IsNativeCoins[fromId]
		meta.TokenOutIsNative = &t.staticExtra.IsNativeCoins[toId]
	}
	return meta
}
