package cloberob

import (
	"cmp"
	"context"
	"math/big"
	"slices"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	cloberlib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clober-ob/libraries"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/metrics"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphql.Client
}

var _ = pooltrack.RegisterTicksBasedFactoryCEG(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client, graphqlClient *graphql.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"pool":  p.Address,
		"dexId": t.config.DexId,
	})
	l.Info("start getting new pool state")

	bookId, _ := new(big.Int).SetString(p.Address, 10)

	liquidity, blockNumber, err := t.getAllLiquidity(ctx, bookId, cloberlib.MaxTick, new(big.Int))
	if err != nil {
		return p, err
	}

	var (
		highest        cloberlib.Tick
		maxQuoteAmount uint256.Int
	)
	if len(liquidity) > 0 {
		var staticExtra StaticExtra
		if err = json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
			return p, err
		}

		highest = liquidity[0].Tick
		maxQuoteAmount.Set(calculateMaxQuoteAmount(liquidity, uint256.NewInt(staticExtra.UnitSize)))
	}

	extraBytes, err := json.Marshal(Extra{
		Depths: liquidity,
	})
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{"0", maxQuoteAmount.String()}
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber

	l.WithFields(logger.Fields{
		"highest":     highest,
		"blockNumber": p.BlockNumber,
		"nDepths":     len(liquidity),
		"maxQuote":    maxQuoteAmount,
	}).Info("finish updating state of pool")

	return p, nil
}

func (t *PoolTracker) GetNewState(ctx context.Context, p entity.Pool, logs []types.Log,
	_ map[uint64]entity.BlockHeader) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"pool":  p.Address,
		"dexId": t.config.DexId,
	})
	l.Info("start getting new state")

	if len(logs) == 0 {
		return p, nil
	}

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}

	ticks, err := t.getTicksFromLogs(logs)
	if err != nil {
		return p, err
	}

	bookId, _ := new(big.Int).SetString(p.Address, 10)
	blockNumber := eth.GetLatestBlockNumberFromLogs(logs)

	refetchedTicks, err := t.getTicksFromRPC(ctx, bookId, ticks, big.NewInt(int64(blockNumber)))
	if err != nil {
		return p, err
	}

	refetchedTicksMap := lo.SliceToMap(refetchedTicks, func(liq Liquidity) (cloberlib.Tick, Liquidity) {
		return liq.Tick, liq
	})

	for i, depth := range extra.Depths {
		if tick, ok := refetchedTicksMap[depth.Tick]; ok {
			extra.Depths[i] = tick
			delete(refetchedTicksMap, depth.Tick)
		}
	}

	// Append new ticks
	for _, liq := range refetchedTicksMap {
		extra.Depths = append(extra.Depths, liq)
	}

	// Filter empty ticks
	extra.Depths = lo.Filter(extra.Depths, func(liq Liquidity, _ int) bool {
		return liq.Depth > 0
	})

	// Sort by tick desc
	slices.SortFunc(extra.Depths, func(a, b Liquidity) int {
		return cmp.Compare(b.Tick, a.Tick)
	})

	var (
		highest        cloberlib.Tick
		maxQuoteAmount uint256.Int
	)
	if len(extra.Depths) > 0 {
		highest = extra.Depths[0].Tick

		var staticExtra StaticExtra
		if err = json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
			return p, err
		}

		maxQuoteAmount.Set(calculateMaxQuoteAmount(extra.Depths, uint256.NewInt(staticExtra.UnitSize)))
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{"0", maxQuoteAmount.String()}
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber

	l.WithFields(logger.Fields{
		"highest":     highest,
		"blockNumber": p.BlockNumber,
		"nDepths":     len(extra.Depths),
		"maxQuote":    maxQuoteAmount.String(),
	}).Info("finish updating state of pool")

	return p, nil
}

func (t *PoolTracker) FetchPoolTicks(_ context.Context, p entity.Pool) (entity.Pool, error) {
	return p, nil
}

