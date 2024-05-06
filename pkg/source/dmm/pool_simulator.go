//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple PoolSimulator
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package dmm

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	pool.Pool
	Weights   []uint
	VReserves []*big.Int
	gas       Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var tokens = make([]string, 2)
	var weights = make([]uint, 2)
	var reserves = make([]*big.Int, 2)
	var vReserves = make([]*big.Int, 2)
	var swapFee = NewBig10(extra.FeeInPrecision)

	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		var weight0 = uint(50)
		if entityPool.Tokens[0].Weight > 0 {
			weight0 = entityPool.Tokens[0].Weight
		}
		var weight1 = uint(50)
		if entityPool.Tokens[1].Weight > 0 {
			weight1 = entityPool.Tokens[1].Weight
		}
		tokens[0] = entityPool.Tokens[0].Address
		weights[0] = weight0
		reserves[0] = NewBig10(entityPool.Reserves[0])
		vReserves[0] = NewBig10(extra.VReserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		weights[1] = weight1
		reserves[1] = NewBig10(entityPool.Reserves[1])
		vReserves[1] = NewBig10(extra.VReserves[1])
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    swapFee,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		Weights:   weights,
		VReserves: vReserves,
		gas:       defaultGas,
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var tokenInIndex = t.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("TokenInIndex: %v or TokenOutIndex: %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountOut, err := GetAmountOut(
		tokenAmountIn.Amount,
		t.Info.Reserves[tokenInIndex],
		t.Info.Reserves[tokenOutIndex],
		t.VReserves[tokenInIndex],
		t.VReserves[tokenOutIndex],
		t.Info.SwapFee,
	)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}

	var totalGas = t.gas.SwapBase
	if t.Weights[tokenInIndex] != t.Weights[tokenOutIndex] {
		totalGas = t.gas.SwapNonBase
	}

	if amountOut.Cmp(zeroBI) > 0 {
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut},
			Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: nil},
			Gas:            totalGas,
		}, nil
	}

	return &pool.CalcAmountOutResult{}, fmt.Errorf("invalid amount out: %v", amountOut.String())
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = new(big.Int).Div(new(big.Int).Mul(input.Amount, new(big.Int).Sub(bONE, t.Info.SwapFee)), bONE)
	var outputAmount = output.Amount
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
			t.VReserves[i] = new(big.Int).Add(t.VReserves[i], inputAmount)
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
			t.VReserves[i] = new(big.Int).Sub(t.VReserves[i], outputAmount)
		}
	}
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}
