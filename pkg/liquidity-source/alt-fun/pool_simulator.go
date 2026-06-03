package altfun

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	bouncetech "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bounce-tech"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	reserveToken *uint256.Int
	reserveAsset *uint256.Int
	k            *uint256.Int
	tokenBalance *uint256.Int

	buyFeeBps  *uint256.Int
	sellFeeBps *uint256.Int

	lifecycle              Lifecycle
	zapAddress             string
	ltAddress              string // bounce-tech LT token address (≠ meme token, tokens[1])
	graduationThresholdUsd *uint256.Int

	btPool pool.IPoolSimulator
}

var _ pool.IMetaPoolSimulator = (*PoolSimulator)(nil)

var _ = pool.RegisterFactoryMeta(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool, basePoolMap map[string]pool.IPoolSimulator) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	orZero := func(v *uint256.Int) *uint256.Int {
		if v == nil {
			return new(uint256.Int)
		}
		return v
	}

	ltAddr := strings.ToLower(staticExtra.LTAddress)
	btPool := basePoolMap[ltAddr]

	gradThresh := orZero(staticExtra.GraduationThresholdUsd)

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     strings.ToLower(ep.Address),
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: ep.BlockNumber,
		}},
		reserveToken:           orZero(extra.ReserveToken),
		reserveAsset:           orZero(extra.ReserveAsset),
		k:                      orZero(extra.K),
		tokenBalance:           orZero(extra.TokenBalance),
		buyFeeBps:              uint256.NewInt(staticExtra.BuyFeeBps),
		sellFeeBps:             uint256.NewInt(staticExtra.SellFeeBps),
		lifecycle:              extra.Lifecycle,
		zapAddress:             staticExtra.ZapAddress,
		ltAddress:              ltAddr,
		graduationThresholdUsd: gradThresh,
		btPool:                 btPool,
	}, nil
}

// GetBasePools implements IMetaPoolSimulator.
func (s *PoolSimulator) GetBasePools() []pool.IPoolSimulator {
	if s.btPool == nil {
		return nil
	}
	return []pool.IPoolSimulator{s.btPool}
}

// SetBasePool implements IMetaPoolSimulator — called by the routing engine when
// the bounce-tech base pool is refreshed.
func (s *PoolSimulator) SetBasePool(p pool.IPoolSimulator) {
	if p != nil && strings.EqualFold(p.GetAddress(), s.ltAddress) {
		s.btPool = p
	}
}

// CalcAmountOut computes buy (USDC→meme) or sell (meme→USDC).
// Token layout: tokens[0]=USDC, tokens[1]=meme.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	indexIn := s.GetTokenIndex(tokenAmountIn.Token)
	indexOut := s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if s.lifecycle != LifecycleCurve {
		return nil, ErrPoolGraduated
	}
	if s.k.IsZero() {
		return nil, ErrZeroK
	}
	if s.btPool == nil {
		return nil, ErrBasePoolNotFound
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	isBuy := indexIn == 0
	var (
		amountOut      *uint256.Int
		fee            *uint256.Int
		remainingAmtIn *pool.TokenAmount
		amountInUsed   *uint256.Int
		baseToConvert  *uint256.Int
		gas            int64
	)

	if isBuy {
		var err error
		amountOut, fee, remainingAmtIn, amountInUsed, baseToConvert, err = s.calcBuyOut(amountIn, tokenAmountIn.Token)
		if err != nil {
			return nil, err
		}
		gas = buyGas
	} else {
		var err error
		amountOut, fee, err = s.calcSellOut(amountIn)
		if err != nil {
			return nil, err
		}
		gas = sellGas
	}

	if amountOut.IsZero() {
		return nil, ErrZeroAmount
	}

	result := &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: fee.ToBig()},
		Gas:            gas,
		SwapInfo: SwapInfo{
			Pool:          s.zapAddress,
			IsSell:        !isBuy,
			AmountInUsed:  amountInUsed,
			BaseToConvert: baseToConvert,
		},
	}
	if remainingAmtIn != nil {
		result.RemainingTokenAmountIn = remainingAmtIn
	}
	return result, nil
}

