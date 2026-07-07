package tessera

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
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

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	token0 := common.HexToAddress(p.Tokens[0].Address)
	token1 := common.HexToAddress(p.Tokens[1].Address)

	var rpcResult poolStateResult
	var isInitialised bool
	var reserves = make([]*big.Int, len(p.Tokens))

	req := d.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    tesseraPoolABI,
			Target: p.Address,
			Method: "poolState",
		}, []any{&rpcResult}).
		AddCall(&ethrpc.Call{
			ABI:    tesseraPoolABI,
			Target: p.Address,
			Method: "isInitialised",
		}, []any{&isInitialised})

	for i, token := range p.Tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: token.Address,
			Method: "balanceOf",
			Params: []any{common.HexToAddress(d.config.TesseraTreasury)},
		}, []any{&reserves[i]})
	}

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	poolOffset0, _ := uint256.FromBig(rpcResult.PoolOffset0)
	poolOffset1, _ := uint256.FromBig(rpcResult.PoolOffset1)
	tradingEnabled := rpcResult.TradingEnabled

	orderBook0 := make([]LiquidityLevel, 0, 20)
	orderBook1 := make([]LiquidityLevel, 0, 20)

	for _, level := range rpcResult.OrderBook0 {
		amtU, _ := uint256.FromBig(level.Amount)
		if amtU.IsZero() {
			continue
		}
		orderBook0 = append(orderBook0, LiquidityLevel{
			Amount: amtU,
			Price:  level.Price.Uint64(),
		})
	}

	for _, level := range rpcResult.OrderBook1 {
		amtU, _ := uint256.FromBig(level.Amount)
		if amtU.IsZero() {
			continue
		}
		orderBook1 = append(orderBook1, LiquidityLevel{
			Amount: amtU,
			Price:  level.Price.Uint64(),
		})
	}

	var tmp uint256.Int
	calculateCumulative := func(levels []LiquidityLevel, offsetU *uint256.Int) []*uint256.Int {
		cumulative := make([]*uint256.Int, 0, len(levels))
		sum := new(uint256.Int)

		// Skip filled levels
		activeIdx := -1
		skippedSum := new(uint256.Int)
		for i, level := range levels {
			if !offsetU.Lt(tmp.Add(skippedSum, level.Amount)) {
				skippedSum.Add(skippedSum, level.Amount)
				continue
			}
			activeIdx = i
			break
		}

		if activeIdx == -1 {
			return nil
		}

		// Calculate cumulative amounts starting from active level
		currentOffset := tmp.Sub(offsetU, skippedSum)
		for i := activeIdx; i < len(levels); i++ {
			levelRemaining := new(uint256.Int)
			if i == activeIdx {
				if levels[i].Amount.Gt(currentOffset) {
					levelRemaining.Sub(levels[i].Amount, currentOffset)
				} else {
					continue
				}
			} else {
				levelRemaining.Set(levels[i].Amount)
			}
			sum.Add(sum, levelRemaining)
			cumulative = append(cumulative, sum.Clone())
		}
		return cumulative
	}

	baseToQuoteAmounts := calculateCumulative(orderBook0, poolOffset1)
	quoteToBaseAmounts := calculateCumulative(orderBook1, poolOffset0)

	// Subtract a small percentage (0.1%) from prefetch points to avoid T36 reverts at exact boundaries
	applyShift := func(points []*uint256.Int) []*uint256.Int {
		for _, p := range points {
			shift := tmp.Div(p, tmp.SetUint64(1000))
			if shift.IsZero() && p.CmpUint64(100) > 0 {
				shift.SetUint64(100)
			}
			if !shift.IsZero() {
				p.Sub(p, shift)
			}
		}
		return points
	}
	baseToQuoteAmounts = applyShift(baseToQuoteAmounts)
	quoteToBaseAmounts = applyShift(quoteToBaseAmounts)

	limitPrefetchPoints := func(points []*uint256.Int) []*uint256.Int {
		maxPrefetchPoints := d.config.MaxPrefetchPoints
		if maxPrefetchPoints <= 0 {
			maxPrefetchPoints = 20
		}

		if len(points) > maxPrefetchPoints {
			return points[:maxPrefetchPoints]
		}
		return points
	}

	baseToQuoteAmounts = limitPrefetchPoints(baseToQuoteAmounts)
	quoteToBaseAmounts = limitPrefetchPoints(quoteToBaseAmounts)

	// Prefetch swap rates using tesseraSwapViewAmounts
	reqPrefetch := d.ethrpcClient.NewRequest().SetContext(ctx)
	if resp.BlockNumber != nil {
		reqPrefetch.SetBlockNumber(resp.BlockNumber)
	}

	baseToQuoteResults := make([]poolSwapViewAmounts, len(baseToQuoteAmounts))
	quoteToBaseResults := make([]poolSwapViewAmounts, len(quoteToBaseAmounts))

	for i, amt := range baseToQuoteAmounts {
		reqPrefetch.AddCall(&ethrpc.Call{
			ABI:    tesseraRouterABI,
			Target: d.config.TesseraSwap,
			Method: "tesseraSwapViewAmounts",
			Params: []any{token0, token1, amt.ToBig()},
		}, []any{&baseToQuoteResults[i]})
	}
	for i, amt := range quoteToBaseAmounts {
		reqPrefetch.AddCall(&ethrpc.Call{
			ABI:    tesseraRouterABI,
			Target: d.config.TesseraSwap,
			Method: "tesseraSwapViewAmounts",
			Params: []any{token1, token0, amt.ToBig()},
		}, []any{&quoteToBaseResults[i]})
	}

	_, err = reqPrefetch.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	buffer := new(uint256.Int).SubUint64(big256.UBasisPoint, d.config.PriceTolerance)

	buildPrefetches := func(results []poolSwapViewAmounts, amounts []*uint256.Int) []PrefetchRate {
		prefetches := make([]PrefetchRate, len(results))
		for i, res := range results {
			if res.AmountOut == nil || res.AmountOut.Sign() == 0 {
				return prefetches[:i]
			}
			var rate *uint256.Int
			if res.AmountIn != nil && res.AmountIn.Sign() != 0 {
				rate = uint256.MustFromBig(res.AmountOut)
				rate.MulDivOverflow(rate, buffer, big256.UBasisPoint)
			}
			prefetches[i] = PrefetchRate{
				AmountIn: amounts[i],
				Rate:     rate,
			}
		}
		return prefetches
	}

	baseToQuotePrefetches := buildPrefetches(baseToQuoteResults, baseToQuoteAmounts)
	quoteToBasePrefetches := buildPrefetches(quoteToBaseResults, quoteToBaseAmounts)

	extra := Extra{
		BaseToQuotePrefetches: baseToQuotePrefetches,
		QuoteToBasePrefetches: quoteToBasePrefetches,
		TradingEnabled:        tradingEnabled,
		IsInitialised:         isInitialised,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()

	res0 := "0"
	if reserves[0] != nil {
		res0 = reserves[0].String()
	}
	res1 := "0"
	if reserves[1] != nil {
		res1 = reserves[1].String()
	}
	p.Reserves = []string{res0, res1}

	return p, nil
}
