package slyngfun

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

// nowUnix is the clock the opening surcharge is read against.
var nowUnix = func() uint64 { return uint64(time.Now().Unix()) }

// PoolSimulator prices one Slyng curve. Token 0 is the quote asset the coin is priced in and
// token 1 the coin itself, so a swap into token 1 is a buy and the other way a sell.
//
// A buy is priced for a buyer who is not the coin's creator, which the router never is: the
// creator's position cap and the creator's time gate live in the contracts and bind only the
// creator's own address.
type PoolSimulator struct {
	pool.Pool

	launchpad     string
	isNativeQuote bool
	graduated     bool
	curve         curveState

	gas Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	for _, v := range []*uint256.Int{extra.QuoteReserve, extra.TokenReserve, staticExtra.GraduationTarget,
		staticExtra.VirtualQuote} {
		if v == nil {
			return nil, ErrFailedCall
		}
	}
	if len(entityPool.Tokens) != 2 || staticExtra.Launchpad == "" {
		return nil, ErrInvalidToken
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig10(r) }),
			SwapFee:     new(big.Int).SetUint64(staticExtra.TradeFeeBps),
			BlockNumber: entityPool.BlockNumber,
		}},
		launchpad:     staticExtra.Launchpad,
		isNativeQuote: staticExtra.IsNativeQuote,
		graduated:     extra.Graduated,
		curve: curveState{
			quoteReserve:     *extra.QuoteReserve,
			tokenReserve:     *extra.TokenReserve,
			virtualQuote:     *staticExtra.VirtualQuote,
			graduationTarget: *staticExtra.GraduationTarget,
			createdAt:        staticExtra.CreatedAt,
			tradeFeeBps:      staticExtra.TradeFeeBps,
			snipeBps:         staticExtra.SnipeBps,
			snipeWindow:      staticExtra.SnipeWindowSeconds,
		},
		gas: defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenOut
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}
	if p.graduated {
		return nil, ErrGraduated
	}

	var amountIn uint256.Int
	if overflow := amountIn.SetFromBig(params.TokenAmountIn.Amount); overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}

	swapInfo := &SwapInfo{
		IsBuy:         indexIn == 0,
		IsNativeQuote: p.isNativeQuote,
		Launchpad:     p.launchpad,
		Token:         p.Info.Tokens[1],
	}
	var amountOut, cost uint256.Int
	gas := p.gas.SellERC20
	if p.isNativeQuote {
		gas = p.gas.SellNative
	}

	if swapInfo.IsBuy {
		// The curve keeps everything but the fee: what the buyer paid for, and the surcharge
		// that raises the price for whoever comes next.
		tokensOut, fee, surcharge, err := p.curve.quoteBuy(&amountIn, nowUnix())
		if err != nil {
			return nil, err
		}
		amountOut.Set(&tokensOut)
		cost.Add(&fee, &surcharge)

		var nextQuote, nextToken uint256.Int
		nextQuote.Add(&p.curve.quoteReserve, &amountIn)
		nextQuote.Sub(&nextQuote, &fee)
		nextToken.Sub(&p.curve.tokenReserve, &tokensOut)
		swapInfo.NewQuoteReserve = &nextQuote
		swapInfo.NewTokenReserve = &nextToken
		// The buy that lifts the reserve to the target graduates the curve in the same
		// transaction. The buyer still gets these tokens; nobody trades on the curve after.
		swapInfo.NewGraduated = !nextQuote.Lt(&p.curve.graduationTarget)
		gas = p.gas.BuyERC20
		if p.isNativeQuote {
			gas = p.gas.BuyNative
		}
		if swapInfo.NewGraduated {
			gas += p.gas.Graduation
		}
	} else {
		quoteOut, fee, gross, err := p.curve.quoteSell(&amountIn)
		if err != nil {
			return nil, err
		}
		amountOut.Set(&quoteOut)
		cost.Set(&fee)

		var nextQuote, nextToken uint256.Int
		nextQuote.Sub(&p.curve.quoteReserve, &gross)
		if _, overflow := nextToken.AddOverflow(&p.curve.tokenReserve, &amountIn); overflow {
			return nil, ErrArithmetic
		}
		swapInfo.NewQuoteReserve = &nextQuote
		swapInfo.NewTokenReserve = &nextToken
	}

	amountOutBI := amountOut.ToBig()
	if amountOutBI.Cmp(p.Info.Reserves[indexOut]) > 0 {
		return nil, ErrInvalidReserve
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOutBI},
		// Always on the quote leg, whichever way the trade goes, as on chain.
		Fee:      &pool.TokenAmount{Token: p.Info.Tokens[0], Amount: cost.ToBig()},
		Gas:      gas,
		SwapInfo: swapInfo,
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Info.Reserves = lo.Map(p.Info.Reserves, func(r *big.Int, _ int) *big.Int { return new(big.Int).Set(r) })
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(*SwapInfo)
	if !ok || si == nil || si.NewQuoteReserve == nil || si.NewTokenReserve == nil {
		logger.Warn("failed to UpdateBalance for slyng-fun pool, wrong swapInfo type")
		return
	}

	p.curve.quoteReserve.Set(si.NewQuoteReserve)
	p.curve.tokenReserve.Set(si.NewTokenReserve)
	if si.NewGraduated {
		p.graduated = true
	}
	p.Info.Reserves = []*big.Int{p.curve.quoteReserve.ToBig(), p.curve.tokenReserve.ToBig()}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		Launchpad:       p.launchpad,
		Token:           p.Info.Tokens[1],
		ApprovalAddress: p.GetApprovalAddress(tokenIn, tokenOut),
		IsBuy:           p.GetTokenIndex(tokenOut) == 1,
		IsNativeQuote:   p.isNativeQuote,
		BlockNumber:     p.Info.BlockNumber,
	}
}

// GetApprovalAddress returns the launchpad: a buy pulls an ERC-20 quote and a sell pulls the coin,
// both with transferFrom. A buy paid in native ETH is sent as msg.value and needs no approval.
func (p *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	if p.isNativeQuote && p.GetTokenIndex(tokenIn) == 0 {
		return ""
	}
	return p.launchpad
}

func (p *PoolSimulator) SwapReceiveNativeIn(tokenIn, _ string, _ valueobject.ChainID) bool {
	return p.isNativeQuote && p.GetTokenIndex(tokenIn) == 0
}

func (p *PoolSimulator) SwapReturnNativeOut(_, tokenOut string, _ valueobject.ChainID) bool {
	return p.isNativeQuote && p.GetTokenIndex(tokenOut) == 0
}
