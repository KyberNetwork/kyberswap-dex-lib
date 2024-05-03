package stableng

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	precisionMultipliers []uint256.Int
	Reserves             []uint256.Int // same as pool.Reserves but use uint256.Int

	LpSupply uint256.Int
	gas      Gas

	NumTokens     int
	NumTokensU256 uint256.Int

	Extra       Extra
	StaticExtra StaticExtra
}

type Gas struct {
	Exchange int64
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	sim := &PoolSimulator{}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &sim.StaticExtra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.Extra), &sim.Extra); err != nil {
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

	sim.Reserves = make([]uint256.Int, numTokens)
	sim.precisionMultipliers = make([]uint256.Int, numTokens)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address

		reservesBI[i] = bignumber.NewBig10(entityPool.Reserves[i])
		if err := sim.Reserves[i].SetFromDecimal(entityPool.Reserves[i]); err != nil {
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
			SwapFee:    sim.Extra.SwapFee.ToBig(),
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

	sim.NumTokens = numTokens
	sim.NumTokensU256.SetUint64(uint64(numTokens))
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

func (t *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut := param.TokenAmountOut
	tokenIn := param.TokenIn
	// swap from token to token
	var tokenIndexFrom = t.Info.GetTokenIndex(tokenIn)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenAmountOut.Token)
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		var amountIn, adminFee, amountOut uint256.Int
		amountOut.SetFromBig(tokenAmountOut.Amount)
		err := t.GetDx(
			tokenIndexFrom,
			tokenIndexTo,
			&amountOut,
			nil,
			&amountIn,
			&adminFee,
		)
		if err != nil {
			return &pool.CalcAmountInResult{}, err
		}

		if !amountIn.IsZero() {
			return &pool.CalcAmountInResult{
				TokenAmountIn: &pool.TokenAmount{
					Token:  tokenIn,
					Amount: amountIn.ToBig(),
				},
				Fee: &pool.TokenAmount{
					Token:  tokenAmountOut.Token,
					Amount: adminFee.ToBig(),
				},
				Gas: t.gas.Exchange,
			}, nil
		}
	}

	return &pool.CalcAmountInResult{}, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", tokenIndexFrom, tokenIndexTo)
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
			t.Reserves[i].Add(&t.Reserves[i], number.SetFromBig(inputAmount))
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
			t.Reserves[i].Sub(&t.Reserves[i], number.SetFromBig(outputAmount))
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
	if len(t.StaticExtra.IsNativeCoins) == t.NumTokens {
		meta.TokenInIsNative = &t.StaticExtra.IsNativeCoins[fromId]
		meta.TokenOutIsNative = &t.StaticExtra.IsNativeCoins[toId]
	}
	return meta
}
