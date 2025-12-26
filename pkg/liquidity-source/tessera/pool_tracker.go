package tessera

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
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

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	resp, err := req.AddCall(&ethrpc.Call{
		ABI:    TesseraPoolABI,
		Target: p.Address,
		Method: "poolState",
		Params: nil,
	}, []any{&rpcResult}).AddCall(&ethrpc.Call{
		ABI:    TesseraPoolABI,
		Target: p.Address,
		Method: "isInitialised",
		Params: nil,
	}, []any{&isInitialised}).TryBlockAndAggregate()

	if err != nil {
		return p, err
	}

	poolOffset0, _ := uint256.FromBig(rpcResult.PoolOffset0)
	poolOffset1, _ := uint256.FromBig(rpcResult.PoolOffset1)
	tradingEnabled := rpcResult.TradingEnabled

	orderBook0 := make([]LiquidityLevel, 0, 20)
	orderBook1 := make([]LiquidityLevel, 0, 20)

	for _, level := range rpcResult.OrderBook0 {
		if level.Active.Uint64() != 0 {
			break
		}
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
		if level.Active.Uint64() != 0 {
			break
		}
		amtU, _ := uint256.FromBig(level.Amount)
		if amtU.IsZero() {
			continue
		}
		orderBook1 = append(orderBook1, LiquidityLevel{
			Amount: amtU,
			Price:  level.Price.Uint64(),
		})
	}

	calculateCumulative := func(levels []LiquidityLevel, offsetU *uint256.Int) []*uint256.Int {
		cumulative := make([]*uint256.Int, 0, len(levels))
		sum := uint256.NewInt(0)

		// Skip filled levels
		activeIdx := -1
		skippedSum := uint256.NewInt(0)
		for i, level := range levels {
			if offsetU.Cmp(new(uint256.Int).Add(skippedSum, level.Amount)) >= 0 {
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
		currentOffset := new(uint256.Int).Sub(offsetU, skippedSum)
		for i := activeIdx; i < len(levels); i++ {
			levelRemaining := new(uint256.Int)
			if i == activeIdx {
				if levels[i].Amount.Cmp(currentOffset) > 0 {
					levelRemaining.Sub(levels[i].Amount, currentOffset)
				} else {
					continue
				}
			} else {
				levelRemaining.Set(levels[i].Amount)
			}
			sum.Add(sum, levelRemaining)
			cumulative = append(cumulative, new(uint256.Int).Set(sum))
		}
		return cumulative
	}

	baseToQuoteAmounts := calculateCumulative(orderBook0, poolOffset0)
	quoteToBaseAmounts := calculateCumulative(orderBook1, poolOffset1)

	// Subtract a small percentage (0.1%) from prefetch points to avoid T36 reverts at exact boundaries
	applyShift := func(points []*uint256.Int) []*uint256.Int {
		for _, p := range points {
			shift := new(uint256.Int).Div(p, uint256.NewInt(1000))
			if shift.IsZero() && p.Cmp(uint256.NewInt(100)) > 0 {
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

	var b2qMax, q2bMax *uint256.Int
	if len(baseToQuoteAmounts) > 0 {
		b2qMax = baseToQuoteAmounts[len(baseToQuoteAmounts)-1]
	}
	if len(quoteToBaseAmounts) > 0 {
		q2bMax = quoteToBaseAmounts[len(quoteToBaseAmounts)-1]
	}

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
			ABI:    TesseraRouterABI,
			Target: d.config.TesseraSwap,
			Method: "tesseraSwapViewAmounts",
			Params: []any{token0, token1, amt.ToBig()},
		}, []any{&baseToQuoteResults[i]})
	}
	for i, amt := range quoteToBaseAmounts {
		reqPrefetch.AddCall(&ethrpc.Call{
			ABI:    TesseraRouterABI,
			Target: d.config.TesseraSwap,
			Method: "tesseraSwapViewAmounts",
			Params: []any{token1, token0, amt.ToBig()},
		}, []any{&quoteToBaseResults[i]})
	}

	_, err = reqPrefetch.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	baseToQuotePrefetches := make([]PrefetchRate, len(baseToQuoteResults))
	for i, res := range baseToQuoteResults {
		var rate *uint256.Int
		if res.AmountIn != nil && res.AmountIn.Sign() != 0 {
			rate = uint256.MustFromBig(res.AmountOut)
		}
		baseToQuotePrefetches[i] = PrefetchRate{
			AmountIn: baseToQuoteAmounts[i],
			Rate:     rate,
		}
	}

	quoteToBasePrefetches := make([]PrefetchRate, len(quoteToBaseResults))
	for i, res := range quoteToBaseResults {
		var rate *uint256.Int
		if res.AmountIn != nil && res.AmountIn.Sign() != 0 {
			rate = uint256.MustFromBig(res.AmountOut)
		}
		quoteToBasePrefetches[i] = PrefetchRate{
			AmountIn: quoteToBaseAmounts[i],
			Rate:     rate,
		}
	}

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
	if b2qMax != nil {
		res0 = b2qMax.String()
	}
	res1 := "0"
	if q2bMax != nil {
		res1 = q2bMax.String()
	}
	p.Reserves = []string{res0, res1}

	return p, nil
}
