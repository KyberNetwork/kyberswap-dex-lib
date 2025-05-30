package liquiditybookv21

import (
	"context"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv20"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	cfg           *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

var _ = pooltrack.RegisterFactoryCEG(DexTypeLiquidityBookV21, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		cfg:           cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}, nil
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

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}

func (d *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*QueryRpcPoolStateResult, error) {
	var (
		blockTimestamp uint64
		binStep        uint16

		staticFeeParamsResp   staticFeeParamsResp
		variableFeeParamsResp variableFeeParamsResp

		reserves    reserves
		activeBinID *big.Int

		priceX128 *big.Int

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
		Method: pairMethodGetStaticFeeParameters,
	}, []interface{}{&staticFeeParamsResp})

	req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodGetVariableFeeParameters,
	}, []interface{}{&variableFeeParamsResp})

	req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodGetReserves,
	}, []interface{}{&reserves})

	req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodGetActiveID,
	}, []interface{}{&activeBinID})

	req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodGetBinStep,
	}, []interface{}{&binStep})

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	req = d.ethrpcClient.R().SetContext(ctx)
	if blockTimestamp, err = req.GetCurrentBlockTimestamp(); err != nil {
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

	req = d.ethrpcClient.NewRequest()
	req.SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		req.SetBlockNumber(&blockNumberBI)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairGetPriceFromID,
		Params: []interface{}{activeBinID},
	}, []interface{}{&priceX128})

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	return &QueryRpcPoolStateResult{
		BlockTimestamp:    blockTimestamp,
		StaticFeeParams:   staticFeeParams,
		VariableFeeParams: variableFeeParams,
		Reserves:          reserves,
		ActiveBinID:       uint32(activeBinID.Uint64()),
		BinStep:           binStep,
		Liquidity:         liquiditybookv20.CalculateLiquidity(priceX128, reserves.ReserveX, reserves.ReserveY),
		PriceX128:         priceX128,
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
			decimalX, err := strconv.ParseInt(resp.Pair.TokenX.Decimals, 10, 64)
			if err != nil {
				return nil, err
			}
			unitX = bignumber.TenPowDecimals(uint8(decimalX))
		}
		if unitY == nil {
			decimalY, err := strconv.ParseInt(resp.Pair.TokenY.Decimals, 10, 64)
			if err != nil {
				return nil, err
			}
			unitY = bignumber.TenPowDecimals(uint8(decimalY))
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
