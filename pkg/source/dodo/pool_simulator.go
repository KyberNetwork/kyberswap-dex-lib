package dodo

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulatorState struct {
	B           *big.Float // DODO._BASE_BALANCE_() / 10^baseDecimals
	Q           *big.Float // DODO._QUOTE_BALANCE_() / 10^quoteDecimals
	B0          *big.Float // DODO._TARGET_BASE_TOKEN_AMOUNT_() / 10^baseDecimals
	Q0          *big.Float // DODO._TARGET_QUOTE_TOKEN_AMOUNT_() / 10^quoteDecimals
	RStatus     int        // DODO._R_STATUS_()
	OraclePrice *big.Float // DODO.getOraclePrice() / 10^(18-baseDecimals+quoteDecimals)
	k           *big.Float // DODO._K_()/10^18
	mtFeeRate   *big.Float // DODO._MT_FEE_RATE_()/10^18
	lpFeeRate   *big.Float // DODO._LP_FEE_RATE_()/10^18
}

type Pool struct {
	pool.Pool
	PoolSimulatorState
	Tokens entity.PoolTokens
	Meta   Meta
	gas    Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	// swapFee isn't used to calculate the amountOut, poolState.mtFeeRate and poolState.lpFeeRate are used instead
	swapFee, _ := new(big.Float).Mul(new(big.Float).SetFloat64(entityPool.SwapFee), bignumber.BoneFloat).Int(nil)

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
		new(big.Float).SetInt(extra.Reserves[0]), bignumber.TenPowDecimals(entityPool.Tokens[0].Decimals),
	)
	q := new(big.Float).Quo(
		new(big.Float).SetInt(extra.Reserves[1]), bignumber.TenPowDecimals(entityPool.Tokens[1].Decimals),
	)
	b0 := new(big.Float).Quo(
		new(big.Float).SetInt(extra.TargetReserves[0]), bignumber.TenPowDecimals(entityPool.Tokens[0].Decimals),
	)
	q0 := new(big.Float).Quo(
		new(big.Float).SetInt(extra.TargetReserves[1]), bignumber.TenPowDecimals(entityPool.Tokens[1].Decimals),
	)
	i := new(big.Float).SetInt(extra.I)
	decimalize := bignumber.TenPowDecimals(18 - entityPool.Tokens[0].Decimals + entityPool.Tokens[1].Decimals)
	oraclePrice := new(big.Float).Quo(i, decimalize)
	k := new(big.Float).Quo(new(big.Float).SetInt(extra.K), bignumber.TenPowDecimals(uint8(18)))

	poolState := PoolSimulatorState{
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
		PoolSimulatorState: poolState,
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
			new(big.Float).SetInt(tokenAmountIn.Amount), bignumber.TenPowDecimals(uint8(p.Tokens[0].Decimals)),
		)
		amountOutF, mtFeeF, err := QuerySellBase(amountIn, &p.PoolSimulatorState)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		amountOut, _ := new(big.Float).Mul(amountOutF, bignumber.TenPowDecimals(uint8(p.Tokens[1].Decimals))).Int(nil)
		mtFee, _ := new(big.Float).Mul(mtFeeF, bignumber.BoneFloat).Int(nil)
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
			new(big.Float).SetInt(tokenAmountIn.Amount), bignumber.TenPowDecimals(uint8(p.Tokens[1].Decimals)),
		)
		amountOutF, mtFeeF, err := QuerySellQuote(amountIn, &p.PoolSimulatorState)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		amountOut, _ := new(big.Float).Mul(amountOutF, bignumber.TenPowDecimals(uint8(p.Tokens[0].Decimals))).Int(nil)
		mtFee, _ := new(big.Float).Mul(mtFeeF, bignumber.BoneFloat).Int(nil)
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
			new(big.Float).SetInt(inputAmount), bignumber.TenPowDecimals(uint8(p.Tokens[0].Decimals)),
		)
		amountOutF := new(big.Float).Quo(
			new(big.Float).SetInt(outputAmount), bignumber.TenPowDecimals(uint8(p.Tokens[1].Decimals)),
		)
		// p.Info.Reserves[0] = p.Info.Reserves[0] + inputAmount
		// p.Info.Reserves[1] = p.Info.Reserves[1] - outputAmount - mtFee
		p.Info.Reserves[0] = new(big.Int).Add(p.Info.Reserves[0], inputAmount)
		p.Info.Reserves[1] = new(big.Int).Sub(p.Info.Reserves[1], outputAmount)

		// Update p.PoolSimulatorState
		UpdateStateSellBase(amountInF, amountOutF, &p.PoolSimulatorState)
	} else {
		// amountInF = inputAmount / 10^Tokens[1].Decimals
		// amountOutF = outputAmount / 10^Tokens[0].Decimals
		amountInF := new(big.Float).Quo(
			new(big.Float).SetInt(inputAmount), bignumber.TenPowDecimals(uint8(p.Tokens[1].Decimals)),
		)
		amountOutF := new(big.Float).Quo(
			new(big.Float).SetInt(outputAmount), bignumber.TenPowDecimals(uint8(p.Tokens[0].Decimals)),
		)

		// p.Info.Reserves[0] = p.Info.Reserves[0] - outputAmount - mtFee
		// p.Info.Reserves[1] = p.Info.Reserves[1] + inputAmount
		p.Info.Reserves[0] = new(big.Int).Sub(p.Info.Reserves[0], outputAmount)
		p.Info.Reserves[1] = new(big.Int).Add(p.Info.Reserves[1], inputAmount)

		// Update p.PoolSimulatorState
		UpdateStateSellQuote(amountInF, amountOutF, &p.PoolSimulatorState)
	}
}

func (p *Pool) GetLpToken() string {
	return p.Info.Address
}

func (p *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return p.Meta
}
