package liquiditybookv20

import (
	"context"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	cfg           *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

var _ = pooltrack.RegisterFactoryCEG0(DexTypeLiquidityBookV20, NewPoolTracker)
var _ = pooltrack.RegisterTicksBasedFactoryCEG0(DexTypeLiquidityBookV20, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolTracker {
	return &PoolTracker{
		cfg:           cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	var (
		rpcData        *QueryRpcPoolStateResult
		subgraphResult *querySubgraphPoolStateResult
		err            error
	)

	g := new(errgroup.Group)
	g.Go(func() error {
		rpcData, err = d.FetchRPCData(ctx, &p, 0)
		if err != nil {
			return err
		}
		return nil
	})
	g.Go(func() error {
		subgraphResult, err = d.querySubgraph(ctx, p)
		if err != nil {
			return err
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return entity.Pool{}, err
	}

	extra := Extra{
		RpcBlockTimestamp:      rpcData.BlockTimestamp,
		SubgraphBlockTimestamp: subgraphResult.BlockTimestamp,
		FeeParameters:          rpcData.FeeParameters,
		ActiveBinID:            uint32(rpcData.ReservesAndID.ActiveId.Uint64()),
		Bins:                   subgraphResult.Bins,
		PriceX128:              rpcData.PriceX128,
		Liquidity:              rpcData.Liquidity,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{
		rpcData.ReservesAndID.ReserveX.String(),
		rpcData.ReservesAndID.ReserveY.String(),
	}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}

func (d *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*QueryRpcPoolStateResult, error) {
	var (
		blockTimestamp uint64

		feeParamsResp feeParametersRpcResp
		reservesAndID reservesAndID
		priceX128     *big.Int

		err error
	)

	req := d.ethrpcClient.R().SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		req.SetBlockNumber(&blockNumberBI)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodFeeParameters,
	}, []any{&feeParamsResp})

	req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodGetReservesAndID,
	}, []any{&reservesAndID})

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	req = d.ethrpcClient.R().SetContext(ctx)
	if blockTimestamp, err = req.GetCurrentBlockTimestamp(); err != nil {
		return nil, err
	}

	req = d.ethrpcClient.R().SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		req.SetBlockNumber(&blockNumberBI)
	}
	req.AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: d.cfg.RouterAddress,
		Method: routerGetPriceFromIDMethod,
		Params: []any{
			common.HexToAddress(p.Address),
			reservesAndID.ActiveId,
		},
	}, []any{&priceX128})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	// params
	feeParameters := feeParameters{
		BinStep:                  feeParamsResp.State.BinStep,
		BaseFactor:               feeParamsResp.State.BaseFactor,
		FilterPeriod:             feeParamsResp.State.FilterPeriod,
		DecayPeriod:              feeParamsResp.State.DecayPeriod,
		ReductionFactor:          feeParamsResp.State.ReductionFactor,
		VariableFeeControl:       uint32(feeParamsResp.State.VariableFeeControl.Uint64()),
		ProtocolShare:            feeParamsResp.State.ProtocolShare,
		MaxVolatilityAccumulated: uint32(feeParamsResp.State.MaxVolatilityAccumulated.Uint64()),
		VolatilityAccumulated:    uint32(feeParamsResp.State.VolatilityAccumulated.Uint64()),
		VolatilityReference:      uint32(feeParamsResp.State.VolatilityReference.Uint64()),
		IndexRef:                 uint32(feeParamsResp.State.IndexRef.Uint64()),
		Time:                     uint64(feeParamsResp.State.Time.Uint64()),
	}

	return &QueryRpcPoolStateResult{
		BlockTimestamp: blockTimestamp,
		FeeParameters:  feeParameters,
		ReservesAndID:  reservesAndID,
		PriceX128:      priceX128,
		Liquidity:      CalculateLiquidity(priceX128, reservesAndID.ReserveX, reservesAndID.ReserveY),
	}, nil
}

