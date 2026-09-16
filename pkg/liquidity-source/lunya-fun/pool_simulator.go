package lunyafun

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
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

// nowUnix is the clock the anti-snipe surcharge is read against.
var nowUnix = func() uint64 { return uint64(time.Now().Unix()) }

// PoolSimulator prices a launch's bonding curve. Token 0 is the quote token the launch is paid in and
// token 1 the token it sells, so a swap into token 1 is a buy and the other way a sell.
//
// A buy is priced for a recipient the launch does not exempt, which is the worst case: an exempt
// recipient - the creator, or an address named in the launch's parameters - pays no surcharge and
// receives more than this quotes, never less.
type PoolSimulator struct {
	pool.Pool

	phase uint8
	curve curveState

	gas Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	for _, v := range []*uint256.Int{extra.Reserve, extra.Sold, extra.VirtualQuote, extra.VirtualToken,
		extra.CurveSupply} {
		if v == nil {
			return nil, ErrFailedCall
		}
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig10(r) }),
			SwapFee:     big.NewInt(int64(extra.CurveFeeBps)),
			BlockNumber: entityPool.BlockNumber,
		}},
		phase: extra.Phase,
		curve: curveState{
			reserve:      *extra.Reserve,
			sold:         *extra.Sold,
			virtualQuote: *extra.VirtualQuote,
			virtualToken: *extra.VirtualToken,
			curveSupply:  *extra.CurveSupply,
			curveFeeBps:  uint64(extra.CurveFeeBps),
			snipeTaxBps:  uint64(extra.SnipeTaxBps),
			snipeWindow:  uint64(extra.SnipeWindow),
			snipeDecay:   uint64(extra.SnipeDecay),
			openedAt:     extra.OpenedAt,
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
	if p.phase != phaseTrading {
		return nil, ErrNotTrading
	}

	var amountIn uint256.Int
	if overflow := amountIn.SetFromBig(params.TokenAmountIn.Amount); overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}

	var (
		amountOut uint256.Int
		remaining uint256.Int
		swapInfo  SwapInfo
	)

	if indexIn == 0 {
		// a buy: the curve takes the quote token, keeps the net of the fee, and hands back whatever it
		// could not spend once its supply ran out
		tokensOut, fee, refund, err := p.curve.quoteBuy(&amountIn, nowUnix())
		if err != nil {
			return nil, err
		}
		if tokensOut.IsZero() {
			return nil, ErrZeroAmount
		}
		amountOut.Set(&tokensOut)
		remaining.Set(&refund)

		var net, nextReserve, nextSold uint256.Int
		net.Sub(&amountIn, &fee)
		net.Sub(&net, &refund)
		nextReserve.Add(&p.curve.reserve, &net)
		nextSold.Add(&p.curve.sold, &tokensOut)
		swapInfo = SwapInfo{NextReserve: &nextReserve, NextSold: &nextSold}
	} else {
		// a sell: the curve pays the quote token out of its reserve, fee included
		quoteOut, fee, err := p.curve.quoteSell(&amountIn)
		if err != nil {
			return nil, err
		}
		if quoteOut.IsZero() {
			return nil, ErrZeroAmount
		}
		amountOut.Set(&quoteOut)

		var gross, nextReserve, nextSold uint256.Int
		gross.Add(&quoteOut, &fee)
		if p.curve.reserve.Lt(&gross) {
			return nil, ErrInsufficientReserve
		}
		nextReserve.Sub(&p.curve.reserve, &gross)
		nextSold.Sub(&p.curve.sold, &amountIn)
		swapInfo = SwapInfo{NextReserve: &nextReserve, NextSold: &nextSold}
	}

	amountOutBI := amountOut.ToBig()
	if amountOutBI.Cmp(p.Info.Reserves[indexOut]) > 0 {
		return nil, ErrInsufficientReserve
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: tokenOut, Amount: amountOutBI},
		Fee:                    &pool.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: remaining.ToBig()},
		Gas:                    p.gas.BaseGas,
		SwapInfo:               swapInfo,
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok || si.NextReserve == nil || si.NextSold == nil {
		logger.Warn("failed to UpdateBalance for lunya-fun pool, wrong swapInfo type")
		return
	}

	p.curve.reserve.Set(si.NextReserve)
	p.curve.sold.Set(si.NextSold)

	// the reserves are what each side can still pay out: the quote held, and the tokens left to sell
	var tokensLeft uint256.Int
	if p.curve.curveSupply.Gt(&p.curve.sold) {
		tokensLeft.Sub(&p.curve.curveSupply, &p.curve.sold)
	}
	p.Info.Reserves = []*big.Int{p.curve.reserve.ToBig(), tokensLeft.ToBig()}
}

func (p *PoolSimulator) GetMetaInfo(_, tokenOut string) any {
	return PoolMeta{
		ApprovalAddress: p.Info.Address,
		IsBuy:           p.GetTokenIndex(tokenOut) == 1,
		BlockNumber:     p.Info.BlockNumber,
	}
}

// GetApprovalAddress returns the launch itself: it pulls both legs with transferFrom.
func (p *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return p.Info.Address
}
