package bouncetech

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	exchangeRate   *uint256.Int
	redemptionFee  *uint256.Int
	targetLeverage *uint256.Int
	minTxSize      *uint256.Int
	mintPaused     bool
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	exchangeRate := extra.ExchangeRate
	if exchangeRate == nil {
		exchangeRate = new(uint256.Int)
	}
	redemptionFee := extra.RedemptionFee
	if redemptionFee == nil {
		redemptionFee = new(uint256.Int)
	}
	targetLeverage := extra.TargetLeverage
	if targetLeverage == nil {
		targetLeverage = precision // default 1x (1e18)
	}
	minTxSize := extra.MinTransactionSize
	if minTxSize == nil {
		minTxSize = new(uint256.Int)
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     strings.ToLower(ep.Address),
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: ep.BlockNumber,
		}},
		exchangeRate:   exchangeRate,
		redemptionFee:  redemptionFee,
		targetLeverage: targetLeverage,
		minTxSize:      minTxSize,
		mintPaused:     extra.MintPaused,
	}, nil
}

// CalcAmountOut computes the output for a mint (USDC→LT) or redeem (LT→USDC) swap.
//
// Token layout: tokens[0]=USDC, tokens[1]=LT.
//   - indexIn==0 → mint:   ltOut  = usdcIn * 1e30 / exchangeRate
//   - indexIn==1 → redeem: usdcOut = ltIn * exchangeRate / 1e30 - redemptionFee
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	indexIn := s.GetTokenIndex(tokenAmountIn.Token)
	indexOut := s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}
	if tokenAmountIn.Amount == nil || tokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	if s.exchangeRate.IsZero() {
		return nil, ErrZeroExchangeRate
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrZeroAmount
	}

	isMint := indexIn == 0
	if isMint && s.mintPaused {
		return nil, ErrMintPaused
	}

	var amountOut *uint256.Int
	var fee *uint256.Int
	var grossAmountOut *uint256.Int
	var gas int64

	if isMint {
		if !s.minTxSize.IsZero() && amountIn.Lt(s.minTxSize) {
			return nil, ErrBelowMinAmount
		}
		amountOut, fee = s.calcMintOut(amountIn)
		gas = mintGas
	} else {
		grossAmountOut, amountOut, fee = s.calcRedeemOut(amountIn)
		gas = redeemGas

		if !s.minTxSize.IsZero() && grossAmountOut.Lt(s.minTxSize) {
			return nil, ErrBelowMinAmount
		}

		// The LT contract checks the gross base amount before paying redemption fees.
		reserve := s.Info.Reserves[0] // USDC reserve = baseAssetBalance
		if grossAmountOut.ToBig().Cmp(reserve) > 0 {
			return nil, ErrInsufficientBalance
		}
	}

	if amountOut.IsZero() {
		return nil, ErrZeroAmount
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: fee.ToBig()},
		Gas:            gas,
		SwapInfo:       SwapInfo{IsMint: isMint},
	}, nil
}

// CalcAmountIn computes the input required for a mint (USDC→LT) or redeem (LT→USDC) swap
// to produce at least tokenAmountOut. It inverts the flooring done by CalcAmountOut, so the
// returned amountIn is rounded up (ceiling) to guarantee the requested output is met.
func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut, tokenIn := params.TokenAmountOut, params.TokenIn

	indexIn := s.GetTokenIndex(tokenIn)
	indexOut := s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}
	if tokenAmountOut.Amount == nil || tokenAmountOut.Amount.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	if s.exchangeRate.IsZero() {
		return nil, ErrZeroExchangeRate
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrZeroAmount
	}

	isMint := indexIn == 0
	if isMint && s.mintPaused {
		return nil, ErrMintPaused
	}

	var amountIn *uint256.Int
	var fee *uint256.Int
	var gas int64

	if isMint {
		amountIn = s.calcMintIn(amountOut)
		fee = new(uint256.Int)
		gas = mintGas

		if !s.minTxSize.IsZero() && amountIn.Lt(s.minTxSize) {
			return nil, ErrBelowMinAmount
		}
	} else {
		grossUsdc, err := s.calcRedeemGrossOut(amountOut)
		if err != nil {
			return nil, err
		}
		gas = redeemGas

		if !s.minTxSize.IsZero() && grossUsdc.Lt(s.minTxSize) {
			return nil, ErrBelowMinAmount
		}

		reserve := s.Info.Reserves[0] // USDC reserve = baseAssetBalance
		if grossUsdc.ToBig().Cmp(reserve) > 0 {
			return nil, ErrInsufficientBalance
		}

		fee = new(uint256.Int).Mul(grossUsdc, s.redemptionFee)
		fee.Div(fee, precision) // / 1e18
		fee.Mul(fee, s.targetLeverage)
		fee.Div(fee, precision) // / 1e18
		if fee.Gt(grossUsdc) {
			fee.Set(grossUsdc)
		}

		amountIn = big256.MulDivUp(new(uint256.Int), grossUsdc, mintScale, s.exchangeRate)
	}

	if amountIn.IsZero() {
		return nil, ErrZeroAmount
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: fee.ToBig()},
		Gas:           gas,
		SwapInfo:      SwapInfo{IsMint: isMint},
	}, nil
}

