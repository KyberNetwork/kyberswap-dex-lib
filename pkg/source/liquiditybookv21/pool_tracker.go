package liquiditybookv21

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/machinebox/graphql"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	cfg           *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphql.Client
}

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	graphqlClient := graphqlPkg.NewWithTimeout(cfg.SubgraphAPI, graphQLRequestTimeout)

	return &PoolTracker{
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	rpcResult, err := d.queryRpc(ctx, p)
	if err != nil {
		return entity.Pool{}, err
	}

	subgraphResult, err := d.querySubgraph(ctx, p)
	if err != nil {
		return entity.Pool{}, err
	}

	extra := Extra{
		RpcBlockTimestamp:      rpcResult.BlockTimestamp,
		SubgraphBlockTimestamp: subgraphResult.BlockTimestamp,
		StaticFeeParams:        rpcResult.StaticFeeParams,
		VariableFeeParams:      rpcResult.VariableFeeParams,
		BinStep:                rpcResult.BinStep,
		Bins:                   subgraphResult.Bins,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{
		rpcResult.Reserves.ReserveX.String(),
		rpcResult.Reserves.ReserveY.String(),
	}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}

func (d *PoolTracker) queryRpc(ctx context.Context, p entity.Pool) (*queryRpcPoolStateResult, error) {
	var (
		blockTimestamp    uint64
		staticFeeParams   staticFeeParams
		variableFeeParams variableFeeParams
		reserves          reserves
		activeBinID       uint32
		binStep           uint16

		err error
	)

	req := d.ethrpcClient.R().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodGetStaticFeeParameters,
	}, []interface{}{&staticFeeParams})

	req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMethodGetVariableFeeParameters,
	}, []interface{}{&variableFeeParams})

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

	return &queryRpcPoolStateResult{
		BlockTimestamp:    blockTimestamp,
		StaticFeeParams:   staticFeeParams,
		VariableFeeParams: variableFeeParams,
		Reserves:          reserves,
		BinStep:           binStep,
	}, nil
}

func (d *PoolTracker) querySubgraph(ctx context.Context, p entity.Pool) (*querySubgraphPoolStateResult, error) {
	var (
		skip           = 0
		bins           []bin
		blockTimestamp int64
		unitX          *big.Float
		unitY          *big.Float
	)

	// bins
	for {
		// query
		var (
			query = buildQueryGetBins(p.Address, skip)
			req   = graphql.NewRequest(query)

			resp struct {
				Pair *lbpairSubgraphResp       `json:"lbpair"`
				Meta *valueobject.SubgraphMeta `json:"_meta"`
			}
		)

		if err := d.graphqlClient.Run(ctx, req, &resp); err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       err,
			}).Errorf("failed to query subgraph")
			return nil, err
		}
		resp.Meta.CheckIsLagging(d.cfg.DexID, p.Address)

		// init value
		if blockTimestamp == 0 {
			blockTimestamp = resp.Meta.Block.Timestamp
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

		// if no bin returned, stop
		if resp.Pair == nil || len(resp.Pair.Bins) == 0 {
			break
		}

		// transform
		b, err := transformSubgraphBins(resp.Pair.Bins, unitX, unitY)
		if err != nil {
			return nil, err
		}
		// TODO: remove empty bin
		bins = append(bins, b...)

		// for next cycle
		if len(resp.Pair.Bins) < graphFirstLimit {
			break
		}
		skip += len(resp.Pair.Bins)
		if skip > graphSkipLimit {
			logger.Infoln("hit skip limit, continue in next cycle")
			break
		}
	}

	return &querySubgraphPoolStateResult{
		BlockTimestamp: uint64(blockTimestamp),
		Bins:           bins,
	}, nil
}
