package flap

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	_                             = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ pool.IPoolExactOutSimulator = (*PoolSimulator)(nil)
)

// PoolSimulator simulates a single flap.sh bonding-curve token against Portal's LibCurve math.
// Tokens[0] is always the quote token, Tokens[1] the launched (base) token, matching the list updater.
//
// Two deductions apply, both cross-checked live on-chain and against PortalTradeV2's decompiled
// bytecode (source unverified; confirmed via dedaub decompilation of
// 0x153470d98e538d9e2761b3add795f908517aae48, the address resolved from the Portal implementation's
// own deployment calldata, not guessed):
//   - Portal's own protocol fee (getFeeRate: buyFeeBps/sellFeeBps, observed 100/100 = 1%/1%).
//   - The launched token's own transfer tax (getTokenV8: buyTaxRate/sellTaxRate), but only when
//     Portal.enableTaxOnBondingCurve() is true (observed true on-chain) - a global switch.
//
// Per the decompiled quote function, the two are combined into a single bps figure
// (10000 - (taxBps + feeBps)) and applied as one mulDiv - not as two sequential deductions - taken off
// the input for buys and off the output for sells (buySideDeductionBps/sellSideDeductionBps).
type PoolSimulator struct {
	pool.Pool

	status TokenStatus
	curve  Curve

	circulatingSupply *uint256.Int
	dexSupplyThresh   *uint256.Int

	buyFeeBps  uint64
	sellFeeBps uint64

	buyTaxBps                uint64
	sellTaxBps               uint64
	taxOnBondingCurveEnabled bool

	quoteDecimals uint8
	baseDecimals  uint8

	portalAddress string
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
	if extra.BuyFeeBps >= bpsDenominator || extra.SellFeeBps >= bpsDenominator ||
		extra.BuyTaxBps >= bpsDenominator || extra.SellTaxBps >= bpsDenominator ||
		extra.BuyFeeBps+extra.BuyTaxBps >= bpsDenominator || extra.SellFeeBps+extra.SellTaxBps >= bpsDenominator {
		return nil, ErrInvalidFeeBps
	}

	tokens := make([]string, len(entityPool.Tokens))
	reserves := make([]*big.Int, len(entityPool.Tokens))
	for i, t := range entityPool.Tokens {
		tokens[i] = t.Address
		reserves[i] = bigIntFromReserve(entityPool.Reserves, i)
	}

	quoteDecimals, baseDecimals := uint8(0), uint8(0)
	if len(entityPool.Tokens) > 0 {
		quoteDecimals = entityPool.Tokens[0].Decimals
	}
	if len(entityPool.Tokens) > 1 {
		baseDecimals = entityPool.Tokens[1].Decimals
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: reserves,
			},
		},
		status:                   extra.Status,
		curve:                    extra.Curve,
		circulatingSupply:        extra.CirculatingSupply,
		dexSupplyThresh:          extra.DexSupplyThresh,
		buyFeeBps:                extra.BuyFeeBps,
		sellFeeBps:               extra.SellFeeBps,
		buyTaxBps:                extra.BuyTaxBps,
		sellTaxBps:               extra.SellTaxBps,
		taxOnBondingCurveEnabled: extra.TaxOnBondingCurveEnabled,
		quoteDecimals:            quoteDecimals,
		baseDecimals:             baseDecimals,
		portalAddress:            staticExtra.PortalAddress,
	}, nil
}

func bigIntFromReserve(reserves entity.PoolReserves, i int) *big.Int {
	if i >= len(reserves) {
		return big.NewInt(0)
	}
	r, ok := new(big.Int).SetString(reserves[i], 10)
	if !ok {
		return big.NewInt(0)
	}
	return r
}

// buySideDeductionBps/sellSideDeductionBps return the single *combined* protocol-fee + token-tax bps
// for each direction, confirmed against PortalTradeV2's decompiled bytecode: the facet fetches both
// rates via one helper (taxRate, protocolFeeRate) and combines them as `10000 - (taxBps + feeBps)`
// before a single mulDiv - not two sequential deductions. Token tax is 0 unless
// taxOnBondingCurveEnabled.
func (s *PoolSimulator) buySideDeductionBps() uint64 {
	if s.taxOnBondingCurveEnabled {
		return s.buyFeeBps + s.buyTaxBps
	}
	return s.buyFeeBps
}

