package capricornpamm

import (
	"context"
	"errors"
	"math"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, client *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: cfg, ethrpcClient: client}, nil
}

const ladderPointsPerDirection = 5

func pushAgeStale(publishTime, maxPushAge uint64) (stale bool, age, safeAge uint64) {
	if publishTime == 0 || maxPushAge == 0 {
		return false, 0, 0
	}
	now := uint64(time.Now().Unix())
	if now < publishTime {
		return false, 0, maxPushAge
	}
	age = now - publishTime
	if maxPushAge > pushAgeSafetyBufferSec {
		safeAge = maxPushAge - pushAgeSafetyBufferSec
	}
	return age > safeAge, age, safeAge
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	lg := logger.WithFields(logger.Fields{"poolAddress": p.Address, "dexID": t.config.DexID})

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	token0Addr := p.Tokens[0].Address
	token1Addr := p.Tokens[1].Address

	var (
		reserves struct {
			Reserve0 *big.Int
			Reserve1 *big.Int
		}
		feeBpsRaw       *big.Int
		pricingEngineHx common.Address
		paused          bool
	)
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{ABI: pammPoolABI, Target: p.Address, Method: methodGetReserves}, []any{&reserves})
	req.AddCall(&ethrpc.Call{ABI: pammPoolABI, Target: p.Address, Method: methodFeeBps}, []any{&feeBpsRaw})
	req.AddCall(&ethrpc.Call{ABI: pammPoolABI, Target: p.Address, Method: methodPricingEngine}, []any{&pricingEngineHx})
	req.AddCall(&ethrpc.Call{ABI: pammPoolABI, Target: p.Address, Method: methodPaused}, []any{&paused})
	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}
	blockNumber := resp.BlockNumber
	pricingEngineAddr := hexutil.Encode(pricingEngineHx[:])

	var (
		maxAmountIn0     *big.Int
		maxAmountIn1     *big.Int
		oracleRegistryHx common.Address
	)
	req2 := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	req2.AddCall(&ethrpc.Call{ABI: pricingEngineABI, Target: pricingEngineAddr, Method: methodMaxInputAmount, Params: []any{common.HexToAddress(token0Addr)}}, []any{&maxAmountIn0})
	req2.AddCall(&ethrpc.Call{ABI: pricingEngineABI, Target: pricingEngineAddr, Method: methodMaxInputAmount, Params: []any{common.HexToAddress(token1Addr)}}, []any{&maxAmountIn1})
	req2.AddCall(&ethrpc.Call{ABI: pricingEngineABI, Target: pricingEngineAddr, Method: methodEngineOracleRegistry}, []any{&oracleRegistryHx})
	resp2, err := req2.TryAggregate()
	if err != nil {
		return p, err
	}

	// [0]=maxAmountIn0, [1]=maxAmountIn1, [2]=oracleRegistry.
	if len(resp2.Result) < 3 || !resp2.Result[2] {
		return p, errors.New("capricorn-pamm: oracleRegistry() reverted")
	}
	if !resp2.Result[0] {
		lg.Warnf("maxAmountIn(token0) reverted — using reserve/2")
		maxAmountIn0 = nil
	}
	if !resp2.Result[1] {
		lg.Warnf("maxAmountIn(token1) reverted — using reserve/2")
		maxAmountIn1 = nil
	}
	oracleRegistryAddr := hexutil.Encode(oracleRegistryHx[:])

	var (
		oraclePaused        bool
		maxPushPriceAge     *big.Int
		pythValidTimePeriod *big.Int
		oraclePrice         struct {
			PriceUd60x18 *big.Int
			PublishTime  uint32
		}
	)
	req3 := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	req3.AddCall(&ethrpc.Call{ABI: oracleRegistryABI, Target: oracleRegistryAddr, Method: methodOracleRegistryPaused}, []any{&oraclePaused})
	req3.AddCall(&ethrpc.Call{ABI: oracleRegistryABI, Target: oracleRegistryAddr, Method: methodMaxPushPriceAge}, []any{&maxPushPriceAge})
	req3.AddCall(&ethrpc.Call{ABI: oracleRegistryABI, Target: oracleRegistryAddr, Method: methodPythValidTimePeriod}, []any{&pythValidTimePeriod})
	req3.AddCall(&ethrpc.Call{ABI: oracleRegistryABI, Target: oracleRegistryAddr, Method: methodGetPrice, Params: []any{common.HexToHash(staticExtra.OracleId)}}, []any{&oraclePrice})
	resp3, err := req3.TryAggregate()
	if err != nil {
		return p, err
	}
	// [0]=paused, [1]=maxPushAge, [2]=pythAge, [3]=getPrice.
	getPriceOK := len(resp3.Result) >= 4 && resp3.Result[3]

	maxPushAge := uint64(0)
	if maxPushPriceAge != nil {
		maxPushAge = maxPushPriceAge.Uint64()
	}
	publishTime := uint64(0)
	if getPriceOK {
		publishTime = uint64(oraclePrice.PublishTime)
	}

	r0 := uint256.MustFromBig(reserves.Reserve0)
	r1 := uint256.MustFromBig(reserves.Reserve1)
	extra := Extra{
		FeeBps:          feeBpsRaw.Uint64(),
		Paused:          paused,
		PublishTime:     publishTime,
		MaxPushPriceAge: maxPushAge,
	}

	switch {
	case oraclePaused:
		extra.Unquoteable = true
		return t.persist(p, extra, r0, r1, blockNumber), nil
	case !getPriceOK:
		lg.Warnf("getPrice reverted at block %s, unquoteable", blockNumber)
		extra.Unquoteable = true
		return t.persist(p, extra, r0, r1, blockNumber), nil
	}
	if stale, age, safeAge := pushAgeStale(publishTime, maxPushAge); stale {
		lg.Warnf("push-age %ds > safe budget %ds (gate %ds, buffer %ds) at block %s, unquoteable",
			age, safeAge, maxPushAge, pushAgeSafetyBufferSec, blockNumber)
		extra.Unquoteable = true
		return t.persist(p, extra, r0, r1, blockNumber), nil
	}

	grid0 := buildGrid(p.Tokens[0].Decimals, reserves.Reserve0, maxAmountIn0)
	grid1 := buildGrid(p.Tokens[1].Decimals, reserves.Reserve1, maxAmountIn1)
	ladder0, ladder1, err := t.probeLadders(ctx, p.Address, token0Addr, token1Addr, grid0, grid1, blockNumber)
	if err != nil {
		return p, err
	}
	extra.Ladder0 = ladder0
	extra.Ladder1 = ladder1

	if len(ladder0) == 0 && len(ladder1) == 0 {
		extra.Unquoteable = true
	}
	return t.persist(p, extra, r0, r1, blockNumber), nil
}

