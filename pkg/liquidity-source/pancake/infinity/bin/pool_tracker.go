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
	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/bin/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/metrics"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = pooltrack.RegisterFactoryCEG0(DexType, NewPoolTracker)

var poolFilterer = lo.Must(abi.NewPancakeInfinityPoolManagerFilterer(common.Address{}, nil))

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
	lensGuard     *ephemeral.Guard
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
		lensGuard:     ephemeral.NewGuard(config.LensGuard),
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

	hook, _ := GetHook(staticExtra.HooksAddress)
	if h, ok := hook.(SwapFeeHook); ok {
		rpcRequests.AddCall(h.SwapFeeCall(staticExtra.HooksAddress, p.Address), []any{&result.HookSwapFee})
	}

	resp, err := rpcRequests.Aggregate()
	if err != nil {
		return nil, err
	}
	result.BlockNumber = resp.BlockNumber.Uint64()

	lpFee := staticExtra.Fee
	if shared.IsDynamicFee(staticExtra.Fee) {
		if staticExtra.HasSwapPermissions {
			lpFee = t.GetDynamicFee(ctx, staticExtra.HooksAddress, result.Slot0.LpFee)
		} else {
			// no swap callbacks: the manager charges the dynamic fee stored in slot0
			lpFee = result.Slot0.LpFee
		}
	}

	// swap fee includes protocolFee (charged first) and lpFee
	protocolFee := result.Slot0.ProtocolFee
	result.SwapFee = lo.Ternary(protocolFee == 0, uint64(lpFee), uint64(calculateSwapFee(protocolFee, lpFee)))

	return &result, err
}

func (t *PoolTracker) BootstrapPoolState(
	ctx context.Context,
	p entity.Pool,
	param poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	l.Info("Start getting new state of pancake-infinity-bin pool")

	if t.config.UseLens {
		p, _, err := t.bootstrapFromLens(ctx, p, nil)
		return p, err
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
		HookSwapFee: rpcData.HookSwapFee,
		Bins:        withActiveBin(bins, rpcData.Slot0.ActiveId),
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
	p.BlockNumber = rpcData.BlockNumber
	p.Reserves = rescaleNativeReserves(valueobject.ChainID(t.config.ChainID), &p)

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
	decimalX, err := kutils.Atou[uint8](decimals)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token decimals: %w", err)
	}

	return bignumber.TenPowDecimals(decimalX), nil
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

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, param poolpkg.GetNewPoolStateParams) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	if t.config.UseLens && !hasState(p) {
		return tickspkg.StateFromLens(ctx, t.lensGuard, p, param,
			func(ctx context.Context, block *big.Int) (entity.Pool, uint64, error) {
				return t.bootstrapFromLens(ctx, p, block)
			},
			l), nil
	}

	blockNumber := tickspkg.ReadBlock(param.Logs, p.BlockNumber)
	rpcState, err := t.FetchRPCData(ctx, &p, blockNumber)
	if err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			blockNumber = 0
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

	// The block asked for when there was one: a multicall's own block is not the chain's number on
	// every chain (Etherlink reports the next one). The pool's block never goes back.
	if blockNumber == 0 {
		blockNumber = rpcState.BlockNumber
	}
	activeID := rpcState.Slot0.ActiveId

	if len(param.Logs) > 0 {
		binIDsFromLogs, minted, err := t.binIDsFromLogs(currentActiveBinID, extra.Bins, param.Logs)
		if err != nil {
			return p, err
		}
		// The active bin too, whatever the logs name: a swap ends in it, and it tells whether the
		// stored set was missing a bin.
		binIDsFromLogs = append(binIDsFromLogs, activeID)

		binsFromLogs, err := t.queryRPCBins(ctx, p.Address, lo.Uniq(binIDsFromLogs), blockNumber)
		if err != nil {
			return p, err
		}

		bins, err := t.mergePoolBins(extra.Bins, binsFromLogs)
		if err != nil {
			return p, err
		}

		if lacksBin(extra.Bins, binsFromLogs, minted, activeID) {
			bins = t.readAllBins(ctx, p.Address, blockNumber, activeID, bins)
		}

		extra.Bins = bins
	}

	extra.Bins = withActiveBin(filterEmptyAndSortBins(extra.Bins), activeID)
	extra.ActiveBinID = activeID
	extra.ProtocolFee = rpcState.Slot0.ProtocolFee
	extra.HookSwapFee = rpcState.HookSwapFee
	p.BlockNumber = max(p.BlockNumber, blockNumber)

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.Error(err.Error())
		return p, err
	}

	p.Extra = string(extraBytes)
	p.SwapFee = float64(rpcState.SwapFee)
	p.Reserves = calculateReservesFromBins(extra.Bins)
	p.Reserves = rescaleNativeReserves(valueobject.ChainID(t.config.ChainID), &p)
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	// The lens read the bins at the block it read the state, which is what reading them again is for.
	if t.config.UseLens {
		return p, nil
	}

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
	extra.Bins = withActiveBin(filterEmptyAndSortBins(bins), extra.ActiveBinID)
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

