package ponsv2

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// PoolSimulator ports PonsV2BondingCurve.buy()/sell() exactly, including the
// buy-side partial-fill-and-refund clamp against sellableTokens() and the
// two-part graduation gate (graduated flag OR sellableTokens()==0).
//
// Token order: Tokens[0] is the quote asset (wrapped native when
// IsNativeQuote, otherwise the ERC-20 pairToken), Tokens[1] is the launched
// memecoin. A swap with indexIn==0 is a buy (quote -> token); indexIn==1 is
// a sell (token -> quote).
type PoolSimulator struct {
	pool.Pool

	feeBps         uint64
	creatorTaxBps  uint64
	reservedTokens *uint256.Int
	isNativeQuote  bool

	quoteReserve *uint256.Int
	tokenReserve *uint256.Int
	graduated    bool
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

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, _ int) *big.Int { return bignum.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},

		feeBps:         uint64(staticExtra.FeeBps),
		creatorTaxBps:  uint64(staticExtra.CreatorTaxBps),
		reservedTokens: staticExtra.ReservedTokens,
		isNativeQuote:  staticExtra.IsNativeQuote,

		quoteReserve: extra.QuoteReserve,
		tokenReserve: extra.TokenReserve,
		graduated:    extra.Graduated,
	}, nil
}

func (s *PoolSimulator) sellableTokens() *uint256.Int {
	if s.tokenReserve.Cmp(s.reservedTokens) <= 0 {
		return new(uint256.Int)
	}
	return new(uint256.Int).Sub(s.tokenReserve, s.reservedTokens)
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}

	// Reject non-positive amounts against the original big.Int, not the
	// converted uint256: uint256.FromBig reads big.Int.Bits(), which is
	// sign-agnostic, so a negative amount would otherwise silently convert to
	// its positive magnitude instead of being caught here.
	if tokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	isBuy := indexIn == 0

	if s.graduated {
		return nil, ErrPoolGraduated
	}

	var (
		amountOut       *uint256.Int
		remainingAmount *uint256.Int
		totalFee        *uint256.Int
		swapInfo        *SwapInfo
		gas             int64
		err             error
	)
	if isBuy {
		amountOut, remainingAmount, totalFee, swapInfo, gas, err = s.buy(amountIn)
	} else {
		// sell() reverts when graduated OR readyToGraduate() (sellableTokens()==0
		// even before the flag flips); graduated is already handled above.
		if s.sellableTokens().IsZero() {
			return nil, ErrPoolGraduated
		}
		amountOut, totalFee, swapInfo, gas, err = s.sell(amountIn)
	}
	if err != nil {
		return nil, err
	}

	result := &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		// feeBps + creatorTaxBps are always charged on the quote leg
		// (Tokens[0]) on both buy and sell; totalFee is the exact amount
		// deducted for the swap actually executed (post partial-fill clamp
		// on a buy).
		Fee:      &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: totalFee.ToBig()},
		Gas:      gas,
		SwapInfo: swapInfo,
	}
	if remainingAmount != nil && !remainingAmount.IsZero() {
		result.RemainingTokenAmountIn = &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: remainingAmount.ToBig(),
		}
	}
	return result, nil
}

// buy ports PonsV2BondingCurve.buy() exactly (fee/tax deducted from the
// quote input before curve pricing, then clamped to sellableTokens() with
// the input re-derived via getAmountIn and grossed back up through the fee
// rate; the unspent remainder is a refund, never a revert).
func (s *PoolSimulator) buy(
	received *uint256.Int,
) (tokensOut, refund, totalFee *uint256.Int, info *SwapInfo, gas int64, err error) {
	if s.feeBps+s.creatorTaxBps > maxTotalTradeFeeBps {
		return nil, nil, nil, nil, 0, ErrInvalidFeeConfig
	}

	quoteReserveBefore := s.quoteReserve
	tokenReserveBefore := s.tokenReserve

	spent := new(uint256.Int).Set(received)
	fee, err := bpsOf(spent, s.feeBps)
	if err != nil {
		return nil, nil, nil, nil, 0, err
	}
	tax, err := bpsOf(spent, s.creatorTaxBps)
	if err != nil {
		return nil, nil, nil, nil, 0, err
	}

	netIn := new(uint256.Int).Sub(spent, fee)
	netIn.Sub(netIn, tax)

	tokensOut, err = getAmountOut(netIn, quoteReserveBefore, tokenReserveBefore, 0)
	if err != nil {
		return nil, nil, nil, nil, 0, err
	}

	sellable := s.sellableTokens()
	if sellable.IsZero() {
		return nil, nil, nil, nil, 0, ErrPoolGraduated
	}

	if tokensOut.Cmp(sellable) > 0 {
		tokensOut = sellable

		net, err := getAmountIn(sellable, quoteReserveBefore, tokenReserveBefore, 0)
		if err != nil {
			return nil, nil, nil, nil, 0, err
		}

		// spent = min(ceil(net * BASIS_POINTS / (BASIS_POINTS - feeBps - creatorTaxBps)), received)
		denom := basisPoints - s.feeBps - s.creatorTaxBps
		if denom == 0 {
			return nil, nil, nil, nil, 0, ErrInsufficientLiquidity
		}
		grossed, err := grossUpCeil(net, denom)
		if err != nil {
			return nil, nil, nil, nil, 0, err
		}
		if grossed.Cmp(received) < 0 {
			spent = grossed
		} else {
			spent = new(uint256.Int).Set(received)
		}

		fee, err = bpsOf(spent, s.feeBps)
		if err != nil {
			return nil, nil, nil, nil, 0, err
		}
		tax, err = bpsOf(spent, s.creatorTaxBps)
		if err != nil {
			return nil, nil, nil, nil, 0, err
		}
	}

	newQuoteReserve, overflow := new(uint256.Int).AddOverflow(quoteReserveBefore, spent)
	if overflow {
		return nil, nil, nil, nil, 0, ErrOverflow
	}
	newQuoteReserve.Sub(newQuoteReserve, fee)
	newQuoteReserve.Sub(newQuoteReserve, tax)

	newTokenReserve := new(uint256.Int).Sub(tokenReserveBefore, tokensOut)

	refund = new(uint256.Int).Sub(received, spent)
	totalFee = new(uint256.Int).Add(fee, tax)

	return tokensOut, refund, totalFee, &SwapInfo{
		IsBuy:           true,
		IsNativeQuote:   s.isNativeQuote,
		NewQuoteReserve: newQuoteReserve,
		NewTokenReserve: newTokenReserve,
	}, buyGas, nil
}