// calcBuyOut: USDC → meme via Zap fee → BounceTech mint → bonding curve.
func (s *PoolSimulator) calcBuyOut(
	usdcIn *uint256.Int, tokenInAddr string,
) (*uint256.Int, *uint256.Int, *pool.TokenAmount, *uint256.Int, *uint256.Int, error) {
	if usdcIn.Lt(minUSDCU) {
		return nil, nil, nil, nil, nil, ErrBelowMinAmount
	}

	// Zap fee on gross
	feeOnGross := new(uint256.Int).Mul(usdcIn, s.buyFeeBps)
	feeOnGross.Div(feeOnGross, bpsDenomU)
	netUsdc := new(uint256.Int).Sub(usdcIn, feeOnGross)
	if netUsdc.Lt(minUSDCU) {
		return nil, nil, nil, nil, nil, ErrBelowMinAmount
	}

	// ltIfFull: delegate USDC→LT to bounce-tech base pool.
	ltIfFullResult, err := s.btPool.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[0], Amount: netUsdc.ToBig()},
		TokenOut:      s.ltAddress,
	})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	ltIfFull, overflow := uint256.FromBig(ltIfFullResult.TokenAmountOut.Amount)
	if overflow || ltIfFull.IsZero() {
		return nil, nil, nil, nil, nil, ErrZeroAmount
	}

	// Graduation pre-sizing: compute ltUntilGrad dynamically.
	ltUntilGrad := s.computeLtUntilGrad()

	baseToConvert := new(uint256.Int)
	if ltUntilGrad.IsZero() || ltUntilGrad.Cmp(ltIfFull) >= 0 {
		baseToConvert.Set(netUsdc)
	} else {
		// Pre-size mint to exactly ltUntilGrad.
		if !ltUntilGrad.IsZero() {
			// ltToBase: ltUntilGrad * exchangeRate / 1e18 / 1e12
			baseToConvert = s.ltToBaseViaPool(ltUntilGrad)
			// Floor-bump: ensure mint yields ≥ ltUntilGrad.
			check, _ := s.baseToLTViaPool(baseToConvert)
			if check != nil && check.Lt(ltUntilGrad) {
				baseToConvert.Add(baseToConvert, big256.U1)
			}
		}
		if baseToConvert.Gt(netUsdc) {
			baseToConvert.Set(netUsdc)
		}
		if baseToConvert.Lt(minUSDCU) {
			baseToConvert.Set(minUSDCU)
			if baseToConvert.Gt(netUsdc) {
				return nil, nil, nil, nil, nil, ErrBelowMinAmount
			}
		}
	}

	// Mint: delegate baseToConvert USDC → LT via bounce-tech.
	ltMintResult, err := s.btPool.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[0], Amount: baseToConvert.ToBig()},
		TokenOut:      s.ltAddress,
	})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	ltMinted, overflow := uint256.FromBig(ltMintResult.TokenAmountOut.Amount)
	if overflow || ltMinted.IsZero() {
		return nil, nil, nil, nil, nil, ErrZeroAmount
	}

	// Router._computeBuy
	newReserveAsset := new(uint256.Int).Add(s.reserveAsset, ltMinted)
	newReserveToken := new(uint256.Int).Div(s.k, newReserveAsset)
	tokensOut := new(uint256.Int).Sub(s.reserveToken, newReserveToken)
	amountInUsed := ltMinted.Clone()

	// Cap at actual token balance.
	if tokensOut.Gt(s.tokenBalance) {
		tokensOut.Set(s.tokenBalance)
		cappedReserveToken := new(uint256.Int).Sub(s.reserveToken, tokensOut)
		if cappedReserveToken.IsZero() {
			return nil, nil, nil, nil, nil, ErrInsufficientLiquidity
		}
		cappedReserveAsset := new(uint256.Int).Add(s.k, cappedReserveToken)
		cappedReserveAsset.Sub(cappedReserveAsset, big256.U1)
		cappedReserveAsset.Div(cappedReserveAsset, cappedReserveToken)
		amountInUsed.Sub(cappedReserveAsset, s.reserveAsset)
	}

	// Pro-rate fee.
	effectiveBaseSpent := new(uint256.Int).Mul(amountInUsed, baseToConvert)
	effectiveBaseSpent.Div(effectiveBaseSpent, ltMinted)
	actualFee := new(uint256.Int).Mul(usdcIn, s.buyFeeBps)
	actualFee.Mul(actualFee, effectiveBaseSpent)
	denom := new(uint256.Int).Mul(bpsDenomU, netUsdc)
	actualFee.Div(actualFee, denom)

	feeRefund := new(uint256.Int)
	if feeOnGross.Gt(actualFee) {
		feeRefund.Sub(feeOnGross, actualFee)
	}
	usdcLeft := new(uint256.Int)
	if netUsdc.Gt(baseToConvert) {
		usdcLeft.Sub(netUsdc, baseToConvert)
	}
	refund := new(uint256.Int).Add(usdcLeft, feeRefund)

	var remaining *pool.TokenAmount
	if !refund.IsZero() {
		remaining = &pool.TokenAmount{Token: tokenInAddr, Amount: refund.ToBig()}
	}

	return tokensOut, actualFee, remaining, amountInUsed, baseToConvert, nil
}

