package mento

import (
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// nowFunc is the clock the simulator evaluates oracle staleness, FX market
// hours and trading-limit window resets against. Tests override it.
var nowFunc = func() uint64 { return uint64(time.Now().Unix()) }

type PoolSimulator struct {
	pool.Pool

	reserves                  [2]uint256.Int
	rates                     [2]quoteRate // indexed by tokenIn
	protocolFee, totalFee     uint64
	rateTimestamp, rateExpiry uint64
	tradingMode               uint8
	enforceMarketHours        bool
	unquoteable               bool
	limits                    [2]TradingLimit
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, errors.New("mento: pool must have exactly 2 tokens and 2 reserves")
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	decimals0, err := big256.NewUint256(staticExtra.Decimals0)
	if err != nil {
		return nil, err
	}
	decimals1, err := big256.NewUint256(staticExtra.Decimals1)
	if err != nil {
		return nil, err
	}

	if len(ep.Extra) == 0 {
		return nil, errors.New("mento: pool extra is empty")
	}
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	if extra.RateNumerator == nil || extra.RateDenominator == nil {
		return nil, errors.New("mento: pool extra missing oracle rate")
	}
	for i := range extra.Limits {
		fillNilLimits(&extra.Limits[i])
	}

	// FPMM.getAmountOut: BASIS_POINTS_DENOMINATOR - (lpFee + protocolFee) reverts on underflow.
	totalFee := extra.LpFee + extra.ProtocolFee
	if totalFee > bps {
		return nil, ErrInvalidFee
	}
	s := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: ep.BlockNumber,
		}},
		protocolFee:        extra.ProtocolFee,
		totalFee:           totalFee,
		rateTimestamp:      extra.RateTimestamp,
		rateExpiry:         extra.RateExpiry,
		tradingMode:        extra.TradingMode,
		enforceMarketHours: extra.EnforceMarketHours,
		unquoteable:        extra.Unquoteable,
		limits:             extra.Limits,
	}
	for i, r := range ep.Reserves {
		if err = s.reserves[i].SetFromDecimal(r); err != nil {
			return nil, err
		}
	}
	// token0 -> token1 uses (num, den); token1 -> token0 the reciprocal.
	num, den := extra.RateNumerator, extra.RateDenominator
	if s.rates[0], err = newQuoteRate(num, den, decimals0, decimals1, bps-totalFee); err != nil {
		return nil, err
	}
	if s.rates[1], err = newQuoteRate(den, num, decimals1, decimals0, bps-totalFee); err != nil {
		return nil, err
	}
	return s, nil
}

func fillNilLimits(tl *TradingLimit) {
	for _, p := range []**int256.Int{&tl.Limit0, &tl.Limit1, &tl.Netflow0, &tl.Netflow1} {
		if *p == nil {
			*p = new(int256.Int)
		}
	}
}

// CalcAmountOut ports FPMM.getAmountOut plus every guard FPMM.swap applies
// afterwards, in on-chain order: oracle validity (OracleAdapter.
// getFXRateIfValid), the conversion, amountOut < reserveOut, and the
// TradingLimitsV2 update for both tokens.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}

	if params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmountIn
	}
	var amountIn uint256.Int
	if amountIn.SetFromBig(params.TokenAmountIn.Amount) {
		return nil, ErrOverflow
	}

	now := nowFunc()
	if err := s.validate(now); err != nil {
		return nil, err
	}

	var amountOut, fee uint256.Int
	if err := s.rates[indexIn].amountOut(&amountIn, &amountOut, &fee); err != nil {
		return nil, err
	}
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}
	if !amountOut.Lt(&s.reserves[indexOut]) {
		return nil, ErrInsufficientLiquidity
	}

	var zero uint256.Int
	var swapInfo SwapInfo
	var err error
	if swapInfo.Limits[indexIn], err = applyTradingLimits(s.limits[indexIn], &amountIn, &zero, s.totalFee,
		now); err != nil {
		return nil, err
	}
	if swapInfo.Limits[indexOut], err = applyTradingLimits(s.limits[indexOut], &zero, &amountOut, s.totalFee,
		now); err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: fee.ToBig()},
		Gas:            defaultGas,
		SwapInfo:       swapInfo,
	}, nil
}

// validate ports OracleAdapter.getFXRateIfValid's guards. Time-based guards also
// hold at now + executionDelaySeconds, so a quote is not routed into a revert.
func (s *PoolSimulator) validate(now uint64) error {
	if s.unquoteable {
		return ErrUnquoteable
	}
	if s.enforceMarketHours && (!isFXMarketOpen(now) || !isFXMarketOpen(now+executionDelaySeconds)) {
		return ErrFXMarketClosed
	}
	if s.tradingMode != tradingModeBidirectional {
		return ErrTradingSuspended
	}
	// on-chain: medianTimestamp >= block.timestamp - reportExpiry
	if now+executionDelaySeconds > s.rateTimestamp+s.rateExpiry {
		return ErrNoRecentRate
	}
	return nil
}

// UpdateBalance mirrors FPMM.swap's state transition: the input reserve grows
// by amountIn minus the protocol fee that leaves the pool, the output reserve
// shrinks by amountOut, and both tokens' trading-limit states are replaced
// with the ones CalcAmountOut already computed.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return
	}

	// FPMM._transferProtocolFee: amountIn * protocolFee / BASIS_POINTS_DENOMINATOR leaves the pool.
	var amount, fee uint256.Int
	amount.SetFromBig(params.TokenAmountIn.Amount)
	big256.MulDivDown(&fee, &amount, fee.SetUint64(s.protocolFee), uBps)
	s.reserves[indexIn].Add(&s.reserves[indexIn], amount.Sub(&amount, &fee))

	amount.SetFromBig(params.TokenAmountOut.Amount)
	if s.reserves[indexOut].Lt(&amount) {
		s.reserves[indexOut].Clear()
	} else {
		s.reserves[indexOut].Sub(&s.reserves[indexOut], &amount)
	}

	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		s.limits = swapInfo.Limits
	}
}

// CloneState is a struct copy: reserves and limits are value arrays, and UpdateBalance
// replaces limit pointers instead of writing through them.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{BlockNumber: s.Info.BlockNumber}
}