// calcMintIn inverts calcMintOut: usdcIn = ceil(ltOut * exchangeRate / 1e30)
func (s *PoolSimulator) calcMintIn(ltOut *uint256.Int) *uint256.Int {
	return big256.MulDivUp(new(uint256.Int), ltOut, s.exchangeRate, mintScale)
}

// calcRedeemGrossOut inverts the net side of calcRedeemOut: given the desired net usdcOut,
// find the gross USDC amount (pre-fee) that yields at least usdcOut after the redemption fee.
//
//	netMultiplier = 1e18 - redemptionFee*targetLeverage/1e18
//	grossUsdc = ceil(usdcOut * 1e18 / netMultiplier)
func (s *PoolSimulator) calcRedeemGrossOut(usdcOut *uint256.Int) (*uint256.Int, error) {
	feeRate := big256.MulDivDown(new(uint256.Int), s.redemptionFee, s.targetLeverage, precision)
	if feeRate.Cmp(precision) >= 0 {
		return nil, ErrFeeRateTooHigh
	}
	netMultiplier := new(uint256.Int).Sub(precision, feeRate)

	return big256.MulDivUp(new(uint256.Int), usdcOut, precision, netMultiplier), nil
}

// calcMintOut: ltOut = usdcIn.scaleFrom(6).div(exchangeRate)
//
//	= usdcIn * 1e12 * 1e18 / exchangeRate
//	= usdcIn * 1e30 / exchangeRate
func (s *PoolSimulator) calcMintOut(usdcIn *uint256.Int) (*uint256.Int, *uint256.Int) {
	scaled := new(uint256.Int).Mul(usdcIn, scaleUp) // * 1e12
	scaled.Mul(scaled, precision)                   // * 1e18  → usdcIn * 1e30
	scaled.Div(scaled, s.exchangeRate)
	return scaled, new(uint256.Int) // no explicit mint fee
}

// calcRedeemOut: grossUsdc = ltIn.mul(exchangeRate).scaleTo(6)
//
//	= ltIn * exchangeRate / 1e18 / 1e12
//	fee = grossUsdc * redemptionFee * targetLeverage / 1e36
//	usdcOut = grossUsdc - fee
func (s *PoolSimulator) calcRedeemOut(ltIn *uint256.Int) (*uint256.Int, *uint256.Int, *uint256.Int) {
	// ltToBaseAmount: ltIn * exchangeRate / 1e18 / 1e12
	grossUsdc := new(uint256.Int).Mul(ltIn, s.exchangeRate)
	grossUsdc.Div(grossUsdc, precision) // / 1e18
	grossUsdc.Div(grossUsdc, scaleUp)   // / 1e12

	// redemptionFee = grossUsdc * redemptionFee * targetLeverage / 1e36
	fee := new(uint256.Int).Mul(grossUsdc, s.redemptionFee)
	fee.Div(fee, precision) // / 1e18
	fee.Mul(fee, s.targetLeverage)
	fee.Div(fee, precision) // / 1e18

	if fee.Gt(grossUsdc) {
		fee.Set(grossUsdc)
	}
	usdcOut := new(uint256.Int).Sub(grossUsdc, fee)
	return grossUsdc, usdcOut, fee
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

	usdcReserve, overflow := uint256.FromBig(s.Info.Reserves[0])
	if overflow {
		return
	}
	ltReserve, overflow := uint256.FromBig(s.Info.Reserves[1])
	if overflow {
		return
	}

	if si.IsMint {
		// USDC in, LT out: USDC reserve increases, LT supply increases
		usdcReserve.Add(usdcReserve, amountIn)
		ltReserve.Add(ltReserve, amountOut)
	} else {
		// LT in, USDC out: base balance decreases by gross USDC (user output + redemption fee).
		grossOut := amountOut.Clone()
		if params.Fee.Amount != nil {
			fee, overflow := uint256.FromBig(params.Fee.Amount)
			if !overflow {
				grossOut.Add(grossOut, fee)
			}
		}
		if grossOut.Gt(usdcReserve) {
			usdcReserve.Clear()
		} else {
			usdcReserve.Sub(usdcReserve, grossOut)
		}
		ltReserve.Sub(ltReserve, amountIn)
	}

	s.Info.Reserves[0] = usdcReserve.ToBig()
	s.Info.Reserves[1] = ltReserve.ToBig()
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return map[string]any{
		"blockNumber": s.Info.BlockNumber,
	}
}

// ExchangeRate returns the current 1e18-scaled USDC-per-LT rate.
// Used by alt-fun (meta pool) to delegate USDC↔LT pricing without duplicating state.
func (s *PoolSimulator) ExchangeRate() *uint256.Int { return s.exchangeRate }
