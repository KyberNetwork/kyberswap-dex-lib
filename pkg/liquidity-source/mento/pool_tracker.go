package mento

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{config: cfg, ethrpcClient: ethrpcClient}
}

// ABI decode targets. Field names must match the output/component names in
// abi/*.json after CamelCasing.
type (
	poolReserves struct {
		Reserve0           *big.Int
		Reserve1           *big.Int
		BlockTimestampLast *big.Int
	}
	tradingLimitConfig struct {
		Limit0   *big.Int
		Limit1   *big.Int
		Decimals uint8
	}
	tradingLimitState struct {
		LastUpdated0 uint32
		LastUpdated1 uint32
		Netflow0     *big.Int
		Netflow1     *big.Int
	}
	tradingLimits struct {
		Config tradingLimitConfig
		State  tradingLimitState
	}
	medianRate struct {
		Numerator   *big.Int
		Denominator *big.Int
	}
)

// GetNewPoolState refreshes everything FPMM.swap() checks, in three
// multicalls pinned to the block of the first:
//  1. pool: reserves, fees, oracle adapter, rate feed, invert flag, trading limits
//  2. oracle adapter: sortedOracles, breakerBox, marketHoursBreaker
//  3. feed: median rate + timestamp + expiry, breaker trading mode, and a
//     market-hours probe at a known-closed timestamp
//
// Round 3 is revert-tolerant: a feed that is not registered in the
// BreakerBox reverts getRateFeedTradingMode, which is exactly the state in
// which swaps revert too, so the pool is persisted as Unquoteable rather
// than dropped.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ pool.GetNewPoolStateParams) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, errors.New("mento: pool must have exactly 2 tokens")
	}
	lg := logger.WithFields(logger.Fields{"dex": t.config.DexID, "pool": p.Address})

	// round 1: pool
	var (
		reserves      poolReserves
		lpFee         *big.Int
		protocolFee   *big.Int
		oracleAdapter common.Address
		rateFeedID    common.Address
		invert        bool
		limits        [2]tradingLimits
	)
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: methodGetReserves}, []any{&reserves})
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: methodLpFee}, []any{&lpFee})
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: methodProtocolFee}, []any{&protocolFee})
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: methodOracleAdapter}, []any{&oracleAdapter})
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: methodReferenceRateFeedID}, []any{&rateFeedID})
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: methodInvertRateFeed}, []any{&invert})
	for i := range limits {
		req.AddCall(&ethrpc.Call{
			ABI: poolABI, Target: p.Address, Method: methodGetTradingLimits,
			Params: []any{common.HexToAddress(p.Tokens[i].Address)},
		}, []any{&limits[i]})
	}
	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}
	blockNumber := resp.BlockNumber
	if reserves.Reserve0 == nil || reserves.Reserve1 == nil || lpFee == nil || protocolFee == nil {
		return p, errors.New("mento: pool state read returned nil")
	}

	// round 2: oracle adapter
	var sortedOracles, breakerBox, marketHoursBreaker common.Address
	adapterAddr := hexutil.Encode(oracleAdapter[:])
	req2 := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	req2.AddCall(&ethrpc.Call{ABI: oracleAdapterABI, Target: adapterAddr, Method: methodSortedOracles}, []any{&sortedOracles})
	req2.AddCall(&ethrpc.Call{ABI: oracleAdapterABI, Target: adapterAddr, Method: methodBreakerBox}, []any{&breakerBox})
	req2.AddCall(&ethrpc.Call{ABI: oracleAdapterABI, Target: adapterAddr, Method: methodMarketHoursBreaker}, []any{&marketHoursBreaker})
	if _, err = req2.Aggregate(); err != nil {
		return p, err
	}

	// round 3: rate feed
	var (
		rate           medianRate
		rateTimestamp  *big.Int
		rateExpiry     *big.Int
		tradingMode    uint8
		openOnSaturday bool
	)
	feed := rateFeedID
	req3 := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	req3.AddCall(&ethrpc.Call{ABI: sortedOraclesABI, Target: hexutil.Encode(sortedOracles[:]), Method: methodMedianRate, Params: []any{feed}}, []any{&rate})
	req3.AddCall(&ethrpc.Call{ABI: sortedOraclesABI, Target: hexutil.Encode(sortedOracles[:]), Method: methodMedianTimestamp, Params: []any{feed}}, []any{&rateTimestamp})
	req3.AddCall(&ethrpc.Call{ABI: sortedOraclesABI, Target: hexutil.Encode(sortedOracles[:]), Method: methodGetTokenReportExpirySeconds, Params: []any{feed}}, []any{&rateExpiry})
	req3.AddCall(&ethrpc.Call{ABI: breakerBoxABI, Target: hexutil.Encode(breakerBox[:]), Method: methodGetRateFeedTradingMode, Params: []any{feed}}, []any{&tradingMode})
	req3.AddCall(&ethrpc.Call{ABI: marketHoursBreakerABI, Target: hexutil.Encode(marketHoursBreaker[:]), Method: methodIsFXMarketOpen, Params: []any{big.NewInt(closedMarketProbeTimestamp)}}, []any{&openOnSaturday})
	resp3, err := req3.TryAggregate()
	if err != nil {
		return p, err
	}
	if len(resp3.Result) < 5 {
		return p, errors.New("mento: short multicall response")
	}

	extra := Extra{
		LpFee:              lpFee.Uint64(),
		ProtocolFee:        protocolFee.Uint64(),
		RateNumerator:      new(uint256.Int),
		RateDenominator:    new(uint256.Int),
		TradingMode:        tradingMode,
		EnforceMarketHours: true,
	}

	switch {
	case !resp3.Result[0] || !resp3.Result[1] || !resp3.Result[2] || !resp3.Result[3]:
		lg.Warnf("rate feed %s reads reverted at block %s (results %v), marking unquoteable",
			feed.Hex(), blockNumber, resp3.Result)
		extra.Unquoteable = true
	case rate.Numerator == nil || rate.Denominator == nil || rateTimestamp == nil || rateExpiry == nil:
		lg.Warn("rate feed read returned nil, marking unquoteable")
		extra.Unquoteable = true
	default:
		num, den := big256.FromBig(rate.Numerator), big256.FromBig(rate.Denominator)
		// OracleAdapter._getOracleRate asserts the fixidity denominator and
		// scales both terms down by 1e6.
		if !den.Eq(sortedOraclesDenom) {
			lg.Warnf("medianRate denominator %s != 1e24, marking unquoteable", den)
			extra.Unquoteable = true
		}
		num.Div(num, oracleAdapterScaleDiv)
		den.Div(den, oracleAdapterScaleDiv)
		if invert {
			num, den = den, num
		}
		extra.RateNumerator, extra.RateDenominator = num, den
		extra.RateTimestamp = rateTimestamp.Uint64()
		extra.RateExpiry = rateExpiry.Uint64()
	}
	// A breaker that reports the FX market open on a Saturday does not
	// enforce hours. If the probe itself reverted, keep the conservative
	// default and enforce the calendar.
	if resp3.Result[4] {
		extra.EnforceMarketHours = !openOnSaturday
	}

	for i := range limits {
		extra.Limits[i] = TradingLimit{
			Limit0:       int256.MustFromBig(nilToZero(limits[i].Config.Limit0)),
			Limit1:       int256.MustFromBig(nilToZero(limits[i].Config.Limit1)),
			Decimals:     limits[i].Config.Decimals,
			LastUpdated0: limits[i].State.LastUpdated0,
			LastUpdated1: limits[i].State.LastUpdated1,
			Netflow0:     int256.MustFromBig(nilToZero(limits[i].State.Netflow0)),
			Netflow1:     int256.MustFromBig(nilToZero(limits[i].State.Netflow1)),
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{reserves.Reserve0.String(), reserves.Reserve1.String()}
	p.BlockNumber = blockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
}

func nilToZero(b *big.Int) *big.Int {
	if b == nil {
		return new(big.Int)
	}
	return b
}