func (s *PoolSimulator) sellSideDeductionBps() uint64 {
	if s.taxOnBondingCurveEnabled {
		return s.sellFeeBps + s.sellTaxBps
	}
	return s.sellFeeBps
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.status != TokenStatusTradable {
		return nil, ErrPoolNotTradable
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrCurveUnderflow
	}

	isBuy := params.TokenAmountIn.Token == s.Info.Tokens[0]

	if isBuy {
		return s.calcBuy(amountIn)
	}
	return s.calcSell(amountIn)
}

// calcBuy: tokenIn is the quote token, tokenOut is the base token.
func (s *PoolSimulator) calcBuy(amountIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	deductionBps := s.buySideDeductionBps()
	amountInAfterFee := applyFeeDown(amountIn, deductionBps)

	currentReserve, err := estimateReserveV2(s.curve, s.circulatingSupply, s.quoteDecimals)
	if err != nil {
		return nil, err
	}

	newReserve := new(uint256.Int).Add(currentReserve, amountInAfterFee)
	newSupply, err := estimateSupplyV2(s.curve, newReserve, s.quoteDecimals)
	if err != nil {
		return nil, err
	}

	var remainingAmountIn *uint256.Int
	if newSupply.Gt(s.dexSupplyThresh) {
		newSupply = s.dexSupplyThresh

		cappedReserve, err := estimateReserveV2(s.curve, s.dexSupplyThresh, s.quoteDecimals)
		if err != nil {
			return nil, err
		}
		if cappedReserve.Lt(currentReserve) {
			return nil, ErrCurveUnderflow
		}
		usedAfterFee := new(uint256.Int).Sub(cappedReserve, currentReserve)

		// Back out the pre-fee amount that was actually consumed, rounding up so the fee/tax-adjusted
		// portion still clears usedAfterFee after the contract re-applies the same flooring.
		usedAmountIn := growForFeeUp(usedAfterFee, deductionBps)
		if usedAmountIn.Gt(amountIn) {
			usedAmountIn = amountIn
		}
		remainingAmountIn = new(uint256.Int).Sub(amountIn, usedAmountIn)
	}

	if newSupply.Lt(s.circulatingSupply) {
		return nil, ErrCurveUnderflow
	}
	tokensOut := new(uint256.Int).Sub(newSupply, s.circulatingSupply)
	if tokensOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	result := &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[1], Amount: tokensOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: new(big.Int).Sub(amountIn.ToBig(), amountInAfterFee.ToBig())},
		Gas:            defaultGas,
		SwapInfo:       SwapInfo{NewCirculatingSupply: newSupply, NewStatus: s.newStatusAfterBuy(newSupply)},
	}
	if remainingAmountIn != nil && !remainingAmountIn.IsZero() {
		result.RemainingTokenAmountIn = &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: remainingAmountIn.ToBig()}
	}

	return result, nil
}

// calcSell: tokenIn is the base token, tokenOut is the quote token.
func (s *PoolSimulator) calcSell(amountIn *uint256.Int) (*pool.CalcAmountOutResult, error) {
	if amountIn.Gt(s.circulatingSupply) {
		return nil, ErrInsufficientSupply
	}
	newSupply := new(uint256.Int).Sub(s.circulatingSupply, amountIn)

	currentReserve, err := estimateReserveV2(s.curve, s.circulatingSupply, s.quoteDecimals)
	if err != nil {
		return nil, err
	}
	newReserveRequired, err := estimateReserveV2(s.curve, newSupply, s.quoteDecimals)
	if err != nil {
		return nil, err
	}
	if newReserveRequired.Gt(currentReserve) {
		return nil, ErrCurveUnderflow
	}
	grossOut := new(uint256.Int).Sub(currentReserve, newReserveRequired)

	// Sell deducts the protocol fee and token tax as two independently-floored amounts (verified
	// bit-for-bit against Portal.quoteExactInput), unlike buy's single combined-bps deduction.
	protocolFeeAmount := floorFeeAmount(grossOut, s.sellFeeBps)
	taxAmount := new(uint256.Int)
	if s.taxOnBondingCurveEnabled {
		taxAmount = floorFeeAmount(grossOut, s.sellTaxBps)
	}
	amountOut := new(uint256.Int).Sub(grossOut, protocolFeeAmount)
	amountOut.Sub(amountOut, taxAmount)
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: new(big.Int).Sub(grossOut.ToBig(), amountOut.ToBig())},
		Gas:            defaultGas,
		SwapInfo:       SwapInfo{NewCirculatingSupply: newSupply, NewStatus: s.status},
	}, nil
}