func (d *PoolTracker) querySubgraph(ctx context.Context, p entity.Pool) (*querySubgraphPoolStateResult, error) {
	var (
		bins           []Bin
		blockTimestamp int64
		unitX          *big.Float
		unitY          *big.Float
		binIDGT        int64 = -1
	)

	// bins
	for {
		// query
		var (
			query = buildQueryGetBins(p.Address, binIDGT)
			req   = graphqlpkg.NewRequest(query)

			resp struct {
				Pair *lbpairSubgraphResp       `json:"lbpair"`
				Meta *valueobject.SubgraphMeta `json:"_meta"`
			}
		)

		if err := d.graphqlClient.Run(ctx, req, &resp); err != nil {
			if !d.cfg.AllowSubgraphError {
				logger.WithFields(logger.Fields{
					"poolAddress":        p.Address,
					"error":              err,
					"allowSubgraphError": d.cfg.AllowSubgraphError,
				}).Errorf("failed to query subgraph")
				return nil, err
			}

			if resp.Pair == nil {
				logger.WithFields(logger.Fields{
					"poolAddress":        p.Address,
					"error":              err,
					"allowSubgraphError": d.cfg.AllowSubgraphError,
				}).Errorf("failed to query subgraph")
				return nil, err
			}
		}
		resp.Meta.CheckIsLagging(d.cfg.DexID, p.Address)

		// init value
		if blockTimestamp == 0 && resp.Meta != nil {
			blockTimestamp = resp.Meta.Block.Timestamp
		}

		// if no bin returned, stop
		if resp.Pair == nil || len(resp.Pair.Bins) == 0 {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
			}).Info("no bin returned")
			break
		}

		if unitX == nil {
			decimalX, err := kutils.Atou[uint8](resp.Pair.TokenX.Decimals)
			if err != nil {
				return nil, err
			}
			unitX = bignumber.TenPowDecimals(decimalX)
		}
		if unitY == nil {
			decimalY, err := kutils.Atou[uint8](resp.Pair.TokenY.Decimals)
			if err != nil {
				return nil, err
			}
			unitY = bignumber.TenPowDecimals(decimalY)
		}

		// transform
		if len(resp.Pair.Bins) > 0 {
			b, err := transformSubgraphBins(resp.Pair.Bins, unitX, unitY)
			if err != nil {
				return nil, err
			}
			bins = append(bins, b...)
		}

		// for next cycle
		if len(resp.Pair.Bins) < graphFirstLimit {
			break
		}

		binIDGT = int64(bins[len(bins)-1].ID)
	}

	sort.Slice(bins, func(i, j int) bool {
		return bins[i].ID < bins[j].ID
	})

	return &querySubgraphPoolStateResult{
		BlockTimestamp: uint64(blockTimestamp),
		Bins:           bins,
	}, nil
}

func (t *PoolTracker) GetNewState(ctx context.Context, p entity.Pool, logs []ethtypes.Log,
	_ map[uint64]entity.BlockHeader) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	if err := t.updateStateByDexLib(ctx, &p, logs); err != nil {
		l.WithFields(logger.Fields{
			"msg": err.Error(),
		}).Error(ErrUpdateStateByDexLibFailed.Error())

		return p, errors.Wrap(ErrUpdateStateByDexLibFailed, err.Error())
	}

	if err := t.updateBinsData(ctx, &p, logs); err != nil {
		l.WithFields(logger.Fields{
			"msg": err.Error(),
		}).Error(ErrUpdateBinsDataFailed.Error())

		return p, errors.Wrap(ErrUpdateBinsDataFailed, err.Error())
	}

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

func (t *PoolTracker) updateStateByDexLib(ctx context.Context, p *entity.Pool, logs []ethtypes.Log) error {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	var blockNumber uint64
	if len(logs) > 0 {
		blockNumber = logs[len(logs)-1].BlockNumber
	}

	rpcState, err := t.FetchRPCData(ctx, p, blockNumber)
	if err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			rpcState, err = t.FetchRPCData(ctx, p, 0)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to fetch latest state from RPC")
				return err
			}
		} else {
			l.WithFields(logger.Fields{
				"error":       err,
				"blockNumber": blockNumber,
			}).Error("failed to fetch state from RPC")
			return err
		}
	}

	p.Reserves = entity.PoolReserves{
		rpcState.ReservesAndID.ReserveX.String(),
		rpcState.ReservesAndID.ReserveY.String(),
	}

	var extra Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			l.Error(err.Error())
			return err
		}
	}

	extra.RpcBlockTimestamp = rpcState.BlockTimestamp
	extra.FeeParameters = rpcState.FeeParameters
	extra.ActiveBinID = uint32(rpcState.ReservesAndID.ActiveId.Uint64())
	extra.PriceX128 = rpcState.PriceX128
	extra.Liquidity = rpcState.Liquidity

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	p.Extra = string(extraBytes)

	return nil
}

func (t *PoolTracker) updateBinsData(ctx context.Context, p *entity.Pool, logs []ethtypes.Log) error {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	if len(logs) == 0 {
		return nil
	}
	blockNumber := eth.GetBlockNumberFromLogs(logs)

	binIDsFromLogs, err := t.binIDsFromLogs(logs)
	if err != nil {
		return err
	}

	binsFromLogs, err := t.queryRPCBins(ctx, p.Address, binIDsFromLogs, blockNumber)
	if err != nil {
		return err
	}

	bins, err := t.mergeBinsFromLogsToPoolBins(p, binsFromLogs)
	if err != nil {
		return err
	}

	if err := t.validateBins(p, bins); err != nil {

		bins, err = t.fetchAllBins(ctx, p, binsFromLogs, blockNumber)
		if err != nil {
			return err
		}

		_ = t.validateBins(p, bins)
	}

	var extra Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			l.Error(err.Error())
			return err
		}
	}

	extra.Bins = bins
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.Error(err.Error())
		return err
	}
	p.Extra = string(extraBytes)

	return nil
}