func (t *PoolTracker) getAllLiquidity(ctx context.Context, bookId *big.Int, highest cloberlib.Tick, blockNumber *big.Int) ([]Liquidity, uint64, error) {
	var depths []Liquidity
	tick := highest

	for tick >= cloberlib.MinTick {
		var liquidity []Liquidity
		resp, err := t.ethrpcClient.NewRequest().SetBlockNumber(blockNumber).
			SetContext(ctx).
			AddCall(&ethrpc.Call{
				ABI:    bookViewerABI,
				Target: t.config.BookViewer.String(),
				Method: bookViewerMethodGetLiquidity,
				Params: []any{bookId, new(big.Int).SetInt64(int64(tick)), new(big.Int).SetUint64(uint64(maxTickLimit))},
			}, []any{&liquidity}).
			Aggregate()
		if err != nil {
			logger.Errorf("failed to get all depths %v", err)
			return nil, 0, err
		}

		blockNumber = resp.BlockNumber

		depths = append(depths, liquidity...)
		if len(liquidity) < maxTickLimit {
			break
		}

		newTick := liquidity[len(liquidity)-1].Tick - 1
		if tick <= newTick {
			break
		}

		tick = newTick
	}

	return depths, blockNumber.Uint64(), nil
}

func (t *PoolTracker) getTicksFromLogs(logs []types.Log) ([]cloberlib.Tick, error) {
	ticks := make(map[cloberlib.Tick]struct{})
	for _, event := range logs {
		if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) {
			continue
		}

		switch event.Topics[0] {
		case bookManagerABI.Events["Make"].ID:
			makeEvent, err := bookManagerFilterer.ParseMake(event)
			if err != nil {
				return nil, err
			}

			ticks[cloberlib.Tick(makeEvent.Tick.Int64())] = struct{}{}

		case bookManagerABI.Events["Take"].ID:
			takeEvent, err := bookManagerFilterer.ParseTake(event)
			if err != nil {
				return nil, err
			}

			ticks[cloberlib.Tick(takeEvent.Tick.Int64())] = struct{}{}

		case bookManagerABI.Events["Cancel"].ID:
			cancelEvent, err := bookManagerFilterer.ParseCancel(event)
			if err != nil {
				return nil, err
			}

			_, tick := cloberlib.DecodeOrderId(cancelEvent.OrderId)
			ticks[tick] = struct{}{}

		case bookManagerABI.Events["Claim"].ID:
			claimEvent, err := bookManagerFilterer.ParseClaim(event)
			if err != nil {
				return nil, err
			}

			_, tick := cloberlib.DecodeOrderId(claimEvent.OrderId)
			ticks[tick] = struct{}{}

		default:
			metrics.IncrUnprocessedEventTopic(DexType, event.Topics[0].Hex())
		}
	}

	return lo.Keys(ticks), nil
}

func (t *PoolTracker) getTicksFromRPC(ctx context.Context, bookId *big.Int, ticks []cloberlib.Tick, blockNumber *big.Int) ([]Liquidity, error) {
	fetchedTicks := make([]uint64, len(ticks))
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	for i, tick := range ticks {
		req.AddCall(&ethrpc.Call{
			ABI:    bookManagerABI,
			Target: t.config.BookManager.String(),
			Method: bookManagerMethodGetDepth,
			Params: []any{bookId, new(big.Int).SetInt64(int64(tick))},
		}, []any{&fetchedTicks[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		logger.Errorf("failed to get depths from RPC %v", err)
		return nil, err
	}

	return lo.Map(fetchedTicks, func(t uint64, idx int) Liquidity {
		return Liquidity{
			Tick:  ticks[idx],
			Depth: fetchedTicks[idx],
		}
	}), nil
}

func calculateMaxQuoteAmount(depths []Liquidity, unitSize *uint256.Int) *uint256.Int {
	var maxQuoteAmount, temp uint256.Int
	for _, d := range depths {
		temp.SetUint64(d.Depth).Mul(&temp, unitSize)
		maxQuoteAmount.Add(&maxQuoteAmount, &temp)
	}

	return &maxQuoteAmount
}
