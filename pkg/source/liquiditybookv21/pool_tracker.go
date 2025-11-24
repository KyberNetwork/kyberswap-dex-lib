package liquiditybookv21

import (
	"context"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv20"
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

var _ = pooltrack.RegisterFactoryCEG0(DexTypeLiquidityBookV21, NewPoolTracker)
var _ = pooltrack.RegisterTicksBasedFactoryCEG0(DexTypeLiquidityBookV21, NewPoolTracker)

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

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool,
	error) {
	l := log.Ctx(ctx).With().Str("address", p.Address).Str("exchange", p.Exchange).Logger()
	l.Info().Msg("GetNewPoolState starts")

	var (
		rpcData        *QueryRpcPoolStateResult
		subgraphResult *querySubgraphPoolStateResult
		err            error
	)

	g := new(errgroup.Group)
	g.Go(func() error {
		rpcData, err = t.FetchRPCData(ctx, &p, 0)
		if err != nil {
			return err
		}
		return nil
	})
	g.Go(func() error {
		subgraphResult, err = t.querySubgraph(ctx, p)
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
		StaticFeeParams:        rpcData.StaticFeeParams,
		VariableFeeParams:      rpcData.VariableFeeParams,
		ActiveBinID:            rpcData.ActiveBinID,
		BinStep:                rpcData.BinStep,
		Bins:                   subgraphResult.Bins,
		PriceX128:              rpcData.PriceX128,
		Liquidity:              rpcData.Liquidity,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{
		rpcData.Reserves.ReserveX.String(),
		rpcData.Reserves.ReserveY.String(),
	}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	l.Info().Msg("GetNewPoolState finished")
	return p, nil
}

func (t *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*QueryRpcPoolStateResult,
	error) {
	var (
		binStep               uint16
		staticFeeParamsResp   staticFeeParamsResp
		variableFeeParamsResp variableFeeParamsResp
		reserves              reserves
		activeBinID           *big.Int
		priceX128             *big.Int
		blockNumberBI         *big.Int
	)

	if blockNumber > 0 {
		blockNumberBI = big.NewInt(int64(blockNumber))
	}

	if _, err := t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(blockNumberBI).AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodGetStaticFeeParameters,
	}, []any{&staticFeeParamsResp}).AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodGetVariableFeeParameters,
	}, []any{&variableFeeParamsResp}).AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodGetReserves,
	}, []any{&reserves}).AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodGetActiveID,
	}, []any{&activeBinID}).AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodGetBinStep,
	}, []any{&binStep}).Aggregate(); err != nil {
		return nil, err
	}

	// params
	staticFeeParams := staticFeeParams{
		BaseFactor:               staticFeeParamsResp.BaseFactor,
		FilterPeriod:             staticFeeParamsResp.FilterPeriod,
		DecayPeriod:              staticFeeParamsResp.DecayPeriod,
		ReductionFactor:          staticFeeParamsResp.ReductionFactor,
		VariableFeeControl:       uint32(staticFeeParamsResp.VariableFeeControl.Uint64()),
		ProtocolShare:            staticFeeParamsResp.ProtocolShare,
		MaxVolatilityAccumulator: uint32(staticFeeParamsResp.MaxVolatilityAccumulator.Uint64()),
	}
	variableFeeParams := variableFeeParams{
		VolatilityAccumulator: uint32(variableFeeParamsResp.VolatilityAccumulator.Uint64()),
		VolatilityReference:   uint32(variableFeeParamsResp.VolatilityReference.Uint64()),
		IdReference:           uint32(variableFeeParamsResp.IdReference.Uint64()),
		TimeOfLastUpdate:      variableFeeParamsResp.TimeOfLastUpdate.Uint64(),
	}

	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumberBI).AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairGetPriceFromID,
		Params: []any{activeBinID},
	}, []any{&priceX128}).Call(); err != nil {
		return nil, err
	}

	return &QueryRpcPoolStateResult{
		BlockTimestamp:    uint64(time.Now().Unix()),
		StaticFeeParams:   staticFeeParams,
		VariableFeeParams: variableFeeParams,
		Reserves:          reserves,
		ActiveBinID:       uint32(activeBinID.Uint64()),
		BinStep:           binStep,
		Liquidity:         liquiditybookv20.CalculateLiquidity(priceX128, reserves.ReserveX, reserves.ReserveY),
		PriceX128:         priceX128,
	}, nil
}