// calcSellOut: meme → USDC via bonding curve → BounceTech redeem → Zap fee.
func (s *PoolSimulator) calcSellOut(tokenIn *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	newReserveToken := new(uint256.Int).Add(s.reserveToken, tokenIn)
	newReserveToken.Div(s.k, newReserveToken) // reuse: newReserveToken = newReserveAsset
	if newReserveToken.Gt(s.reserveAsset) {
		return nil, nil, ErrInsufficientLiquidity
	}
	ltOut := new(uint256.Int).Sub(s.reserveAsset, newReserveToken)
	if ltOut.IsZero() {
		return nil, nil, ErrZeroAmount
	}

	// Delegate LT→USDC to bounce-tech base pool (handles btFee + liquidity check internally).
	btResult, err := s.btPool.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: s.ltAddress, Amount: ltOut.ToBig()},
		TokenOut:      s.Info.Tokens[0],
	})
	if err != nil {
		return nil, nil, err
	}
	grossUsdc, overflow := uint256.FromBig(btResult.TokenAmountOut.Amount)
	if overflow || grossUsdc.IsZero() {
		return nil, nil, ErrZeroAmount
	}
	if grossUsdc.Lt(minUSDCU) {
		return nil, nil, ErrBelowMinAmount
	}

	btFee, _ := uint256.FromBig(btResult.Fee.Amount)

	// Zap sell fee on grossUsdc (= what redeem() returns, i.e. after btFee).
	zapFee := new(uint256.Int).Mul(grossUsdc, s.sellFeeBps)
	zapFee.Div(zapFee, bpsDenomU)
	usdcOut := new(uint256.Int).Sub(grossUsdc, zapFee)

	totalFee := new(uint256.Int).Add(btFee, zapFee)
	return usdcOut, totalFee, nil
}

// computeLtUntilGrad computes Bonding.previewLtUntilGraduation dynamically using
// the bounce-tech base pool's current exchangeRate, avoiding a separate RPC call.
//
//	launchTimeVirtualLt = K / TOTAL_SUPPLY
//	realLtRaised        = reserveAsset - launchTimeVirtualLt
//	thresholdRealLt     = ceil(graduationThresholdUsd * 1e18 / exchangeRate)
//	ltUntilGrad         = thresholdRealLt - realLtRaised  (0 if already past threshold)
func (s *PoolSimulator) computeLtUntilGrad() *uint256.Int {
	if s.graduationThresholdUsd.IsZero() {
		return new(uint256.Int)
	}
	exchangeRate := s.btExchangeRate()
	if exchangeRate == nil || exchangeRate.IsZero() {
		return new(uint256.Int)
	}

	launchVirtual := new(uint256.Int).Div(s.k, memeTokenTotalSupply)
	if s.reserveAsset.Lt(launchVirtual) {
		return new(uint256.Int)
	}
	realLtRaised := new(uint256.Int).Sub(s.reserveAsset, launchVirtual)

	// thresholdRealLt = ceil(graduationThresholdUsd * 1e18 / exchangeRate)
	thresholdRealLt := new(uint256.Int).Mul(s.graduationThresholdUsd, precision)
	thresholdRealLt.Add(thresholdRealLt, exchangeRate)
	thresholdRealLt.Sub(thresholdRealLt, big256.U1)
	thresholdRealLt.Div(thresholdRealLt, exchangeRate)

	if realLtRaised.Cmp(thresholdRealLt) >= 0 {
		return new(uint256.Int)
	}
	thresholdRealLt.Sub(thresholdRealLt, realLtRaised)
	return thresholdRealLt
}

