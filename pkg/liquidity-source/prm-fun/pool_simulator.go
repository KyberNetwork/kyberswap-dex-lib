package prmfun

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Token index convention: 0 = pair token (wrapped native for ETH), 1 = meme.
// All three pair types share curve math. Graduated pools belong to the v4 source.
const (
	indexDesk = 0
	indexMeme = 1
)

type PoolSimulator struct {
	pool.Pool

	routerAddress  string
	curveAddress   string
	graduationDesk *uint256.Int
	isNativeQuote  bool

	phase       uint8
	virtualMeme *uint256.Int
	virtualDesk *uint256.Int
	memeSold    *uint256.Int
	deskRaised  *uint256.Int
}

var (
	_                             = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ pool.IPoolSupportNativeSwap = (*PoolSimulator)(nil)
)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	graduationDesk, err := uint256.FromDecimal(staticExtra.GraduationDesk)
	if err != nil {
		return nil, err
	}

	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 || ep.Tokens[0] == nil || ep.Tokens[1] == nil ||
		!common.IsHexAddress(ep.Tokens[0].Address) || !common.IsHexAddress(ep.Tokens[1].Address) ||
		common.HexToAddress(ep.Tokens[0].Address) == (common.Address{}) || common.HexToAddress(ep.Tokens[1].Address) == (common.Address{}) ||
		strings.EqualFold(ep.Tokens[0].Address, ep.Tokens[1].Address) ||
		!strings.EqualFold(ep.Tokens[1].Address, staticExtra.MemeToken) ||
		!common.IsHexAddress(staticExtra.RouterAddress) || common.HexToAddress(staticExtra.RouterAddress) == (common.Address{}) ||
		!common.IsHexAddress(staticExtra.CurveAddress) || common.HexToAddress(staticExtra.CurveAddress) == (common.Address{}) ||
		!strings.EqualFold(ep.Address, staticExtra.CurveAddress) || !validState(extra, graduationDesk) {
		return nil, ErrInvalidState
	}
	reserves := make([]*big.Int, 2)
	for i, value := range ep.Reserves {
		amount, ok := new(big.Int).SetString(value, 10)
		if !ok || amount.Sign() < 0 || amount.BitLen() > 256 {
			return nil, ErrInvalidState
		}
		reserves[i] = amount
	}
	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    reserves,
			BlockNumber: ep.BlockNumber,
		}},
		routerAddress:  staticExtra.RouterAddress,
		curveAddress:   staticExtra.CurveAddress,
		graduationDesk: graduationDesk,
		isNativeQuote:  staticExtra.IsNativeQuote,
		phase:          extra.Phase,
		virtualMeme:    extra.VirtualMeme,
		virtualDesk:    extra.VirtualDesk,
		memeSold:       extra.MemeSold,
		deskRaised:     extra.DeskRaised,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}

	if s.phase != PhaseTrading {
		return nil, ErrPoolNotTrading
	}

	if tokenAmountIn.Amount == nil || tokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}
	if amountIn.IsZero() {
		return nil, ErrZeroAmount
	}

	isBuy := indexIn == indexDesk

	if isBuy {
		return s.calcBuy(tokenOut, amountIn)
	}
	return s.calcSell(tokenOut, amountIn)
}

func (s *PoolSimulator) calcBuy(tokenOut string, deskIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	// Required to avoid an underflow in math.go's room := graduationDesk - deskRaised; a
	// sold-out pool (memeSold == saleSupply) is caught below via result.MemeOut.IsZero().
	if s.deskRaised.Cmp(s.graduationDesk) >= 0 {
		return nil, ErrPoolNotTrading
	}

	// Solidity checks virtualDesk + accepted net input for overflow. This also
	// bounds the quote denominator before the pure math helpers run.
	feeOnAll := big256.MulDivUp(new(uint256.Int), deskIn, uFeeBps, uBps)
	net := new(uint256.Int).Sub(deskIn, feeOnAll)
	room := new(uint256.Int).Sub(s.graduationDesk, s.deskRaised)
	if net.Gt(room) {
		net.Set(room)
	}
	var nextDesk uint256.Int
	if _, overflow := nextDesk.AddOverflow(s.virtualDesk, net); overflow {
		return nil, ErrOverflow
	}
	result := QuoteBuy(deskIn, s.virtualMeme, s.virtualDesk, s.memeSold, s.deskRaised, s.graduationDesk, uSaleSupply)
	if result.MemeOut.IsZero() || result.DeskUsed.IsZero() {
		return nil, ErrInvalidAmount
	}

	var remainingIn *pool.TokenAmount
	if deskIn.Gt(result.DeskUsed) {
		remainingIn = &pool.TokenAmount{
			Token:  s.Info.Tokens[indexDesk],
			Amount: new(uint256.Int).Sub(deskIn, result.DeskUsed).ToBig(),
		}
	}

	next := s.stateCopy()
	net.Sub(result.DeskUsed, result.Fee)
	next.DeskRaised.Add(next.DeskRaised, net)
	next.VirtualDesk.Set(&nextDesk)
	next.VirtualMeme.Sub(next.VirtualMeme, result.MemeOut)
	next.MemeSold.Add(next.MemeSold, result.MemeOut)
	gas := int64(buyGas)
	if next.DeskRaised.Eq(s.graduationDesk) {
		next.Phase = PhaseGraduated
		gas += graduationGas
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: tokenOut, Amount: result.MemeOut.ToBig()},
		RemainingTokenAmountIn: remainingIn,
		Fee:                    &pool.TokenAmount{Token: s.Info.Tokens[indexDesk], Amount: result.Fee.ToBig()},
		Gas:                    gas,
		SwapInfo: &SwapInfo{
			IsBuy: true, CurveAddress: s.curveAddress, IsNativeQuote: s.isNativeQuote, NewState: next,
		},
	}, nil
}

