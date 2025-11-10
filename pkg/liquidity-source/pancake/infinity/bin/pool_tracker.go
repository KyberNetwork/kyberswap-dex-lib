package bin

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/bin/abi"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

var _ = pooltrack.RegisterFactoryCEG0(DexType, NewPoolTracker)
var _ = pooltrack.RegisterTicksBasedFactoryCEG0(DexType, NewPoolTracker)

var poolFilterer = lo.Must(abi.NewPancakeInfinityPoolManagerFilterer(common.Address{}, nil))

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolTracker {
	return &PoolTracker{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (t *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	rpcRequests := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		rpcRequests.SetBlockNumber(big.NewInt(int64(blockNumber)))
	}

	var result FetchRPCResult

	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    shared.BinPoolManagerABI,
		Target: t.config.BinPoolManagerAddress,
		Method: shared.BinPoolManagerMethodGetSlot0,
		Params: []any{common.HexToHash(p.Address)},
	}, []any{&result.Slot0})

	_, err := rpcRequests.Aggregate()
	if err != nil {
		return nil, err
	}

	lpFee := staticExtra.Fee
	if shared.IsDynamicFee(staticExtra.Fee) {
		lpFee = t.GetDynamicFee(ctx, staticExtra.HooksAddress, lpFee)
	}

	// swap fee includes protocolFee (charged first) and lpFee
	protocolFee := result.Slot0.ProtocolFee
	result.SwapFee = lo.Ternary(protocolFee == 0, uint64(lpFee), uint64(calculateSwapFee(protocolFee, lpFee)))

	return &result, err
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	param poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	l.Info("Start getting new state of pancake-infinity-bin pool")

	blockNumber, err := t.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get block number")
		return entity.Pool{}, err
	}

	var (
		rpcData         *FetchRPCResult
		newPoolReserves entity.PoolReserves
		bins            []Bin
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = t.FetchRPCData(ctx, &p, 0)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to fetch data from RPC")

		}

		return err
	})

	g.Go(func(context.Context) error {
		var err error

		bins, newPoolReserves, err = t.getBinsFromSubgraph(ctx, p.Address)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to query subgraph for bins")
		}

		return err
	})

	if err := g.Wait(); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool state")
		return entity.Pool{}, err
	}

	extra := Extra{
		ProtocolFee: rpcData.Slot0.ProtocolFee,
		ActiveBinID: rpcData.Slot0.ActiveId,
		Bins:        bins,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.SwapFee = float64(rpcData.SwapFee)
	p.Reserves = newPoolReserves
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber

	l.Infof("Finish updating state of pool")

	return p, nil
}

func (t *PoolTracker) getBinsFromSubgraph(ctx context.Context, poolAddress string) ([]Bin, entity.PoolReserves, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       t.config.DexID,
	})

	var (
		allowSubgraphError = t.config.IsAllowSubgraphError()

		lastBinId       int32 = -1
		unitX, unitY    *big.Float
		bins            []Bin
		newPoolReserves = entity.PoolReserves{"0", "0"}
		err             error
	)

	for {
		req := graphqlpkg.NewRequest(getBinsQuery(poolAddress, lastBinId, allowSubgraphError))

		var resp struct {
			Pair *LBPair `json:"lbpair"`
		}

		if err := t.graphqlClient.Run(ctx, req, &resp); err != nil {
			if allowSubgraphError {
				if resp.Pair == nil {
					l.WithFields(logger.Fields{
						"error":              err,
						"allowSubgraphError": allowSubgraphError,
					}).Error("failed to query subgraph")

					return nil, entity.PoolReserves{}, err
				}
			} else {
				l.WithFields(logger.Fields{
					"error":              err,
					"allowSubgraphError": allowSubgraphError,
				}).Error("failed to query subgraph")

				return nil, entity.PoolReserves{}, err
			}
		}

		if resp.Pair == nil || len(resp.Pair.Bins) == 0 {
			break
		}

		if unitX == nil {
			unitX, err = parseTokenDecimal(resp.Pair.TokenX.Decimals)
			if err != nil {
				return nil, entity.PoolReserves{}, err
			}
		}

		if unitY == nil {
			unitY, err = parseTokenDecimal(resp.Pair.TokenY.Decimals)
			if err != nil {
				return nil, entity.PoolReserves{}, err
			}
		}

		if newPoolReserves[0] != resp.Pair.ReserveX {
			newPoolReserves[0], err = parsePoolReserve(resp.Pair.ReserveX, unitX)
			if err != nil {
				return nil, entity.PoolReserves{}, err
			}
		}

		if newPoolReserves[1] != resp.Pair.ReserveY {
			newPoolReserves[1], err = parsePoolReserve(resp.Pair.ReserveY, unitY)
			if err != nil {
				return nil, entity.PoolReserves{}, err
			}
		}

		subgraphBins := resp.Pair.Bins
		for _, subgraphBin := range subgraphBins {
			bin, err := transformSubgraphBin(subgraphBin, unitX, unitY)
			if err != nil {
				return nil, entity.PoolReserves{}, err
			}

			bins = append(bins, bin)
		}

		if len(subgraphBins) < graphFirstLimit {
			break
		}

		lastBinId = subgraphBins[len(subgraphBins)-1].BinID
	}

	sort.Slice(bins, func(i, j int) bool {
		return bins[i].ID < bins[j].ID
	})

	return bins, newPoolReserves, nil
}

