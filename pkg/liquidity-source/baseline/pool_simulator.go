package baseline

import (
	"math/big"
	"slices"
	"strings"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	pool.Pool
	extra Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if len(entityPool.Extra) > 0 {
		if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
			return nil, err
		}
	}

	reserves := make([]*big.Int, len(entityPool.Tokens))
	for i, r := range entityPool.Reserves {
		reserve, ok := new(big.Int).SetString(r, 10)
		if !ok {
			reserve = big.NewInt(0)
		}
		reserves[i] = reserve
	}

	info := pool.PoolInfo{
		Address:  strings.ToLower(entityPool.Address),
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   lo.Map(entityPool.Tokens, func(e *entity.PoolToken, _ int) string { return e.Address }),
		Reserves: reserves,
	}

	return &PoolSimulator{
		Pool:  pool.Pool{Info: info},
		extra: extra,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut

	tokenInIndex := p.GetTokenIndex(tokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	}
	if tokenAmountIn.Amount == nil || tokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	isBuy, err := swapDirection(tokenInIndex, tokenOutIndex)
	if err != nil {
		return nil, err
	}

	quote, err := p.quoteAmountOut(isBuy, tokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}
	amountOut := quote.AmountOut.ToBig()
	if amountOut.Sign() <= 0 {
		return nil, ErrNoRate
	}
	if amountOut.Cmp(p.effectiveReserveLimit(tokenOutIndex)) > 0 {
		return nil, ErrInvalidAmountOut
	}

	swapInfo := SwapInfo{
		RelayAddress: p.extra.RelayAddress,
		BToken:       p.Info.Address,
		IsBuy:        isBuy,
		State:        quote.State,
	}
	if isBuy {
		swapInfo.AmountOut = amountOut.String()
	}
	if quote.ReserveDelta != nil {
		swapInfo.ReserveDelta = quote.ReserveDelta.String()
	}
	if quote.Fee != nil {
		swapInfo.Fee = quote.Fee.String()
	}

	result := &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: p.Info.Tokens[0], Amount: quote.Fee.ToBig()},
		SwapInfo:       swapInfo,
		Gas:            defaultGas,
	}
	if isBuy && quote.ReserveDelta != nil {
		actualCost := absBI(quote.ReserveDelta)
		if tokenAmountIn.Amount.Cmp(actualCost) > 0 {
			result.RemainingTokenAmountIn = &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: subBI(tokenAmountIn.Amount, actualCost),
			}
		}
	}
	return result, nil
}

func (p *PoolSimulator) effectiveReserveLimit(tokenIndex int) *big.Int {
	reserves := p.GetReserves()
	if tokenIndex != 0 || p.extra.QuoteState == nil {
		return reserves[tokenIndex]
	}

	state := p.extra.QuoteState
	limit := uToBI(state.TotalReserves)
	if state.SettlePendingSurplus && state.PendingSurplus != nil {
		bufferThreshold := mulWad(uToBI(state.TotalSupply), mustBI("950000000000000000"))
		if uToBI(state.TotalBTokens).Cmp(bufferThreshold) < 0 {
			limit = addBI(limit, uToBI(state.PendingSurplus))
		}
	}
	return limit
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut, tokenIn := param.TokenAmountOut, param.TokenIn

	tokenInIndex := p.GetTokenIndex(tokenIn)
	tokenOutIndex := p.GetTokenIndex(tokenAmountOut.Token)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	}
	if tokenAmountOut.Amount == nil || tokenAmountOut.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}

	isBuy, err := swapDirection(tokenInIndex, tokenOutIndex)
	if err != nil {
		return nil, err
	}

	quote, err := p.quoteAmountIn(isBuy, tokenAmountOut.Amount)
	if err != nil {
		return nil, err
	}
	amountIn := quote.AmountOut.ToBig()
	if amountIn.Sign() <= 0 {
		return nil, ErrNoRate
	}

	swapInfo := SwapInfo{
		RelayAddress: p.extra.RelayAddress,
		BToken:       p.Info.Address,
		IsBuy:        isBuy,
		State:        quote.State,
	}
	if isBuy {
		swapInfo.AmountOut = tokenAmountOut.Amount.String()
	}
	if quote.ReserveDelta != nil {
		swapInfo.ReserveDelta = quote.ReserveDelta.String()
	}
	if quote.Fee != nil {
		swapInfo.Fee = quote.Fee.String()
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		Fee:           &pool.TokenAmount{Token: p.Info.Tokens[0], Amount: quote.Fee.ToBig()},
		SwapInfo:      swapInfo,
		Gas:           defaultGas,
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Info.Reserves = slices.Clone(p.Info.Reserves)
	cloned.extra.QuoteState = cloneQuoteState(p.extra.QuoteState)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenAmtIn, tokenAmtOut := params.TokenAmountIn, params.TokenAmountOut
	inIndex := p.GetTokenIndex(tokenAmtIn.Token)
	outIndex := p.GetTokenIndex(tokenAmtOut.Token)
	p.Info.Reserves = slices.Clone(p.Info.Reserves)
	p.Info.Reserves[inIndex] = new(big.Int).Add(p.Info.Reserves[inIndex], tokenAmtIn.Amount)
	p.Info.Reserves[outIndex] = new(big.Int).Sub(p.Info.Reserves[outIndex], tokenAmtOut.Amount)

	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok && swapInfo.State != nil {
		p.extra.QuoteState = cloneQuoteState(swapInfo.State)
		p.Info.Reserves[0] = uToBI(swapInfo.State.TotalReserves)
		p.Info.Reserves[1] = uToBI(swapInfo.State.TotalBTokens)
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	tokenInIndex := p.GetTokenIndex(tokenIn)
	tokenOutIndex := p.GetTokenIndex(tokenOut)
	isBuy, _ := swapDirection(tokenInIndex, tokenOutIndex)

	return PoolMeta{
		Pool:        p.extra.RelayAddress,
		IsBuyBase:   isBuy,
		BlockNumber: p.Info.BlockNumber,
	}
}

func swapDirection(tokenInIndex, tokenOutIndex int) (bool, error) {
	// Token[0] = reserve, Token[1] = bToken.
	switch {
	case tokenInIndex == 0 && tokenOutIndex == 1:
		return true, nil
	case tokenInIndex == 1 && tokenOutIndex == 0:
		return false, nil
	default:
		return false, ErrInvalidToken
	}
}
