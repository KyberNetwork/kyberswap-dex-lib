package dodo

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type Pool struct {
	pool.Pool
	PoolState
	Tokens entity.PoolTokens
	Meta   Meta
	gas    Gas
}

func NewPool(entityPool entity.Pool) (*Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	// swapFee isn't used to calculate the amountOut, poolState.mtFeeRate and poolState.lpFeeRate are used instead
	swapFee, _ := new(big.Float).Mul(new(big.Float).SetFloat64(entityPool.SwapFee), constant.BoneFloat).Int(nil)

	info := pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		SwapFee:    swapFee,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     staticExtra.Tokens,
		Reserves:   extra.Reserves,
		Checked:    false,
	}

	b := new(big.Float).Quo(
		new(big.Float).SetInt(extra.Reserves[0]), constant.TenPowDecimals(entityPool.Tokens[0].Decimals),
	)
	q := new(big.Float).Quo(
		new(big.Float).SetInt(extra.Reserves[1]), constant.TenPowDecimals(entityPool.Tokens[1].Decimals),
	)
	b0 := new(big.Float).Quo(
		new(big.Float).SetInt(extra.TargetReserves[0]), constant.TenPowDecimals(entityPool.Tokens[0].Decimals),
	)
	q0 := new(big.Float).Quo(
		new(big.Float).SetInt(extra.TargetReserves[1]), constant.TenPowDecimals(entityPool.Tokens[1].Decimals),
	)
	i := new(big.Float).SetInt(extra.I)
	decimalize := constant.TenPowDecimals(18 - entityPool.Tokens[0].Decimals + entityPool.Tokens[1].Decimals)
	oraclePrice := new(big.Float).Quo(i, decimalize)
	k := new(big.Float).Quo(new(big.Float).SetInt(extra.K), constant.TenPowDecimals(uint8(18)))

	poolState := PoolState{
		B:           b,
		Q:           q,
		B0:          b0,
		Q0:          q0,
		RStatus:     extra.RStatus,
		OraclePrice: oraclePrice,
		k:           k,
		mtFeeRate:   extra.MtFeeRate,
		lpFeeRate:   extra.LpFeeRate,
	}

	meta := Meta{
		Type:             staticExtra.Type,
		DodoV1SellHelper: staticExtra.DodoV1SellHelper,
		BaseToken:        entityPool.Tokens[0].Address,
		QuoteToken:       entityPool.Tokens[1].Address,
	}

	return &Pool{
		Pool: pool.Pool{
			Info: info,
		},
		PoolState: poolState,
		Tokens:    entity.ClonePoolTokens(entityPool.Tokens),
		Meta:      meta,
		gas:       DefaultGas,
	}, nil
}

func (p *Pool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var totalGas int64

	if tokenAmountIn.Token == p.Info.Tokens[0] {
		if strings.EqualFold(p.Meta.Type, TypeV1Pool) {
			totalGas = p.gas.SellBaseV1
		} else {
			totalGas = p.gas.SellBaseV2
		}

		amountIn := new(big.Float).Quo(
			new(big.Float).SetInt(tokenAmountIn.Amount), constant.TenPowDecimals(uint8(p.Tokens[0].Decimals)),
		)
		amountOutF, mtFeeF, err := QuerySellBase(amountIn, &p.PoolState)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		amountOut, _ := new(big.Float).Mul(amountOutF, constant.TenPowDecimals(uint8(p.Tokens[1].Decimals))).Int(nil)
		mtFee, _ := new(big.Float).Mul(mtFeeF, constant.BoneFloat).Int(nil)
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: amountOut,
			},
			Fee: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: mtFee,
			},
			Gas: totalGas,
		}, nil
	} else if tokenAmountIn.Token == p.Info.Tokens[1] {
		if strings.EqualFold(p.Meta.Type, TypeV1Pool) {
			totalGas = p.gas.BuyBaseV1
		} else {
			totalGas = p.gas.BuyBaseV2
		}

		amountIn := new(big.Float).Quo(
			new(big.Float).SetInt(tokenAmountIn.Amount), constant.TenPowDecimals(uint8(p.Tokens[1].Decimals)),
		)
		amountOutF, mtFeeF, err := QuerySellQuote(amountIn, &p.PoolState)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		amountOut, _ := new(big.Float).Mul(amountOutF, constant.TenPowDecimals(uint8(p.Tokens[0].Decimals))).Int(nil)
		mtFee, _ := new(big.Float).Mul(mtFeeF, constant.BoneFloat).Int(nil)
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: amountOut,
			},
			Fee: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: mtFee,
			},
			Gas: totalGas,
		}, nil
	}
	return &pool.CalcAmountOutResult{}, errors.New("could not calculate the amountOut")
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var isSellBase bool
	if input.Token == p.Info.Tokens[0] {
		isSellBase = true
	} else {
		isSellBase = false
	}
	inputAmount := input.Amount
	// output.Amount was already fee-deducted in CalcAmountOut above, need to add back to update balances
	outputAmount := new(big.Int).Add(output.Amount, params.Fee.Amount)

	if isSellBase {
		// amountInF = inputAmount / 10^Tokens[0].Decimals
		// amountOutF = outputAmount / 10^Tokens[1].Decimals
		amountInF := new(big.Float).Quo(
			new(big.Float).SetInt(inputAmount), constant.TenPowDecimals(uint8(p.Tokens[0].Decimals)),
		)
		amountOutF := new(big.Float).Quo(
			new(big.Float).SetInt(outputAmount), constant.TenPowDecimals(uint8(p.Tokens[1].Decimals)),
		)
		// p.Info.Reserves[0] = p.Info.Reserves[0] + inputAmount
		// p.Info.Reserves[1] = p.Info.Reserves[1] - outputAmount - mtFee
		p.Info.Reserves[0] = new(big.Int).Add(p.Info.Reserves[0], inputAmount)
		p.Info.Reserves[1] = new(big.Int).Sub(p.Info.Reserves[1], outputAmount)

		// Update p.PoolState
		UpdateStateSellBase(amountInF, amountOutF, &p.PoolState)
	} else {
		// amountInF = inputAmount / 10^Tokens[1].Decimals
		// amountOutF = outputAmount / 10^Tokens[0].Decimals
		amountInF := new(big.Float).Quo(
			new(big.Float).SetInt(inputAmount), constant.TenPowDecimals(uint8(p.Tokens[1].Decimals)),
		)
		amountOutF := new(big.Float).Quo(
			new(big.Float).SetInt(outputAmount), constant.TenPowDecimals(uint8(p.Tokens[0].Decimals)),
		)

		// p.Info.Reserves[0] = p.Info.Reserves[0] - outputAmount - mtFee
		// p.Info.Reserves[1] = p.Info.Reserves[1] + inputAmount
		p.Info.Reserves[0] = new(big.Int).Sub(p.Info.Reserves[0], outputAmount)
		p.Info.Reserves[1] = new(big.Int).Add(p.Info.Reserves[1], inputAmount)

		// Update p.PoolState
		UpdateStateSellQuote(amountInF, amountOutF, &p.PoolState)
	}
}

func (p *Pool) GetLpToken() string {
	return p.Info.Address
}

func (p *Pool) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	return constant.Zero
}

func (p *Pool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {

	return constant.Zero
}

func (p *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return p.Meta
}