func (t *PoolTracker) querySubgraph(ctx context.Context, p entity.Pool) (*querySubgraphPoolStateResult, error) {
	l := log.Ctx(ctx).With().Str("address", p.Address).Str("exchange", p.Exchange).Logger()
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

		if err := t.graphqlClient.Run(ctx, req, &resp); err != nil {
			if !t.cfg.AllowSubgraphError || resp.Pair == nil {
				l.Err(err).Msg("failed to query subgraph")
				return nil, err
			}
		}
		resp.Meta.CheckIsLagging(t.cfg.DexID, p.Address)

		// init value
		if blockTimestamp == 0 && resp.Meta != nil {
			blockTimestamp = resp.Meta.Block.Timestamp
		}

		// if no bin returned, stop
		if resp.Pair == nil || len(resp.Pair.Bins) == 0 {
			l.Info().Msg("no bin returned")
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
	l := log.Ctx(ctx).With().Str("address", p.Address).Str("exchange", p.Exchange).Logger()

	if err := t.updateStateByDexLib(ctx, &p, logs); err != nil {
		l.Err(err).Msg(ErrUpdateStateByDexLibFailed.Error())
		return p, errors.Wrap(ErrUpdateStateByDexLibFailed, err.Error())
	}
	if err := t.updateBinsData(ctx, &p, logs); err != nil {
		l.Err(err).Msg(ErrUpdateBinsDataFailed.Error())
		return p, errors.Wrap(ErrUpdateBinsDataFailed, err.Error())
	}

	p.Timestamp = time.Now().Unix()
	return p, nil
}

func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	l := log.Ctx(ctx).With().Str("address", p.Address).Str("exchange", p.Exchange).Logger()

	var extra Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			l.Err(err).Msg("failed to unmarshal extra")
			return p, err
		}
	}

	binIDs := lo.Map(extra.Bins, func(item Bin, _ int) uint32 {
		return item.ID
	})
	bins, err := t.queryRPCBins(ctx, p.Address, binIDs, p.BlockNumber)
	if err != nil {
		l.Err(err).Msg("failed to query RPC bins")
		return p, err
	}
	bins = filterEmptyAndSortBins(bins)

	extra.Bins = bins
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.Err(err).Msg("failed to marshal extra")
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) updateStateByDexLib(ctx context.Context, p *entity.Pool, logs []ethtypes.Log) error {
	l := log.Ctx(ctx).With().Str("address", p.Address).Str("exchange", p.Exchange).Logger()

	var blockNumber uint64
	if len(logs) > 0 {
		blockNumber = logs[len(logs)-1].BlockNumber
	}

	rpcState, err := t.FetchRPCData(ctx, p, blockNumber)
	if err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			rpcState, err = t.FetchRPCData(ctx, p, 0)
			if err != nil {
				l.Err(err).Msg("failed to FetchRPCData")
				return err
			}
		} else {
			l.Err(err).Uint64("blockNumber", blockNumber).Msg("failed to FetchRPCData")
			return err
		}
	}

	p.Reserves = entity.PoolReserves{
		rpcState.Reserves.ReserveX.String(),
		rpcState.Reserves.ReserveY.String(),
	}

	var extra Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			l.Err(err).Msg("failed to unmarshal extra")
			return err
		}
	}

	extra.RpcBlockTimestamp = rpcState.BlockTimestamp
	extra.StaticFeeParams = rpcState.StaticFeeParams
	extra.VariableFeeParams = rpcState.VariableFeeParams
	extra.ActiveBinID = rpcState.ActiveBinID
	extra.BinStep = rpcState.BinStep
	extra.PriceX128 = rpcState.PriceX128
	extra.Liquidity = rpcState.Liquidity

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.Err(err).Msg("failed to marshal extra")
		return err
	}

	p.Extra = string(extraBytes)

	return nil
}

