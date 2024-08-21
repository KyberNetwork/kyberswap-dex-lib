package dpp

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/libv2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	pool.Pool
	libv2.PMMState
	Tokens entity.PoolTokens
	Meta   shared.V2Meta
	gas    shared.V2Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	if len(entityPool.StaticExtra) == 0 {
		return nil, shared.ErrStaticExtraEmpty
	}

	if len(entityPool.Extra) == 0 {
		return nil, shared.ErrExtraEmpty
	}

	var staticExtra shared.StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra shared.V2Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	// swapFee isn't used to calculate the amountOut, poolState.mtFeeRate and poolState.lpFeeRate are used instead
	swapFee := number.Add(extra.LpFeeRate, extra.MtFeeRate).ToBig()

	info := pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		SwapFee:    swapFee,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     staticExtra.Tokens,
		Reserves:   []*big.Int{extra.B.ToBig(), extra.Q.ToBig()},
		Checked:    false,
	}

	poolState := libv2.PMMState{
		I:         extra.I,
		K:         extra.K,
		B:         extra.B,
		Q:         extra.Q,
		B0:        extra.B0,
		Q0:        extra.Q0,
		R:         libv2.RState(extra.R.Uint64()),
		MtFeeRate: extra.MtFeeRate,
		LpFeeRate: extra.LpFeeRate,
	}

	libv2.AdjustedTarget(&poolState)

	meta := shared.V2Meta{
		Type:       staticExtra.Type,
		BaseToken:  entityPool.Tokens[0].Address,
		QuoteToken: entityPool.Tokens[1].Address,
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: info,
		},
		PMMState: poolState,
		Tokens:   entity.ClonePoolTokens(entityPool.Tokens),
		Meta:     meta,
		gas:      shared.V2DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	amountIn := number.SetFromBig(tokenAmountIn.Amount)

	if tokenAmountIn.Token == p.Info.Tokens[0] { // tokenIn is base token
		receiveQuoteAmount, lpFee, mtFee, err := p.querySellBase(amountIn)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}

		fee := new(uint256.Int).Add(lpFee, mtFee)

		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: receiveQuoteAmount.ToBig(),
			},
			RemainingTokenAmountIn: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: integer.Zero(),
			},
			Fee: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: fee.ToBig(),
			},
			Gas: p.gas.SellBase,
		}, nil
	} else if tokenAmountIn.Token == p.Info.Tokens[1] { // tokenIn is quote token
		receiveBaseAmount, lpFee, mtFee, err := p.querySellQuote(amountIn)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}

		fee := new(uint256.Int).Add(lpFee, mtFee)

		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: receiveBaseAmount.ToBig(),
			},
			RemainingTokenAmountIn: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: integer.Zero(),
			},
			Fee: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: fee.ToBig(),
			},
			Gas: p.gas.SellQuote,
		}, nil
	}

	return &pool.CalcAmountOutResult{}, shared.ErrInvalidToken
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
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
		// p.Info.Reserves[0] = p.Info.Reserves[0] + inputAmount
		// p.Info.Reserves[1] = p.Info.Reserves[1] - outputAmount - mtFee
		p.Info.Reserves[0].Add(p.Info.Reserves[0], inputAmount)
		p.Info.Reserves[1].Sub(p.Info.Reserves[1], outputAmount)

		// Update p.Storage
		p.UpdateStateSellBase(number.SetFromBig(inputAmount), number.SetFromBig(outputAmount))
	} else {
		// p.Info.Reserves[0] = p.Info.Reserves[0] - outputAmount - mtFee
		// p.Info.Reserves[1] = p.Info.Reserves[1] + inputAmount
		p.Info.Reserves[0].Sub(p.Info.Reserves[0], outputAmount)
		p.Info.Reserves[1].Add(p.Info.Reserves[1], inputAmount)

		// Update p.Storage
		p.UpdateStateSellQuote(number.SetFromBig(inputAmount), number.SetFromBig(outputAmount))
	}

	libv2.AdjustedTarget(&p.PMMState)
}

func (p *PoolSimulator) GetLpToken() string {
	return p.Info.Address
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return p.Meta
}