// btExchangeRate extracts the exchangeRate from the bounce-tech base pool.
func (s *PoolSimulator) btExchangeRate() *uint256.Int {
	type exchangeRateProvider interface {
		ExchangeRate() *uint256.Int
	}
	if erp, ok := s.btPool.(*bouncetech.PoolSimulator); ok {
		return erp.ExchangeRate()
	}
	return nil
}

// ltToBaseViaPool converts LT → USDC using bounce-tech base pool (approximate, no fee check).
func (s *PoolSimulator) ltToBaseViaPool(ltAmount *uint256.Int) *uint256.Int {
	er := s.btExchangeRate()
	if er == nil || er.IsZero() {
		return new(uint256.Int)
	}
	// ltToBaseAmount: ltAmount * exchangeRate / 1e18 / 1e12
	v := new(uint256.Int).Mul(ltAmount, er)
	v.Div(v, precision)
	v.Div(v, scaleUp)
	return v
}

// baseToLTViaPool converts USDC → LT (approximate, for pre-sizing check).
func (s *PoolSimulator) baseToLTViaPool(baseAmount *uint256.Int) (*uint256.Int, error) {
	res, err := s.btPool.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[0], Amount: baseAmount.ToBig()},
		TokenOut:      s.ltAddress,
	})
	if err != nil {
		return nil, err
	}
	lt, _ := uint256.FromBig(res.TokenAmountOut.Amount)
	return lt, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return
	}
	amountOut, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow {
		return
	}

	if !si.IsSell {
		ltIn := si.AmountInUsed
		baseToConvert := si.BaseToConvert
		if ltIn == nil || ltIn.IsZero() {
			// Fallback: approximate ltIn from amountIn (no graduation pre-sizing).
			fee := new(uint256.Int).Mul(amountIn, s.buyFeeBps)
			fee.Div(fee, bpsDenomU)
			netUsdc := new(uint256.Int).Sub(amountIn, fee)
			er := s.btExchangeRate()
			if er != nil && !er.IsZero() {
				scaled := new(uint256.Int).Mul(netUsdc, scaleUp)
				scaled.Mul(scaled, precision)
				ltIn = new(uint256.Int).Div(scaled, er)
			} else {
				ltIn = new(uint256.Int)
			}
			baseToConvert = netUsdc
		}

		s.reserveAsset.Add(s.reserveAsset, ltIn)
		s.reserveToken.Div(s.k, s.reserveAsset)
		if amountOut.Lt(s.tokenBalance) {
			s.tokenBalance.Sub(s.tokenBalance, amountOut)
		} else {
			s.tokenBalance.Clear()
		}
		_ = baseToConvert
		// Detect graduation: if ltUntilGrad reaches 0 after this buy, mark graduating.
		if s.btPool != nil && s.computeLtUntilGrad().IsZero() {
			s.lifecycle = LifecycleGraduating
		}
	} else {
		s.reserveToken.Add(s.reserveToken, amountIn)
		s.reserveAsset.Div(s.k, s.reserveToken)
		s.tokenBalance.Add(s.tokenBalance, amountIn)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	// Only clone fields mutated in UpdateBalance.
	// btPool, k, buyFeeBps, sellFeeBps, graduationThresholdUsd are read-only.
	cloned.reserveToken = s.reserveToken.Clone()
	cloned.reserveAsset = s.reserveAsset.Clone()
	cloned.tokenBalance = s.tokenBalance.Clone()
	cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return map[string]any{
		"blockNumber": s.Info.BlockNumber,
		"zapAddress":  s.zapAddress,
	}
}