func buildGrid(decimals uint8, reserveIn, maxAmountIn *big.Int) []*big.Int {
	smallest := bignumber.TenPowInt(decimals)
	if smallest.Sign() == 0 || reserveIn == nil || reserveIn.Sign() == 0 {
		return nil
	}

	half := new(big.Int).Rsh(reserveIn, 1)
	largest := half
	if maxAmountIn != nil && maxAmountIn.Sign() > 0 && maxAmountIn.Cmp(half) < 0 {
		largest = new(big.Int).Set(maxAmountIn)
	}
	if largest.Cmp(smallest) <= 0 {
		return []*big.Int{new(big.Int).Set(smallest)}
	}

	n := ladderPointsPerDirection
	pts := []*big.Int{new(big.Int).Set(smallest)}
	if n == 1 {
		return pts
	}

	smallestF := new(big.Float).SetInt(smallest)
	largestF := new(big.Float).SetInt(largest)
	rFloat, _ := new(big.Float).Quo(largestF, smallestF).Float64()
	if rFloat <= 1 {
		return pts
	}
	step := math.Pow(rFloat, 1.0/float64(n-1))

	cur := new(big.Float).SetInt(smallest)
	for i := 1; i < n-1; i++ {
		cur.Mul(cur, big.NewFloat(step))
		pt, _ := new(big.Float).Set(cur).Int(nil)
		if pt == nil || pt.Sign() == 0 {
			continue
		}
		if pt.Cmp(pts[len(pts)-1]) <= 0 || pt.Cmp(largest) >= 0 {
			continue
		}
		pts = append(pts, pt)
	}
	return append(pts, new(big.Int).Set(largest))
}

func (t *PoolTracker) probeLadders(
	ctx context.Context,
	poolAddress, token0Addr, token1Addr string,
	grid0, grid1 []*big.Int,
	blockNumber *big.Int,
) ([]LadderPoint, []LadderPoint, error) {
	if len(grid0) == 0 && len(grid1) == 0 {
		return nil, nil, nil
	}

	out0 := make([]*big.Int, len(grid0))
	out1 := make([]*big.Int, len(grid1))
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	addProbe := func(tokenIn string, amounts []*big.Int, outs []*big.Int) {
		for i, amt := range amounts {
			req.AddCall(&ethrpc.Call{
				ABI:    pammPoolABI,
				Target: poolAddress,
				Method: methodQuoteExactIn,
				Params: []any{common.HexToAddress(tokenIn), amt},
			}, []any{&outs[i]})
		}
	}
	addProbe(token0Addr, grid0, out0)
	addProbe(token1Addr, grid1, out1)

	resp, err := req.TryAggregate()
	if err != nil {
		return nil, nil, err
	}
	return collectLadder(grid0, out0, resp.Result, 0),
		collectLadder(grid1, out1, resp.Result, len(grid0)),
		nil
}

func collectLadder(grid, out []*big.Int, results []bool, offset int) []LadderPoint {
	ladder := make([]LadderPoint, 0, len(grid))
	for i, amt := range grid {
		idx := offset + i
		if idx >= len(results) || !results[idx] {
			continue
		}
		if out[i] == nil || out[i].Sign() <= 0 {
			continue
		}
		amtU, _ := uint256.FromBig(amt)
		outU, _ := uint256.FromBig(out[i])
		if amtU == nil || outU == nil {
			continue
		}
		ladder = append(ladder, LadderPoint{AmountIn: amtU, AmountOut: outU})
	}
	return ladder
}

func (t *PoolTracker) persist(p entity.Pool, extra Extra, r0, r1 *uint256.Int, blockNumber *big.Int) entity.Pool {
	extraBytes, _ := json.Marshal(extra)
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{r0.Dec(), r1.Dec()}
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()
	return p
}
