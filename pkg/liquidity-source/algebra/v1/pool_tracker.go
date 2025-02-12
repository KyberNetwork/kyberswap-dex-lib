package algebrav1

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	algebra.PoolTracker[Timepoint, TimepointRPC]
	config        *Config
	graphqlClient *graphqlpkg.Client
}

var _ = pooltrack.RegisterFactoryCEG(DexTypeAlgebraV1, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		PoolTracker:   algebra.PoolTracker[Timepoint, TimepointRPC]{EthrpcClient: ethrpcClient},
		config:        cfg,
		graphqlClient: graphqlClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	param sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	l.Info("Start getting new state of pool")

	var (
		rpcData   FetchRPCResult
		poolTicks []TickResp
	)

	blockNumber, err := d.EthrpcClient.GetBlockNumber(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get block number")
		return entity.Pool{}, err
	}

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = d.fetchRPCData(ctx, p, 0)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to fetch data from RPC")

		}

		return err
	})
	g.Go(func(context.Context) error {
		var err error
		if d.config.AlwaysUseTickLens {
			poolTicks, err = d.getPoolTicksFromSC(ctx, p, param)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to call SC for pool ticks")
			}
			return err
		}

		poolTicks, err = d.getPoolTicks(ctx, p.Address)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to query subgraph for pool ticks")
		}

		return err
	})

	if err := g.Wait(); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool state")
		return entity.Pool{}, err
	}

	ticks := make([]v3Entities.Tick, 0, len(poolTicks))
	for _, tickResp := range poolTicks {
		tick, err := transformTickRespToTickBigInt(tickResp)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to transform tickResp to tick")
			continue
		}

		// LiquidityGross = 0 means that the tick is uninitialized
		if tick.LiquidityGross.Cmp(bignumber.ZeroBI) == 0 {
			continue
		}

		ticks = append(ticks, tick)
	}

	extraBytes, err := json.Marshal(Extra{
		Liquidity:   rpcData.Liquidity,
		GlobalState: rpcData.State,
		Ticks:       ticks,
		TickSpacing: int24(rpcData.TickSpacing.Int64()),
	})

	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		rpcData.Reserve0.String(),
		rpcData.Reserve1.String(),
	}
	p.BlockNumber = blockNumber

	l.WithFields(logger.Fields{
		"feeZto": rpcData.State.FeeZto,
		"feeOtz": rpcData.State.FeeOtz,
	}).Info("Finish updating state of pool")

	return p, nil
}

func (d *PoolTracker) FetchStateFromRPC(ctx context.Context, p entity.Pool, blockNumber uint64) ([]byte, error) {
	rpcData, err := d.fetchRPCData(ctx, p, blockNumber)
	if err != nil {
		return nil, err
	}

	rpcDataBytes, err := json.Marshal(rpcData)
	if err != nil {
		return nil, err
	}

	return rpcDataBytes, nil
}

