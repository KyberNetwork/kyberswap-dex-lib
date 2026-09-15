package curve

import (
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// NowFn is a var so tests can pin the anti-sniper decay clock.
var NowFn = func() int64 { return time.Now().Unix() }

// PoolSimulator ports ArcadeHook.buy / sell on a PUMP launch still on its bonding
// curve (ArcadeV4Curve math, the 1% curve fee, the decaying anti-sniper token
// haircut on buys, and the graduation cap with its partial fill).
//
// Token order: Tokens[0] is USDC, Tokens[1] is the launch token. indexIn == 0 is a
// buy, indexIn == 1 a sell.
type PoolSimulator struct {
	pool.Pool

	hook  string
	extra Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	if extra.TokensSold == nil {
		extra.TokensSold = new(uint256.Int)
	}
	if extra.RealUsdcReserve == nil {
		extra.RealUsdcReserve = new(uint256.Int)
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, _ int) *big.Int { return bignum.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},
		hook:  staticExtra.Hook,
		extra: extra,
	}, nil
}

func (s *PoolSimulator) tradable() error {
	switch {
	case !s.extra.Tracked:
		return ErrNotTracked
	case s.extra.Paused:
		return ErrPaused
	case s.extra.Mode != modePump || s.extra.Status != statusCurving:
		// Direct launches and graduated / graduating curves trade on the V4 pool.
		return ErrNotCurving
	}
	return nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}
	if err := s.tradable(); err != nil {
		return nil, err
	}
	if indexIn == 0 {
		return s.buy(amountIn, params.TokenOut)
	}
	return s.sell(amountIn, params.TokenOut)
}

// buy mirrors ArcadeHook._doCurveBuy with applyTax = true (the public entrypoint).
func (s *PoolSimulator) buy(amountIn *uint256.Int, tokenOut string) (*pool.CalcAmountOutResult, error) {
	r := SimulateBuy(s.extra.TokensSold, s.extra.RealUsdcReserve, amountIn)
	if r.TokensOut.IsZero() {
		return nil, ErrZeroOutput
	}

	netTokensOut := r.TokensOut
	if bps := s.extra.CurrentSnipeBps(NowFn()); bps > 0 {
		tax := new(uint256.Int).Mul(r.TokensOut, uint256.NewInt(bps))
		tax.Div(tax, uint256.NewInt(basisPoints))
		netTokensOut = new(uint256.Int).Sub(r.TokensOut, tax)
	}

	newTokensSold := new(uint256.Int).Add(s.extra.TokensSold, r.TokensOut)
	newReal := new(uint256.Int).Add(s.extra.RealUsdcReserve, r.ActualGross)
	newReal.Sub(newReal, r.Fee)
	graduates := newTokensSold.Cmp(curveSupply) >= 0

	gas := int64(buyGas)
	if graduates {
		gas = graduatingBuyGas
	}

	result := &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: netTokensOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: r.Fee.ToBig()},
		Gas:            gas,
		SwapInfo: &SwapInfo{
			IsBuy:              true,
			Hook:               s.hook,
			Token:              s.Info.Tokens[1],
			AmountIn:           r.ActualGross.ToBig(),
			NewTokensSold:      newTokensSold,
			NewRealUsdcReserve: newReal,
			Graduates:          graduates,
		},
	}
	if !r.Refund.IsZero() {
		result.RemainingTokenAmountIn = &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: r.Refund.ToBig()}
	}
	return result, nil
}

// sell mirrors ArcadeHook.sell.
func (s *PoolSimulator) sell(tokensIn *uint256.Int, tokenOut string) (*pool.CalcAmountOutResult, error) {
	// The hook subtracts tokensIn from tokensSold with checked arithmetic, so a sell
	// larger than the curve's issuance reverts even though simulateSell clips it.
	if tokensIn.Cmp(s.extra.TokensSold) > 0 {
		return nil, ErrSellExceedsSold
	}
	r := SimulateSell(s.extra.TokensSold, s.extra.RealUsdcReserve, tokensIn)
	if r.UsdcOut.IsZero() {
		return nil, ErrZeroOutput
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: r.UsdcOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: r.Fee.ToBig()},
		Gas:            sellGas,
		SwapInfo: &SwapInfo{
			IsBuy:              false,
			Hook:               s.hook,
			Token:              s.Info.Tokens[1],
			AmountIn:           tokensIn.ToBig(),
			NewTokensSold:      new(uint256.Int).Sub(s.extra.TokensSold, tokensIn),
			NewRealUsdcReserve: new(uint256.Int).Sub(s.extra.RealUsdcReserve, r.GrossOut),
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(*SwapInfo)
	if !ok || info == nil || info.NewTokensSold == nil {
		return
	}
	s.extra.TokensSold = info.NewTokensSold
	s.extra.RealUsdcReserve = info.NewRealUsdcReserve
	if info.Graduates {
		// The graduating buy moves the launch to its V4 pool in the same transaction.
		s.extra.Status = statusGraduated
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.extra.TokensSold = new(uint256.Int).Set(s.extra.TokensSold)
	cloned.extra.RealUsdcReserve = new(uint256.Int).Set(s.extra.RealUsdcReserve)
	return &cloned
}

// GetApprovalAddress: the hook pulls both legs itself with transferFrom.
func (s *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return s.hook
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{BlockNumber: s.Info.BlockNumber}
}