// binIDsFromLogs returns the bins logs change, and among them those a Mint names, which may be new
// to the pool's set.
func (t *PoolTracker) binIDsFromLogs(currentActiveBinID uint32, currentBins []Bin,
	logs []ethtypes.Log) ([]uint32, map[uint32]struct{}, error) {
	binSet := map[uint32]struct{}{}
	minted := map[uint32]struct{}{}

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
				return nil, nil, err
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
				return nil, nil, err
			}

			for _, binId := range mint.Ids {
				if binId != nil {
					binSet[uint32(binId.Int64())] = struct{}{}
					minted[uint32(binId.Int64())] = struct{}{}
				}
			}

		case shared.BinPoolManagerABI.Events["Burn"].ID:
			burn, err := poolFilterer.ParseBurn(event)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to parse Burn event")
				return nil, nil, err
			}

			for _, binId := range burn.Ids {
				if binId != nil {
					binSet[uint32(binId.Int64())] = struct{}{}
				}
			}

		default:
			metrics.IncrUnprocessedEventTopic(DexType, event.Topics[0].Hex())
		}
	}

	binIDs := make([]uint32, 0, len(binSet))
	for binID := range binSet {
		binIDs = append(binIDs, binID)
	}

	return binIDs, minted, nil
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

// rescaleNativeReserves converts a native-flagged reserve from its real on-chain decimals to
// the wrapped-native address's decimals used elsewhere for that token; a no-op except on Arc.
func rescaleNativeReserves(chainID valueobject.ChainID, p *entity.Pool) entity.PoolReserves {
	reserves := p.Reserves
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return reserves
	}
	for i, isNative := range staticExtra.IsNative {
		if isNative && i < len(reserves) {
			amount, ok := new(big.Int).SetString(reserves[i], 10)
			if ok {
				reserves[i] = valueobject.WrapNativeAmount(chainID, amount).String()
			}
		}
	}
	return reserves
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

// bootstrapFromLens reads the pool's bins through the lens at block (nil for the latest), then its
// slot0 and fees, which the lens does not return, at the same block. It returns the pool with the
// block the lens read it at.
func (t *PoolTracker) bootstrapFromLens(ctx context.Context, p entity.Pool, block *big.Int) (entity.Pool, uint64,
	error) {
	snap, err := t.snapshot(ctx, p.Address, block)
	if err != nil {
		return entity.Pool{}, 0, err
	}

	rpcData, err := t.FetchRPCData(ctx, &p, snap.block)
	if err != nil {
		return entity.Pool{}, 0, err
	}
	if rpcData.Slot0.ActiveId != snap.activeID {
		return entity.Pool{}, 0, fmt.Errorf("pool %s at block %d: RPC and lens disagree on the active bin",
			p.Address, snap.block)
	}

	extraBytes, err := json.Marshal(Extra{
		ProtocolFee: rpcData.Slot0.ProtocolFee,
		ActiveBinID: snap.activeID,
		HookSwapFee: rpcData.HookSwapFee,
		Bins:        withActiveBin(snap.bins, snap.activeID),
	})
	if err != nil {
		return entity.Pool{}, 0, err
	}

	p.SwapFee = float64(rpcData.SwapFee)
	p.Extra = string(extraBytes)
	p.Reserves = calculateReservesFromBins(snap.bins)
	p.Reserves = rescaleNativeReserves(valueobject.ChainID(t.config.ChainID), &p)
	p.BlockNumber = max(p.BlockNumber, snap.block)
	p.Timestamp = time.Now().Unix()
	return p, snap.block, nil
}