func (t *PoolTracker) GetDynamicFee(ctx context.Context, hookAddress common.Address, lpFee uint32) uint32 {
	hook, _ := GetHook(hookAddress)
	return hook.GetDynamicFee(ctx, t.ethrpcClient, t.config.BinPoolManagerAddress, hookAddress, lpFee)
}

func parseTokenDecimal(decimals string) (*big.Float, error) {
	decimalX, err := strconv.ParseInt(decimals, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token decimals: %w", err)
	}

	return bignumber.TenPowDecimals(uint8(decimalX)), nil
}

func parsePoolReserve(reserve string, unit *big.Float) (string, error) {
	reserveF, ok := new(big.Float).SetString(reserve)
	if !ok {
		return "", errors.New("can not convert pool's reserve from string to big.Float")
	}

	reserveInt, _ := new(big.Float).Mul(reserveF, unit).Int(nil)

	return reserveInt.String(), nil
}

func transformSubgraphBin(
	bin SubgraphBin,
	unitX *big.Float,
	unitY *big.Float,
) (Bin, error) {
	reserveX, ok := new(big.Float).SetString(bin.ReserveX)
	if !ok {
		return Bin{}, fmt.Errorf("[bin: %v] can not convert bin's reserveX from string to big.Float", bin.BinID)
	}
	reserveXInt, _ := new(big.Float).Mul(reserveX, unitX).Int(nil)

	reserveY, ok := new(big.Float).SetString(bin.ReserveY)
	if !ok {
		return Bin{}, fmt.Errorf("[bin: %v] can not convert bin's reserveY from string to big.Float", bin.BinID)
	}
	reserveYInt, _ := new(big.Float).Mul(reserveY, unitY).Int(nil)

	return Bin{
		ID:       uint32(bin.BinID),
		ReserveX: uint256.MustFromBig(reserveXInt),
		ReserveY: uint256.MustFromBig(reserveYInt),
	}, nil
}

func (t *PoolTracker) GetNewState(ctx context.Context, p entity.Pool, logs []ethtypes.Log,
	_ map[uint64]entity.BlockHeader) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	var blockNumber = eth.GetLatestBlockNumberFromLogs(logs)

	rpcState, err := t.FetchRPCData(ctx, &p, blockNumber)
	if err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			rpcState, err = t.FetchRPCData(ctx, &p, 0)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to fetch latest state from RPC")
				return p, err
			}
		} else {
			l.WithFields(logger.Fields{
				"error":       err,
				"blockNumber": blockNumber,
			}).Error("failed to fetch state from RPC")
			return p, err
		}
	}

	var (
		extra              Extra
		currentActiveBinID uint32
	)

	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			l.Error(err.Error())
			return p, err
		}
		currentActiveBinID = extra.ActiveBinID
	} else {
		currentActiveBinID = rpcState.Slot0.ActiveId
	}

	if len(logs) > 0 {
		binIDsFromLogs, err := t.binIDsFromLogs(currentActiveBinID, extra.Bins, logs)
		if err != nil {
			return p, err
		}

		binsFromLogs, err := t.queryRPCBins(ctx, p.Address, binIDsFromLogs, blockNumber)
		if err != nil {
			return p, err
		}

		bins, err := t.mergePoolBins(extra.Bins, binsFromLogs)
		if err != nil {
			return p, err
		}

		extra.Bins = bins
	}

	extra.ActiveBinID = rpcState.Slot0.ActiveId
	extra.ProtocolFee = rpcState.Slot0.ProtocolFee

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.Error(err.Error())
		return p, err
	}

	p.Extra = string(extraBytes)
	p.SwapFee = float64(rpcState.SwapFee)
	p.Reserves = calculateReservesFromBins(extra.Bins)
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	var extra Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			l.Error(err.Error())
			return p, err
		}
	}

	binIDs := lo.Map(extra.Bins, func(item Bin, _ int) uint32 {
		return item.ID
	})
	bins, err := t.queryRPCBins(ctx, p.Address, binIDs, p.BlockNumber)
	if err != nil {
		l.Error(err.Error())
		return p, err
	}
	bins = filterEmptyAndSortBins(bins)

	extra.Bins = bins
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.Error(err.Error())
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) queryRPCBins(ctx context.Context, poolAddress string, binIDs []uint32, blockNumber uint64) ([]Bin, error) {
	if len(binIDs) == 0 {
		return []Bin{}, nil
	}

	bins := make([]Bin, 0, len(binIDs))
	for from := 0; from < len(binIDs); {
		to := min(from+binChunk, len(binIDs))

		b, err := t.queryRPCBinsByChunk(ctx, poolAddress, binIDs[from:to], blockNumber)
		if err != nil {
			return nil, err
		}

		bins = append(bins, b...)
		from = to
	}

	return bins, nil
}