// sell ports PonsV2BondingCurve.sell() exactly (fee/tax deducted from the
// quote output after curve pricing on the full input, no partial fill).
func (s *PoolSimulator) sell(
	tokensIn *uint256.Int,
) (quoteOut, totalFee *uint256.Int, info *SwapInfo, gas int64, err error) {
	quoteReserveBefore := s.quoteReserve
	tokenReserveBefore := s.tokenReserve

	grossQuoteOut, err := getAmountOut(tokensIn, tokenReserveBefore, quoteReserveBefore, 0)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	fee, err := bpsOf(grossQuoteOut, s.feeBps)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	tax, err := bpsOf(grossQuoteOut, s.creatorTaxBps)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	quoteOut = new(uint256.Int).Sub(grossQuoteOut, fee)
	quoteOut.Sub(quoteOut, tax)
	if quoteOut.IsZero() {
		return nil, nil, nil, 0, ErrInsufficientOutputAmount
	}

	if quoteReserveBefore.Cmp(grossQuoteOut) < 0 {
		return nil, nil, nil, 0, ErrInvalidReserve
	}
	newQuoteReserve := new(uint256.Int).Sub(quoteReserveBefore, grossQuoteOut)

	newTokenReserve, overflow := new(uint256.Int).AddOverflow(tokenReserveBefore, tokensIn)
	if overflow {
		return nil, nil, nil, 0, ErrOverflow
	}

	totalFee = new(uint256.Int).Add(fee, tax)

	return quoteOut, totalFee, &SwapInfo{
		IsBuy:           false,
		IsNativeQuote:   s.isNativeQuote,
		NewQuoteReserve: newQuoteReserve,
		NewTokenReserve: newTokenReserve,
	}, sellGas, nil
}

// grossUpCeil computes ceil(amount * BASIS_POINTS / denomBps), matching
// Solidity's `Math.mulDiv(net, BASIS_POINTS, BASIS_POINTS - feeBps - creatorTaxBps, Math.Rounding.Ceil)`.
func grossUpCeil(amount *uint256.Int, denomBps uint64) (*uint256.Int, error) {
	product, overflow := new(uint256.Int).MulOverflow(amount, u256.UBasisPoint)
	if overflow {
		return nil, ErrOverflow
	}
	denom := new(uint256.Int).SetUint64(denomBps)
	res := new(uint256.Int)
	div, mod := res.DivMod(product, denom, new(uint256.Int))
	if !mod.IsZero() {
		div.AddUint64(div, 1)
	}
	return div, nil
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{BlockNumber: s.Info.BlockNumber}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	// Native-in buys never need an allowance; every ERC-20 leg (quote-in buy,
	// or the token-in sell) is pulled directly by the curve contract itself
	// via safeTransferFrom, so the curve's own address is the spender.
	if s.isNativeQuote && s.GetTokenIndex(tokenIn) == 0 {
		return ""
	}
	return s.Info.Address
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.quoteReserve = new(uint256.Int).Set(s.quoteReserve)
	cloned.tokenReserve = new(uint256.Int).Set(s.tokenReserve)
	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(*SwapInfo)
	if !ok || swapInfo == nil {
		return
	}

	s.quoteReserve = swapInfo.NewQuoteReserve
	s.tokenReserve = swapInfo.NewTokenReserve
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, _ string, _ valueobject.ChainID) bool {
	return s.isNativeQuote && s.GetTokenIndex(tokenIn) == 0
}

func (s *PoolSimulator) SwapReturnNativeOut(_, tokenOut string, _ valueobject.ChainID) bool {
	return s.isNativeQuote && s.GetTokenIndex(tokenOut) == 0
}