func (s *PoolSimulator) calcSell(tokenOut string, memeIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	if memeIn.Gt(s.memeSold) {
		return nil, ErrInsufficientLiquidity
	}

	var nextMeme uint256.Int
	if _, overflow := nextMeme.AddOverflow(s.virtualMeme, memeIn); overflow {
		return nil, ErrOverflow
	}
	gross, result := QuoteSell(memeIn, s.virtualMeme, s.virtualDesk)
	if gross.Gt(s.deskRaised) {
		// Mirrors MemeCurve.sol's quoteSell returning (0, 0) here - the curve does not
		// have enough recorded desk-token principal to honor this sell.
		return nil, ErrInsufficientLiquidity
	}
	if result.DeskOut.IsZero() {
		return nil, ErrInvalidAmount
	}

	next := s.stateCopy()
	next.MemeSold.Sub(next.MemeSold, memeIn)
	next.VirtualMeme.Set(&nextMeme)
	next.VirtualDesk.Sub(next.VirtualDesk, gross)
	next.DeskRaised.Sub(next.DeskRaised, gross)
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: result.DeskOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[indexDesk], Amount: result.Fee.ToBig()},
		Gas:            sellGas,
		SwapInfo: &SwapInfo{
			IsBuy: false, CurveAddress: s.curveAddress, IsNativeQuote: s.isNativeQuote, NewState: next,
		},
	}, nil
}

func validState(e Extra, target *uint256.Int) bool {
	return target != nil && !target.IsZero() && e.Phase <= PhasePaused &&
		e.VirtualMeme != nil && !e.VirtualMeme.IsZero() && e.VirtualDesk != nil && !e.VirtualDesk.IsZero() &&
		e.MemeSold != nil && e.MemeSold.Cmp(uSaleSupply) <= 0 && e.DeskRaised != nil && e.DeskRaised.Cmp(target) <= 0 &&
		e.VirtualDesk.Cmp(e.DeskRaised) >= 0 &&
		e.VirtualMeme.Cmp(new(uint256.Int).Sub(uSaleSupply, e.MemeSold)) >= 0
}

func (s *PoolSimulator) stateCopy() Extra {
	return Extra{Phase: s.phase, VirtualMeme: s.virtualMeme.Clone(), VirtualDesk: s.virtualDesk.Clone(),
		MemeSold: s.memeSold.Clone(), DeskRaised: s.deskRaised.Clone()}
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(*SwapInfo)
	if !ok || info == nil || info.CurveAddress != s.curveAddress || !validState(info.NewState, s.graduationDesk) {
		return
	}
	// Never use the caller's offered input: the final buy may have a refund.
	// Copy the quote's result so neither clones nor later updates mutate SwapInfo.
	e := info.NewState
	s.phase = e.Phase
	s.virtualMeme, s.virtualDesk = e.VirtualMeme.Clone(), e.VirtualDesk.Clone()
	s.memeSold, s.deskRaised = e.MemeSold.Clone(), e.DeskRaised.Clone()
	s.Info.Reserves = []*big.Int{s.virtualDesk.ToBig(), s.virtualMeme.ToBig()}
	if s.phase != PhaseTrading {
		s.Info.Reserves = []*big.Int{new(big.Int), new(big.Int)}
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Tokens = append([]string(nil), s.Info.Tokens...)
	cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}
	cloned.virtualMeme, cloned.virtualDesk = s.virtualMeme.Clone(), s.virtualDesk.Clone()
	cloned.memeSold, cloned.deskRaised = s.memeSold.Clone(), s.deskRaised.Clone()
	cloned.graduationDesk = s.graduationDesk.Clone()
	return &cloned
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, tokenOut string, chainID valueobject.ChainID) bool {
	return s.isNativeQuote && s.GetTokenIndex(tokenIn) == indexDesk && s.GetTokenIndex(tokenOut) == indexMeme && valueobject.IsWrappedNative(tokenIn, chainID)
}

func (s *PoolSimulator) SwapReturnNativeOut(tokenIn, tokenOut string, chainID valueobject.ChainID) bool {
	return s.isNativeQuote && s.GetTokenIndex(tokenOut) == indexDesk && s.GetTokenIndex(tokenIn) == indexMeme && valueobject.IsWrappedNative(tokenOut, chainID)
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		ApprovalAddress: s.routerAddress,
		BlockNumber:     s.Info.BlockNumber,
		IsNativeQuote:   s.isNativeQuote,
	}
}
