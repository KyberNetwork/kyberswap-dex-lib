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

	decimals0, decimals1           *uint256.Int
	lpFee, protocolFee             uint64
	rateNumerator, rateDenominator *uint256.Int
	rateTimestamp, rateExpiry      uint64
	tradingMode                    uint8
	enforceMarketHours             bool
	unquoteable                    bool
	limits                         [2]TradingLimit
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

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: ep.BlockNumber,
		}},
		decimals0:          decimals0,
		decimals1:          decimals1,
		lpFee:              extra.LpFee,
		protocolFee:        extra.ProtocolFee,
		rateNumerator:      extra.RateNumerator,
		rateDenominator:    extra.RateDenominator,
		rateTimestamp:      extra.RateTimestamp,
		rateExpiry:         extra.RateExpiry,
		tradingMode:        extra.TradingMode,
		enforceMarketHours: extra.EnforceMarketHours,
		unquoteable:        extra.Unquoteable,
		limits:             extra.Limits,
	}, nil
}

func fillNilLimits(tl *TradingLimit) {
	if tl.Limit0 == nil {
		tl.Limit0 = new(int256.Int)
	}
	if tl.Limit1 == nil {
		tl.Limit1 = new(int256.Int)
	}
	if tl.Netflow0 == nil {
		tl.Netflow0 = new(int256.Int)
	}
	if tl.Netflow1 == nil {
		tl.Netflow1 = new(int256.Int)
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

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}
	if amountIn == nil || amountIn.IsZero() {
		return nil, ErrZeroAmountIn
	}

	now := nowFunc()
	if err := s.validate(now); err != nil {
		return nil, err
	}

	totalFee := s.lpFee + s.protocolFee
	if totalFee > bps {
		return nil, ErrInvalidFee
	}
	feeNum := uint256.NewInt(bps - totalFee)

	// token0 -> token1 uses (num, den); token1 -> token0 the reciprocal.
	fromDec, toDec, num, den := s.decimals0, s.decimals1, s.rateNumerator, s.rateDenominator
	if indexIn == 1 {
		fromDec, toDec, num, den = s.decimals1, s.decimals0, s.rateDenominator, s.rateNumerator
	}

	amountOut, err := convertWithRateAndFee(amountIn, fromDec, toDec, num, den, feeNum, uBps)
	if err != nil {
		return nil, err
	}
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	reserveOut, overflow := uint256.FromBig(s.Info.Reserves[indexOut])
	if overflow || reserveOut == nil || amountOut.Cmp(reserveOut) >= 0 {
		return nil, ErrInsufficientLiquidity
	}

	gross, err := convertWithRate(amountIn, fromDec, toDec, num, den)
	if err != nil {
		return nil, err
	}
	fee := new(uint256.Int)
	if gross.Cmp(amountOut) > 0 {
		fee.Sub(gross, amountOut)
	}

	var swapInfo SwapInfo
	swapInfo.Limits[indexIn], err = applyTradingLimits(s.limits[indexIn], amountIn, new(uint256.Int), totalFee, now)
	if err != nil {
		return nil, err
	}
	swapInfo.Limits[indexOut], err = applyTradingLimits(s.limits[indexOut], new(uint256.Int), amountOut, totalFee, now)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: fee.ToBig()},
		Gas:            defaultGas,
		SwapInfo:       swapInfo,
	}, nil
}

// validate ports OracleAdapter.getFXRateIfValid's guards, in order.
func (s *PoolSimulator) validate(now uint64) error {
	if s.unquoteable {
		return ErrUnquoteable
	}
	if s.enforceMarketHours && !isFXMarketOpen(now) {
		return ErrFXMarketClosed
	}
	if s.tradingMode != tradingModeBidirectional {
		return ErrTradingSuspended
	}
	// on-chain: medianTimestamp >= block.timestamp - reportExpiry
	if now+rateStalenessBufferSeconds > s.rateTimestamp+s.rateExpiry {
		return ErrNoRecentRate
	}
	if s.rateNumerator.IsZero() || s.rateDenominator.IsZero() {
		return ErrInvalidRate
	}
	return nil
}

// UpdateBalance mirrors FPMM.swap's state transition: the input reserve grows
// by amountIn minus the protocol fee that leaves the pool, the output reserve
// shrinks by amountOut, and both tokens' trading-limit states are replaced
// with the ones CalcAmountOut already computed. Every write reassigns a
// slice element or struct field wholesale, so CloneState only needs to copy
// the reserve slice.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return
	}

	protocolFee := new(big.Int).Mul(params.TokenAmountIn.Amount, big.NewInt(int64(s.protocolFee)))
	protocolFee.Div(protocolFee, big.NewInt(bps))
	retained := new(big.Int).Sub(params.TokenAmountIn.Amount, protocolFee)

	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], retained)
	if s.Info.Reserves[indexOut].Cmp(params.TokenAmountOut.Amount) >= 0 {
		s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
	} else {
		s.Info.Reserves[indexOut] = new(big.Int)
	}

	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok {
		s.limits = swapInfo.Limits
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = lo.Map(s.Info.Reserves, func(r *big.Int, _ int) *big.Int { return new(big.Int).Set(r) })
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{BlockNumber: s.Info.BlockNumber}
}