func (t *PoolTracker) queryRPCBinsByChunk(ctx context.Context, poolAddress string, binIDs []uint32, blockNumber uint64) ([]Bin, error) {
	if len(binIDs) == 0 {
		return []Bin{}, nil
	}

	req := t.ethrpcClient.R().SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		req.SetBlockNumber(&blockNumberBI)
	}
	bins := make([]BinResp, len(binIDs))
	for idx, binID := range binIDs {
		req.AddCall(&ethrpc.Call{
			ABI:    shared.BinPoolManagerABI,
			Target: t.config.BinPoolManagerAddress,
			Method: methodGetBin,
			Params: []any{common.HexToHash(poolAddress), big.NewInt(int64(binID))},
		}, []any{&bins[idx]})
	}

	if _, err := req.Aggregate(); err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			// Re-query ticks data with latest block number
			return t.queryRPCBinsByChunk(ctx, poolAddress, binIDs, 0)
		}

		logger.WithFields(logger.Fields{
			"address": poolAddress,
		}).Error(err.Error())
		return nil, err
	}

	return lo.Map(bins, func(_ BinResp, idx int) Bin {
		return Bin{
			ID:       binIDs[idx],
			ReserveX: uint256.MustFromBig(bins[idx].BinReserveX),
			ReserveY: uint256.MustFromBig(bins[idx].BinReserveY),
		}
	}), nil
}

func (t *PoolTracker) mergePoolBins(currentBins, binsFromLogs []Bin) ([]Bin, error) {
	bins := binsFromLogs

	isBinFromLogs := lo.Associate(bins, func(bin Bin) (uint32, struct{}) {
		return bin.ID, struct{}{}
	})

	for _, b := range currentBins {
		if _, ok := isBinFromLogs[b.ID]; ok {
			continue
		}
		bins = append(bins, b)
	}

	return filterEmptyAndSortBins(bins), nil
}

func (t *PoolTracker) binIDsFromLogs(currentActiveBinID uint32, currentBins []Bin, logs []ethtypes.Log) ([]uint32, error) {
	binSet := map[uint32]struct{}{}

	for _, event := range logs {
		if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) {
			continue
		}

		l := logger.WithFields(logger.Fields{
			"blockNumber": event.BlockNumber,
			"blockHash":   event.BlockHash,
			"logIndex":    event.Index,
		})

		switch event.Topics[0] {
		case shared.BinPoolManagerABI.Events["Swap"].ID:
			swap, err := poolFilterer.ParseSwap(event)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to parse Swap event")
				return nil, err
			}

			newActiveBinId := uint32(swap.ActiveId.Uint64())
			if newActiveBinId == currentActiveBinID {
				binSet[currentActiveBinID] = struct{}{}
				continue
			}

			minUpdatedBinId := min(newActiveBinId, currentActiveBinID)
			maxUpdatedBinId := max(newActiveBinId, currentActiveBinID)

			start := sort.Search(len(currentBins), func(i int) bool {
				return currentBins[i].ID >= minUpdatedBinId
			})

			for i := start; i < len(currentBins); i++ {
				id := currentBins[i].ID
				if id > maxUpdatedBinId {
					break
				}
				binSet[id] = struct{}{}
			}

		case shared.BinPoolManagerABI.Events["Mint"].ID:
			mint, err := poolFilterer.ParseMint(event)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to parse Mint event")
				return nil, err
			}

			for _, binId := range mint.Ids {
				if binId != nil {
					binSet[uint32(binId.Int64())] = struct{}{}
				}
			}

		case shared.BinPoolManagerABI.Events["Burn"].ID:
			burn, err := poolFilterer.ParseBurn(event)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to parse Burn event")
				return nil, err
			}

			for _, binId := range burn.Ids {
				if binId != nil {
					binSet[uint32(binId.Int64())] = struct{}{}
				}
			}

		default:
			// metrics.IncrUnprocessedEventTopic(pooltypes.PoolTypes.PancakeInfinityBin, event.Topics[0].Hex())
		}
	}

	binIDs := make([]uint32, 0, len(binSet))
	for binID := range binSet {
		binIDs = append(binIDs, binID)
	}

	return binIDs, nil
}

func filterEmptyAndSortBins(bins []Bin) []Bin {
	b := lo.Filter(bins, func(item Bin, _ int) bool {
		return item.ReserveX.Sign() > 0 ||
			item.ReserveY.Sign() > 0
	})
	sort.Slice(b, func(i, j int) bool {
		return b[i].ID < b[j].ID
	})
	return b
}

func calculateReservesFromBins(bins []Bin) entity.PoolReserves {
	var (
		reserveX = uint256.NewInt(0)
		reserveY = uint256.NewInt(0)
	)

	for _, bin := range bins {
		reserveX.Add(reserveX, bin.ReserveX)
		reserveY.Add(reserveY, bin.ReserveY)
	}

	return entity.PoolReserves{reserveX.String(), reserveY.String()}
}
