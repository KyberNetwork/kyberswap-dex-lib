package classical

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	pool.Pool
	Storage
	Tokens entity.PoolTokens
	Meta   Meta
	gas    Gas
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

	var extra shared.V1Extra
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

	poolState := Storage{
		B:              extra.B,
		Q:              extra.Q,
		B0:             extra.B0,
		Q0:             extra.Q0,
		RStatus:        extra.RStatus,
		OraclePrice:    extra.OraclePrice,
		K:              extra.K,
		MtFeeRate:      extra.MtFeeRate,
		LpFeeRate:      extra.LpFeeRate,
		TradeAllowed:   extra.TradeAllowed,
		SellingAllowed: extra.SellingAllowed,
		BuyingAllowed:  extra.BuyingAllowed,
	}

	meta := Meta{
		Type:             staticExtra.Type,
		DodoV1SellHelper: staticExtra.DodoV1SellHelper,
		BaseToken:        entityPool.Tokens[0].Address,
		QuoteToken:       entityPool.Tokens[1].Address,
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: info,
		},
		Storage: poolState,
		Tokens:  entity.ClonePoolTokens(entityPool.Tokens),
		Meta:    meta,
		gas:     DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if !p.TradeAllowed {
		return &pool.CalcAmountOutResult{}, ErrTradeNotAllowed
	}

	if !p.SellingAllowed {
		return &pool.CalcAmountOutResult{}, ErrSellingNotAllowed
	}

	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	for i := range p.Info.Tokens {
		if p.Info.Reserves[i].Cmp(big.NewInt(0)) <= 0 {
			return &pool.CalcAmountOutResult{}, ErrReserveDepleted
		}
	}

	amountIn := number.SetFromBig(tokenAmountIn.Amount)

	if tokenAmountIn.Token == p.Info.Tokens[0] { // sell base
		receiveQuote, lpFeeQuote, mtFeeQuote, _, _, _, err := p._querySellBaseToken(amountIn)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}

		fee := new(uint256.Int).Add(lpFeeQuote, mtFeeQuote)

		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: receiveQuote.ToBig(),
			},
			RemainingTokenAmountIn: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: integer.Zero(),
			},
			Fee: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: fee.ToBig(),
			},
			Gas: p.gas.SellBase,
		}, nil
	} else if tokenAmountIn.Token == p.Info.Tokens[1] { // sell quote
		// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/dbf02abd4489dfb499b3f97118d4db1570932303/src/contracts/executor-helpers/ExecutorHelper2.sol#L346-L351
		canBuyBaseAmount, err := p.querySellQuoteToken(amountIn)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}

		spentQuoteAmount, lpFeeBase, mtFeeBase, _, _, _, err := p._queryBuyBaseToken(canBuyBaseAmount)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}

		if spentQuoteAmount.Cmp(amountIn) > 0 {
			return &pool.CalcAmountOutResult{}, ErrPaidAmountTooLarge
		}

		fee := new(uint256.Int).Add(lpFeeBase, mtFeeBase)

		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: canBuyBaseAmount.ToBig(),
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
	} else {
		return &pool.CalcAmountOutResult{}, shared.ErrInvalidToken
	}
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if !p.TradeAllowed {
		return &pool.CalcAmountInResult{}, ErrTradeNotAllowed
	}

	if !p.BuyingAllowed {
		return &pool.CalcAmountInResult{}, ErrBuyingNotAllowed
	}

	tokenIn := param.TokenIn
	tokenAmountOut := param.TokenAmountOut

	for i := range p.Info.Tokens {
		if p.Info.Reserves[i].Cmp(big.NewInt(0)) <= 0 {
			return &pool.CalcAmountInResult{}, ErrReserveDepleted
		}
	}

	if tokenAmountOut.Token != p.Info.Tokens[0] {
		return &pool.CalcAmountInResult{}, ErrOnlySupportBuyBase
	}

	amountOut := number.SetFromBig(tokenAmountOut.Amount)

	payQuote, lpFeeBase, mtFeeBase, _, _, _, err := p._queryBuyBaseToken(amountOut)
	if err != nil {
		return &pool.CalcAmountInResult{}, err
	}

	fee := new(uint256.Int).Add(lpFeeBase, mtFeeBase)

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: payQuote.ToBig(),
		},
		RemainingTokenAmountOut: &pool.TokenAmount{
			Token:  tokenAmountOut.Token,
			Amount: integer.Zero(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: fee.ToBig(),
		},
		Gas: p.gas.BuyBase,
	}, nil
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
		p.UpdateStateBuyBase(number.SetFromBig(inputAmount), number.SetFromBig(outputAmount))
	}
}

func (p *PoolSimulator) GetLpToken() string {
	return p.Info.Address
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return p.Meta
}