func (t *PoolTracker) updateBinsData(ctx context.Context, p *entity.Pool, logs []ethtypes.Log) error {
	if len(logs) == 0 {
		return nil
	}
	l := log.Ctx(ctx).With().Str("address", p.Address).Str("exchange", p.Exchange).Logger()
	blockNumber := eth.GetBlockNumberFromLogs(logs)

	binIDsFromLogs, err := t.binIDsFromLogs(l, logs)
	if err != nil {
		return err
	}

	binsFromLogs, err := t.queryRPCBins(ctx, p.Address, binIDsFromLogs, blockNumber)
	if err != nil {
		return err
	}

	bins, err := t.mergeBinsFromLogsToPoolBins(ctx, p, binsFromLogs)
	if err != nil {
		return err
	}

	if err := t.validateBins(ctx, p, bins); err != nil {
		bins, err = t.fetchAllBins(ctx, p, binsFromLogs, blockNumber)
		if err != nil {
			return err
		}

		_ = t.validateBins(ctx, p, bins)
	}

	var extra Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			l.Err(err).Msg("failed to unmarshal extra")
			return err
		}
	}

	extra.Bins = bins
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.Err(err).Msg("failed to marshal extra")
		return err
	}
	p.Extra = string(extraBytes)

	return nil
}

func (t *PoolTracker) fetchAllBins(ctx context.Context, p *entity.Pool, binsFromLogs []Bin, blockNumber uint64) ([]Bin,
	error) {
	l := log.Ctx(ctx).With().Str("address", p.Address).Str("exchange", p.Exchange).Logger()

	isBinFromLogs := map[uint32]struct{}{}
	lo.ForEach(binsFromLogs, func(item Bin, _ int) {
		isBinFromLogs[item.ID] = struct{}{}
	})

	var extra Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			l.Err(err).Msg("failed to unmarshal extra")
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

func (t *PoolTracker) validateBins(ctx context.Context, p *entity.Pool, bins []Bin) error {
	l := log.Ctx(ctx).With().Str("address", p.Address).Str("exchange", p.Exchange).Logger()

	binsReserveX := lo.Reduce(bins, func(agg *big.Int, item Bin, _ int) *big.Int {
		return agg.Add(agg, item.ReserveX)
	}, new(big.Int))

	binsReserveY := lo.Reduce(bins, func(agg *big.Int, item Bin, _ int) *big.Int {
		return agg.Add(agg, item.ReserveY)
	}, new(big.Int))

	reserveX, ok := new(big.Int).SetString(p.Reserves[0], 10)
	if !ok {
		l.Err(ErrInvalidReserve).Msg("can not parse reserve X")
		return ErrInvalidReserve
	}

	reserveY, ok := new(big.Int).SetString(p.Reserves[1], 10)
	if !ok {
		l.Err(ErrInvalidReserve).Msg("can not parse reserve Y")
		return ErrInvalidReserve
	}

	if binsReserveX.Cmp(reserveX) != 0 || binsReserveY.Cmp(reserveY) != 0 {
		l.Err(ErrInvalidReserve).Msg("reserves are not equal to the total reserves of bins")
		return ErrInvalidReserve
	}

	return nil
}

func (t *PoolTracker) queryRPCBins(ctx context.Context, poolAddress string, binIDs []uint32, blockNumber uint64) ([]Bin,
	error) {
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

func (t *PoolTracker) queryRPCBinsByChunk(ctx context.Context, poolAddress string, binIDs []uint32,
	blockNumber uint64) ([]Bin, error) {
	if len(binIDs) == 0 {
		return []Bin{}, nil
	}
	l := log.Ctx(ctx).With().Str("address", poolAddress).Logger()

	req := t.ethrpcClient.R().SetContext(ctx)
	if blockNumber > 0 {
		req.SetBlockNumber(big.NewInt(int64(blockNumber)))
	}
	bins := make([]bin, len(binIDs))
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

		l.Err(err).Msg("failed to query RPC bin chunk")
		return nil, err
	}

	return lo.Map(bins, func(_ bin, idx int) Bin {
		return Bin{
			ID:       binIDs[idx],
			ReserveX: bins[idx].BinReserveX,
			ReserveY: bins[idx].BinReserveY,
		}
	}), nil
}

func (t *PoolTracker) mergeBinsFromLogsToPoolBins(ctx context.Context, p *entity.Pool, binsFromLogs []Bin) ([]Bin,
	error) {
	l := log.Ctx(ctx).With().Str("address", p.Address).Str("exchange", p.Exchange).Logger()

	bins := binsFromLogs

	isBinFromLogs := map[uint32]struct{}{}
	lo.ForEach(binsFromLogs, func(item Bin, _ int) {
		isBinFromLogs[item.ID] = struct{}{}
	})

	var extra Extra
	if p.Extra != "" {
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			l.Err(err).Msg("failed to unmarshal extra")
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

func (t *PoolTracker) binIDsFromLogs(l zerolog.Logger, logs []ethtypes.Log) ([]uint32, error) {
	binSet := map[uint64]struct{}{}

	for _, event := range logs {
		if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) {
			continue
		}

		l := l.With().Uint64("blockNumber", event.BlockNumber).
			Stringer("blockHash", event.BlockHash).
			Uint("logIndex", event.Index).Logger()

		switch event.Topics[0] {
		case pairABI.Events["Swap"].ID:
			swap, err := pairFilterer.ParseSwap(event)
			if err != nil {
				l.Err(err).Msg("failed to parse Swap event")
				return nil, err
			}
			binSet[swap.Id.Uint64()] = struct{}{}

		case ppairABI.Events["Swap0"].ID:
			swap, err := pairFilterer.ParseSwap0(event)
			if err != nil {
				l.Err(err).Msg("failed to parse Swap event")
				return nil, err
			}
			binSet[swap.Id.Uint64()] = struct{}{}

		case pairABI.Events["DepositedToBins"].ID:
			depositedToBins, err := pairFilterer.ParseDepositedToBins(event)
			if err != nil {
				l.Err(err).Msg("failed to parse DepositedToBins event")
				return nil, err
			}
			for _, id := range depositedToBins.Ids {
				binSet[id.Uint64()] = struct{}{}
			}

		case pairABI.Events["DepositedToBins0"].ID:
			depositedToBins, err := pairFilterer.ParseDepositedToBins0(event)
			if err != nil {
				l.Err(err).Msg("failed to parse DepositedToBins event")
				return nil, err
			}
			for _, id := range depositedToBins.Ids {
				binSet[id.Uint64()] = struct{}{}
			}

		case pairABI.Events["WithdrawnFromBins"].ID:
			withdrawnFromBins, err := pairFilterer.ParseWithdrawnFromBins(event)
			if err != nil {
				l.Err(err).Msg("failed to parse WithdrawnFromBins event")
				return nil, err
			}
			for _, id := range withdrawnFromBins.Ids {
				binSet[id.Uint64()] = struct{}{}
			}

		case pairABI.Events["WithdrawnFromBins0"].ID:
			withdrawnFromBins, err := pairFilterer.ParseWithdrawnFromBins0(event)
			if err != nil {
				l.Err(err).Msg("failed to parse WithdrawnFromBins event")
				return nil, err
			}
			for _, id := range withdrawnFromBins.Ids {
				binSet[id.Uint64()] = struct{}{}
			}

		case pairABI.Events["FlashLoan"].ID:
			flashLoan, err := pairFilterer.ParseFlashLoan(event)
			if err != nil {
				l.Err(err).Msg("failed to parse FlashLoan event")
				return nil, err
			}
			binSet[flashLoan.ActiveId.Uint64()] = struct{}{}

		case pairABI.Events["FlashLoan0"].ID:
			flashLoan, err := pairFilterer.ParseFlashLoan0(event)
			if err != nil {
				l.Err(err).Msg("failed to parse FlashLoan event")
				return nil, err
			}
			binSet[flashLoan.ActiveId.Uint64()] = struct{}{}

		case pairABI.Events["TransferBatch"].ID:
			transferBatch, err := pairFilterer.ParseTransferBatch(event)
			if err != nil {
				l.Err(err).Msg("failed to parse TransferBatch event")
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
		return item.ReserveX.Sign() > 0 || item.ReserveY.Sign() > 0
	})
	sort.Slice(b, func(i, j int) bool {
		return b[i].ID < b[j].ID
	})
	return b
}