// CalcAmountIn is the exact inverse of CalcAmountOut, gross-up rounded so it clears the requested
// TokenAmountOut after the same flooring CalcAmountOut applies. Backed by a small corrective loop
// (bounded, integer rounding rarely needs more than one correction) so
// CalcAmountOut(CalcAmountIn(x).TokenAmountIn) >= x always holds.
func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if s.status != TokenStatusTradable {
		return nil, ErrPoolNotTradable
	}

	amountOut, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow || amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	isBuy := params.TokenIn == s.Info.Tokens[0]

	var (
		amountIn *uint256.Int
		err      error
	)
	if isBuy {
		amountIn, err = s.calcAmountInForBuy(amountOut)
	} else {
		amountIn, err = s.calcAmountInForSell(amountOut)
	}
	if err != nil {
		return nil, err
	}

	// Corrective loop: CalcAmountOut floors at several points (fee/tax, LibCurve's own rounding), so
	// the direct inversion can occasionally undershoot by a wei-scale amount. Re-run the forward
	// direction and nudge amountIn up until it clears the target, bounded to avoid any pathological loop.
	const maxCorrections = 5
	for i := 0; i < maxCorrections; i++ {
		out, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: params.TokenIn, Amount: amountIn.ToBig()},
			TokenOut:      params.TokenAmountOut.Token,
		})
		if err == nil && out.TokenAmountOut.Amount.Cmp(amountOut.ToBig()) >= 0 {
			return &pool.CalcAmountInResult{
				TokenAmountIn: &pool.TokenAmount{Token: params.TokenIn, Amount: amountIn.ToBig()},
				Fee:           out.Fee,
				Gas:           defaultGas,
				SwapInfo:      out.SwapInfo,
			}, nil
		}
		amountIn = new(uint256.Int).AddUint64(amountIn, 1)
	}

	return nil, ErrZeroAmountOut
}

func (s *PoolSimulator) calcAmountInForBuy(tokensOut *uint256.Int) (*uint256.Int, error) {
	newSupply := new(uint256.Int).Add(s.circulatingSupply, tokensOut)
	if newSupply.Gt(s.dexSupplyThresh) {
		return nil, ErrSupplyExceedsTotalSupply
	}

	currentReserve, err := estimateReserveV2(s.curve, s.circulatingSupply, s.quoteDecimals)
	if err != nil {
		return nil, err
	}
	requiredReserve, err := estimateReserveV2(s.curve, newSupply, s.quoteDecimals)
	if err != nil {
		return nil, err
	}
	if requiredReserve.Lt(currentReserve) {
		return nil, ErrCurveUnderflow
	}
	netAmountIn := new(uint256.Int).Sub(requiredReserve, currentReserve)

	return growForFeeUp(netAmountIn, s.buySideDeductionBps()), nil
}

func (s *PoolSimulator) calcAmountInForSell(quoteAmountOut *uint256.Int) (*uint256.Int, error) {
	grossOut := growForFeeUp(quoteAmountOut, s.sellSideDeductionBps())

	currentReserve, err := estimateReserveV2(s.curve, s.circulatingSupply, s.quoteDecimals)
	if err != nil {
		return nil, err
	}
	if grossOut.Gt(currentReserve) {
		return nil, ErrCurveUnderflow
	}
	requiredReserve := new(uint256.Int).Sub(currentReserve, grossOut)

	newSupply, err := estimateSupplyV2(s.curve, requiredReserve, s.quoteDecimals)
	if err != nil {
		return nil, err
	}
	if newSupply.Gt(s.circulatingSupply) {
		return nil, ErrCurveUnderflow
	}
	return new(uint256.Int).Sub(s.circulatingSupply, newSupply), nil
}

// newStatusAfterBuy reports TokenStatusDEX once a buy fills the curve up to dexSupplyThresh. This
// mirrors the graduation trigger described in the docs (circulatingSupply reaching dexSupplyThresh);
// the exact on-chain post-graduation bookkeeping (pair creation, etc.) isn't modeled since routing
// only needs to know the pool stops being tradable through the curve from that point on.
func (s *PoolSimulator) newStatusAfterBuy(newSupply *uint256.Int) TokenStatus {
	if newSupply.Eq(s.dexSupplyThresh) {
		return TokenStatusDEX
	}
	return s.status
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	s.circulatingSupply = swapInfo.NewCirculatingSupply
	s.status = swapInfo.NewStatus
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.circulatingSupply = new(uint256.Int).Set(s.circulatingSupply)
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{ApprovalAddress: s.portalAddress}
}

func (s *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return s.portalAddress
}

// PoolMeta is exposed to encoding so it can build the executor calldata / know where to approve.
type PoolMeta struct {
	ApprovalAddress string `json:"approvalAddress"`
}