func (t *PoolTracker) fetchAllBins(ctx context.Context, p *entity.Pool, binsFromLogs []Bin, blockNumber uint64) ([]Bin, error) {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	isBinFromLogs := map[uint32]struct{}{}
	lo.ForEach(binsFromLogs, func(item Bin, _ int) {
		isBinFromLogs[item.ID] = struct{}{}
	})

	var extra Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			l.Error(err.Error())
			return nil, err
		}
	}

	var binIDsFromPool []uint32
	for _, b := range extra.Bins {
		if _, ok := isBinFromLogs[b.ID]; ok {
			continue
		}
		binIDsFromPool = append(binIDsFromPool, b.ID)
	}
	binsFromPool, err := t.queryRPCBins(ctx, p.Address, binIDsFromPool, blockNumber)
	if err != nil {
		return nil, err
	}

	bins := append(binsFromPool, binsFromLogs...)

	return filterEmptyAndSortBins(bins), nil
}

func (t *PoolTracker) validateBins(p *entity.Pool, bins []Bin) error {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	binsReserveX := lo.Reduce(bins, func(agg *big.Int, item Bin, _ int) *big.Int {
		return agg.Add(agg, item.ReserveX)
	}, big.NewInt(0))

	binsReserveY := lo.Reduce(bins, func(agg *big.Int, item Bin, _ int) *big.Int {
		return agg.Add(agg, item.ReserveY)
	}, big.NewInt(0))

	reserveX, ok := new(big.Int).SetString(p.Reserves[0], 10)
	if !ok {
		l.WithFields(logger.Fields{
			"msg": "can not parse reserve X",
		}).Error(ErrInvalidReserve.Error())
		return ErrInvalidReserve
	}

	reserveY, ok := new(big.Int).SetString(p.Reserves[1], 10)
	if !ok {
		l.WithFields(logger.Fields{
			"msg": "can not parse reserve Y",
		}).Error(ErrInvalidReserve.Error())
		return ErrInvalidReserve
	}

	if binsReserveX.Cmp(reserveX) != 0 || binsReserveY.Cmp(reserveY) != 0 {
		l.WithFields(logger.Fields{
			"msg": "reserves are not equal to the total reserves of bins",
		}).Error(ErrInvalidReserve.Error())
		return ErrInvalidReserve
	}

	l.Debug("valid bins")

	return nil
}

func (t *PoolTracker) queryRPCBins(ctx context.Context, poolAddress string, binIDs []uint32, blockNumber uint64) ([]Bin, error) {
	if len(binIDs) == 0 {
		return []Bin{}, nil
	}

	var bins []Bin
	for from := 0; from < len(binIDs); {
		to := from + binChunk
		if to > len(binIDs) {
			to = len(binIDs)
		}

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
			ABI:    pairABI,
			Target: poolAddress,
			Method: pairMethodGetBin,
			Params: []any{big.NewInt(int64(binID))},
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
			ReserveX: bins[idx].ReserveX,
			ReserveY: bins[idx].ReserveY,
		}
	}), nil
}

func (t *PoolTracker) mergeBinsFromLogsToPoolBins(p *entity.Pool, binsFromLogs []Bin) ([]Bin, error) {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	bins := binsFromLogs

	isBinFromLogs := map[uint32]struct{}{}
	lo.ForEach(binsFromLogs, func(item Bin, _ int) {
		isBinFromLogs[item.ID] = struct{}{}
	})

	var extra Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			l.Error(err.Error())
			return nil, err
		}
	}

	for _, b := range extra.Bins {
		if _, ok := isBinFromLogs[b.ID]; ok {
			continue
		}
		bins = append(bins, b)
	}

	return filterEmptyAndSortBins(bins), nil
}

func (t *PoolTracker) binIDsFromLogs(logs []ethtypes.Log) ([]uint32, error) {
	binSet := map[uint64]struct{}{}

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
		case pairABI.Events["Swap"].ID:
			swap, err := pairFilterer.ParseSwap(event)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to parse Swap event")
				return nil, err
			}
			binSet[swap.Id.Uint64()] = struct{}{}

		case pairABI.Events["DepositedToBin"].ID:
			depositedToBin, err := pairFilterer.ParseDepositedToBin(event)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to parse DepositedToBins event")
				return nil, err
			}
			binSet[depositedToBin.Id.Uint64()] = struct{}{}

		case pairABI.Events["WithdrawnFromBin"].ID:
			withdrawnFromBin, err := pairFilterer.ParseWithdrawnFromBin(event)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to parse WithdrawnFromBins event")
				return nil, err
			}
			binSet[withdrawnFromBin.Id.Uint64()] = struct{}{}

		case pairABI.Events["TransferBatch"].ID:
			transferBatch, err := pairFilterer.ParseTransferBatch(event)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to parse TransferBatch event")
				return nil, err
			}
			for _, id := range transferBatch.Ids {
				binSet[id.Uint64()] = struct{}{}
			}
		}
	}

	var binIDs []uint32
	for binID := range binSet {
		binIDs = append(binIDs, uint32(binID))
	}

	return binIDs, nil
}

func filterEmptyAndSortBins(bins []Bin) []Bin {
	b := lo.Filter(bins, func(item Bin, _ int) bool {
		return item.ReserveX.Cmp(bignumber.ZeroBI) > 0 ||
			item.ReserveY.Cmp(bignumber.ZeroBI) > 0
	})
	sort.Slice(b, func(i, j int) bool {
		return b[i].ID < b[j].ID
	})
	return b
}