func (d *PoolTracker) fetchRPCData(ctx context.Context, p entity.Pool, blockNumber uint64) (FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	var (
		dataStorageOperator common.Address
	)
	res := FetchRPCResult{}

	rpcRequest := d.EthrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		rpcRequest.SetBlockNumber(&blockNumberBI)
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: p.Address,
		Method: methodGetLiquidity,
		Params: nil,
	}, []interface{}{&res.Liquidity})

	// the globalstate abi are slightly different across versions
	var rpcState interface{}
	if d.config.UseDirectionalFee {
		rpcState = &rpcGlobalStateDirFee{}
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    algebraV1DirFeePoolABI,
			Target: p.Address,
			Method: methodGetGlobalState,
			Params: nil,
		}, []interface{}{rpcState})
	} else {
		rpcState = &rpcGlobalStateSingleFee{}
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    algebraV1PoolABI,
			Target: p.Address,
			Method: methodGetGlobalState,
			Params: nil,
		}, []interface{}{rpcState})
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: p.Address,
		Method: methodGetDataStorageOperator,
		Params: nil,
	}, []interface{}{&dataStorageOperator})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: p.Address,
		Method: methodGetTickSpacing,
		Params: nil,
	}, []interface{}{&res.TickSpacing})

	if len(p.Tokens) == 2 {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[0].Address,
			Method: erc20MethodBalanceOf,
			Params: []interface{}{common.HexToAddress(p.Address)},
		}, []interface{}{&res.Reserve0})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[1].Address,
			Method: erc20MethodBalanceOf,
			Params: []interface{}{common.HexToAddress(p.Address)},
		}, []interface{}{&res.Reserve1})
	}

	_, err := rpcRequest.Aggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to process tryAggregate")
		return res, err
	}

	if d.config.UseDirectionalFee {
		rpcStateRes := rpcState.(*rpcGlobalStateDirFee)
		res.State = GlobalState{
			Price:              rpcStateRes.Price,
			Tick:               rpcStateRes.Tick,
			FeeZto:             rpcStateRes.FeeZto,
			FeeOtz:             rpcStateRes.FeeOtz,
			TimepointIndex:     rpcStateRes.TimepointIndex,
			CommunityFeeToken0: rpcStateRes.CommunityFeeToken0,
			CommunityFeeToken1: rpcStateRes.CommunityFeeToken1,
			Unlocked:           rpcStateRes.Unlocked,
		}
	} else {
		// for v1 without directional fee, we'll use Fee for both FeeZto/FeeOtz
		rpcStateRes := rpcState.(*rpcGlobalStateSingleFee)
		res.State = GlobalState{
			Price:              rpcStateRes.Price,
			Tick:               rpcStateRes.Tick,
			FeeZto:             rpcStateRes.Fee,
			FeeOtz:             rpcStateRes.Fee,
			TimepointIndex:     rpcStateRes.TimepointIndex,
			CommunityFeeToken0: rpcStateRes.CommunityFeeToken0,
			CommunityFeeToken1: rpcStateRes.CommunityFeeToken1,
			Unlocked:           rpcStateRes.Unlocked,
		}
	}

	if !d.config.SkipFeeCalculating {
		err = d.approximateFee(ctx, p.Address, dataStorageOperator.Hex(), &res.State, res.Liquidity)
		if err != nil {
			return res, err
		}
	}

	if !res.State.Unlocked {
		l.Info("pool has been locked and not usable")
	}

	return res, err
}

func (d *PoolTracker) approximateFee(ctx context.Context, poolAddress, dataStorageOperator string,
	state *GlobalState, currentLiquidity *big.Int) error {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       d.config.DexID,
	})

	// fee approximation: assume that the swap will be soon after this
	blockTimestamp := uint32(time.Now().Unix())
	yesterday := blockTimestamp - WINDOW
	timepoints, err := d.getPoolTimepoints(ctx, state.TimepointIndex, poolAddress, yesterday)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool timepoints")
		return err
	}

	if timepoints == nil {
		// not initialized pool has been locked already, but set here just for sure
		state.Unlocked = false
		return nil
	}

	feeConf := FeeConfiguration{}
	feeConfZto := FeeConfiguration{}
	feeConfOtz := FeeConfiguration{}
	if d.config.UseDirectionalFee {
		err = d.getPoolDirectionalFeeConfig(ctx, dataStorageOperator, &feeConfZto, &feeConfOtz)
	} else {
		err = d.getPoolFeeConfig(ctx, dataStorageOperator, &feeConf)
	}
	if err != nil {
		return err
	}

	volumePerLiquidityInBlock, err := d.getPoolVolumePerLiquidityInBlock(ctx, common.HexToAddress(poolAddress))
	if err != nil {
		return err
	}

	ts := TimepointStorage{
		data:    timepoints,
		updates: map[uint16]Timepoint{},
	}
	currentTick := int24(state.Tick.Int64())
	newTimepointIndex, err := ts.write(
		state.TimepointIndex,
		blockTimestamp,
		currentTick,
		currentLiquidity,
		volumePerLiquidityInBlock,
	)
	if err != nil {
		return err
	}

	if d.config.UseDirectionalFee {
		state.FeeZto, err = ts._getNewFee(blockTimestamp, currentTick, newTimepointIndex, currentLiquidity, &feeConfZto)
		if err != nil {
			return err
		}
		state.FeeOtz, err = ts._getNewFee(blockTimestamp, currentTick, newTimepointIndex, currentLiquidity, &feeConfOtz)
		if err != nil {
			return err
		}
	} else {
		state.FeeZto, err = ts._getNewFee(blockTimestamp, currentTick, newTimepointIndex, currentLiquidity, &feeConf)
		if err != nil {
			return err
		}
		state.FeeOtz = state.FeeZto
	}
	return nil
}