// lacksBin reports whether the stored set was missing a bin: the active bin, read now, holds
// reserves the stored set had no bin for, and no Mint of the batch put them there. A swap only
// moves through bins that hold liquidity, so it can only have ended in one the set lacked.
func lacksBin(stored, read []Bin, minted map[uint32]struct{}, activeID uint32) bool {
	if _, ok := minted[activeID]; ok {
		return false
	}
	if _, err := GetBinById(filterEmptyAndSortBins(read), activeID); err != nil {
		return false // empty, or not read
	}
	_, err := GetBinById(filterEmptyAndSortBins(stored), activeID)
	return err != nil
}

// readAllBins reads the pool's bins again once the stored set is found to lack one. Without the
// lens nothing can find a bin the set never had, and merged stays; the lens reads every non-empty
// bin at block, and its set replaces the stored one.
//
// The lens read goes through the guard, so it never makes the update wait: when the guard declines
// it, or it fails, merged stays.
func (t *PoolTracker) readAllBins(ctx context.Context, poolID string, block uint64, activeID uint32,
	merged []Bin) []Bin {
	l := logger.WithFields(logger.Fields{"address": poolID, "block": block})
	if !t.config.UseLens {
		l.Warn("stored bins lack a bin; without the lens they are kept as they are")
		return merged
	}

	var bins []Bin
	ran, err := t.lensGuard.Try(ctx, poolID, func(ctx context.Context) error {
		snap, err := t.snapshot(ctx, poolID, new(big.Int).SetUint64(block))
		if err != nil {
			return err
		}
		if snap.activeID != activeID {
			return fmt.Errorf("lens read active bin %d at block %d, RPC %d", snap.activeID, snap.block, activeID)
		}
		bins = snap.bins
		return nil
	})
	if ran && err == nil {
		return bins
	}
	l.WithFields(logger.Fields{"ran": ran, "error": err}).
		Warn("bins not read through the lens; keeping the stored ones")
	return merged
}

var _ poolpkg.IPoolTrackerWithSchedule = (*PoolTracker)(nil)

// NextUpdateAt asks for a pool still without bins, its lens read declined by the guard, to be
// updated again as soon as the guard would take it: it may see no log for a long time, and only an
// update gives it its bins. Every other pool needs no update without an event.
func (t *PoolTracker) NextUpdateAt(p entity.Pool) int64 {
	if t.config.UseLens && !hasState(p) {
		return t.lensGuard.ReadyAt(p.Address, time.Now()).Unix()
	}
	return 0
}

var _ poolpkg.IPoolTrackerWithReset = (*PoolTracker)(nil)

// ResetPoolState drops the pool's bins for its next update to read them whole through the lens, at a
// block no lower than block. Without the lens nothing can read them whole, and the pool is kept.
func (t *PoolTracker) ResetPoolState(p entity.Pool, block uint64) (entity.Pool, bool) {
	if !t.config.UseLens {
		return p, false
	}
	extra := map[string]json.RawMessage{}
	_ = json.Unmarshal([]byte(p.Extra), &extra)
	delete(extra, "bins")
	extra["lensDue"] = json.RawMessage(strconv.FormatUint(max(tickspkg.LensDue(p), block), 10))
	b, err := json.Marshal(extra)
	if err != nil {
		return p, false
	}
	p.Extra = string(b)
	return p, true
}
