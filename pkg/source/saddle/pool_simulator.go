package saddle

import (
	"errors"
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	Multipliers []*big.Int
	// extra fields
	InitialA           *big.Int
	FutureA            *big.Int
	InitialATime       int64
	FutureATime        int64
	AdminFee           *big.Int
	DefaultWithdrawFee *big.Int
	LpToken            string
	LpSupply           *big.Int
	gas                Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	multipliers := make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = utils.NewBig10(entityPool.Reserves[i])
		multipliers[i] = utils.NewBig10(staticExtra.PrecisionMultipliers[i])
	}

	swapFee := utils.NewBig10(extra.SwapFee)

	// only have withdrawFee in saddle v1, default to 0
	defaultWithdrawFee := utils.NewBig10(extra.DefaultWithdrawFee)
	if defaultWithdrawFee == nil {
		defaultWithdrawFee = constant.ZeroBI
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
		Multipliers:        multipliers,
		InitialA:           utils.NewBig10(extra.InitialA),
		FutureA:            utils.NewBig10(extra.FutureA),
		InitialATime:       extra.InitialATime,
		FutureATime:        extra.FutureATime,
		AdminFee:           utils.NewBig10(extra.AdminFee),
		DefaultWithdrawFee: defaultWithdrawFee,
		LpToken:            staticExtra.LpToken,
		LpSupply:           utils.NewBig10(entityPool.Reserves[numTokens]),
		gas:                DefaultGas,
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var balances = t.Info.Reserves
	var tokenPrecisionMultipliers = t.Multipliers
	if tokenAmountIn.Token == t.LpToken {
		// withdraw
		var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
		if tokenIndexTo >= 0 {
			amountOut, fee, err := CalculateRemoveLiquidityOneToken(
				balances,
				tokenPrecisionMultipliers,
				t.FutureATime,
				t.FutureA,
				t.InitialATime,
				t.InitialA,
				t.Info.SwapFee,
				t.DefaultWithdrawFee,
				t.LpSupply,
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
					Gas: t.gas.RemoveLiquidity,
				}, nil
			}
		}
	} else if tokenOut == t.LpToken {
		// deposit
		var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
		if tokenIndexFrom >= 0 {
			amountOut, fee, err := CalculateAddLiquidityOneToken(
				balances,
				tokenPrecisionMultipliers,
				t.FutureATime,
				t.FutureA,
				t.InitialATime,
				t.InitialA,
				t.DefaultWithdrawFee,
				t.LpSupply,
				tokenIndexFrom,
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
					Gas: t.gas.AddLiquidity,
				}, nil
			}
		}
	} else {
		// swap from token to token
		var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
		var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
		if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
			amountOut, fee, err := CalculateSwap(
				balances,
				tokenPrecisionMultipliers,
				t.FutureATime,
				t.FutureA,
				t.InitialATime,
				t.InitialA,
				t.Info.SwapFee,
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
		}
	}
	return &pool.CalcAmountOutResult{}, errors.New("i'm dead here")
}

func (t *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *t
	cloned.LpSupply = new(big.Int).Set(t.LpSupply)
	cloned.Info.Reserves = lo.Map(t.Info.Reserves, func(v *big.Int, i int) *big.Int {
		return new(big.Int).Set(v)
	})
	return &cloned
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output, fee := params.TokenAmountIn, params.TokenAmountOut, params.Fee
	var inputAmount = input.Amount
	var outputAmount = output.Amount
	if input.Token == t.LpToken {
		// withdraw
		outputAmount = new(big.Int).Add(
			outputAmount,
			new(big.Int).Div(
				new(big.Int).Mul(fee.Amount, t.AdminFee),
				FeeDenominator,
			),
		)
		t.LpSupply = new(big.Int).Sub(t.LpSupply, inputAmount)
	} else if output.Token == t.LpToken {
		// deposit
		t.LpSupply = new(big.Int).Add(t.LpSupply, outputAmount)
	} else {
		// swap fee
		// output = output + output * swapFee * adminFee
		outputAmount = new(big.Int).Add(
			outputAmount,
			new(big.Int).Div(
				new(big.Int).Mul(
					new(big.Int).Div(new(big.Int).Mul(outputAmount, t.Info.SwapFee), FeeDenominator),
					t.AdminFee,
				),
				FeeDenominator,
			),
		)
	}
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
		}
	}
}

func (t *PoolSimulator) CanSwapFrom(address string) []string { return t.CanSwapTo(address) }

func (t *PoolSimulator) CanSwapTo(address string) []string {
	var ret = make([]string, 0)
	var tokenIndex = t.GetTokenIndex(address)
	if tokenIndex < 0 && address != t.LpToken {
		return nil
	}
	for i := 0; i < len(t.Info.Tokens); i += 1 {
		if i != tokenIndex {
			ret = append(ret, t.Info.Tokens[i])
		}
	}
	if address != t.LpToken {
		ret = append(ret, t.LpToken)
	}
	return ret
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		PoolLength:    len(t.Info.Tokens),
	}
}