func (d *PoolTracker) getPoolFeeConfig(ctx context.Context, dataStorageOperatorAddress string,
	feeConf *FeeConfiguration) error {
	rpcRequest := d.EthrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1DataStorageOperatorAPI,
		Target: dataStorageOperatorAddress,
		Method: methodGetFeeConfig,
		Params: nil,
	}, []interface{}{feeConf})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dataStorageAddress": dataStorageOperatorAddress,
			"error":              err,
		}).Error("failed to fetch from data storage operator")
		return err
	}
	return nil
}

func (d *PoolTracker) getPoolDirectionalFeeConfig(ctx context.Context, dataStorageOperatorAddress string,
	feeConfZto *FeeConfiguration, feeConfOtz *FeeConfiguration) error {
	rpcRequest := d.EthrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1DirFeeDataStorageOperatorAPI,
		Target: dataStorageOperatorAddress,
		Method: methodGetFeeConfigZto,
		Params: nil,
	}, []interface{}{feeConfZto},
	).AddCall(&ethrpc.Call{
		ABI:    algebraV1DirFeeDataStorageOperatorAPI,
		Target: dataStorageOperatorAddress,
		Method: methodGetFeeConfigOtz,
		Params: nil,
	}, []interface{}{feeConfOtz})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dataStorageAddress": dataStorageOperatorAddress,
			"error":              err,
		}).Error("failed to fetch from data storage operator")
		return err
	}
	return nil
}

func (d *PoolTracker) getPoolTimepoints(ctx context.Context, currentIndex uint16, poolAddress string,
	yesterday uint32) (map[uint16]Timepoint, error) {
	return d.GetTimepoints(ctx, &ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: poolAddress,
		Method: methodGetTimepoints,
	}, nil, yesterday, currentIndex, nil)
}

func (d *PoolTracker) getPoolVolumePerLiquidityInBlock(ctx context.Context, poolAddress common.Address) (*big.Int,
	error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       d.config.DexID,
	})

	abiUint256, _ := abi.NewType("uint256", "", nil)
	abi := abi.Arguments{
		// 2 variables are stored in 1 slot, need to read the whole and shift out later
		{Name: "liquidity_volumePerLiquidityInBlock", Type: abiUint256},
	}

	resp, err := d.EthrpcClient.NewRequest().SetContext(ctx).GetStorageAt(
		poolAddress,
		slot3,
		abi,
	)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool volumePerLiquidityInBlock")
		return nil, err
	}

	if len(resp) == 1 {
		if bi, ok := resp[0].(*big.Int); ok {
			return new(big.Int).Rsh(bi, 128), nil
		}
	}
	l.WithFields(logger.Fields{
		"resp": resp,
	}).Error("failed to unmarshal volumePerLiquidityInBlock")
	return nil, ErrUnmarshalVolLiq
}

func (d *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       d.config.DexID,
	})

	allowSubgraphError := d.config.AllowSubgraphError
	skip := 0
	var ticks []TickResp

	for {
		req := graphqlpkg.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, skip))

		var resp struct {
			Pool *SubgraphPoolTicks        `json:"pool"`
			Meta *valueobject.SubgraphMeta `json:"_meta"`
		}

		if err := d.graphqlClient.Run(ctx, req, &resp); err != nil {
			if allowSubgraphError {
				if resp.Pool == nil {
					l.WithFields(logger.Fields{
						"error":              err,
						"allowSubgraphError": allowSubgraphError,
					}).Error("failed to query subgraph")

					return nil, err
				}
			} else {
				l.WithFields(logger.Fields{
					"error":              err,
					"allowSubgraphError": allowSubgraphError,
				}).Error("failed to query subgraph")

				return nil, err
			}
		}
		resp.Meta.CheckIsLagging(d.config.DexID, poolAddress)

		if resp.Pool == nil || len(resp.Pool.Ticks) == 0 {
			break
		}

		ticks = append(ticks, resp.Pool.Ticks...)

		if len(resp.Pool.Ticks) < graphFirstLimit {
			break
		}

		skip += len(resp.Pool.Ticks)
		if skip > graphSkipLimit {
			logger.Infoln("hit skip limit, continue in next cycle")
			break
		}
	}

	return ticks, nil
}
