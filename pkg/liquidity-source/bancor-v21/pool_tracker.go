package bancorv21

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type (
	PoolTracker struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	// 1. update anchors map and convertible tokens anchor state
	allPairsLength, err := getAllPairsLength(ctx, d.ethrpcClient, d.config.ConverterRegistry)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": d.config.DexID}).
			Error("getAllPairsLength failed")
		return p, err
	}
	latestPoolAddresses, latestAnchors, err := listPairAddresses(ctx, d.ethrpcClient, d.config.ConverterRegistry, allPairsLength)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": d.config.DexID, "err": err}).
			Error("listPairAddresses failed")
		return p, err
	}
	innerPools, tokensByAnchor, err := initInnerPools(ctx, d.ethrpcClient, latestPoolAddresses, latestAnchors)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": d.config.DexID, "err": err}).
			Error("initInnerPools failed")
		return p, err
	}

	reservesData, blockNumber, err := d.getReservesFromRPCNode(ctx, innerPools)
	if err != nil {
		return p, err
	}

	if p.BlockNumber > blockNumber.Uint64() {
		logger.
			WithFields(
				logger.Fields{
					"pool_id":           p.Address,
					"pool_block_number": p.BlockNumber,
					"data_block_number": blockNumber.Uint64(),
				},
			).
			Info("skip update: data block number is less than current pool block number")
		return p, nil
	}

	fees, err := d.getFee(ctx, innerPools)
	if err != nil {
		return p, err
	}

	defer func() {
		logger.
			WithFields(
				logger.Fields{
					"pool_id":          p.Address,
					"old_block_number": p.BlockNumber,
					"new_block_number": blockNumber,
					"duration_ms":      time.Since(startTime).Milliseconds(),
				},
			).
			Info("Finished getting new pool state")
	}()

	return d.updatePool(ctx, p, innerPools, latestAnchors, reservesData, fees, blockNumber, tokensByAnchor)
}

// getAllPairsLength gets number of pairs from the factory contracts
// nolint: unused
func (d *PoolTracker) getAnchorCount(ctx context.Context) (int, error) {
	anchorCount := new(big.Int)
	if _, err := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    converterRegistryABI,
		Target: d.config.ConverterRegistry,
		Method: getAnchorCount,
		Params: nil,
	}, []interface{}{&anchorCount}).Call(); err != nil {
		return 0, err
	}
	return int(anchorCount.Uint64()), nil
}

func (d *PoolTracker) updatePool(ctx context.Context, pool entity.Pool, innerPools []entity.Pool, anchors []common.Address, reserveData [][]*big.Int, fees []uint64, blockNumber *big.Int, tokensByAnchor map[string][]string) (entity.Pool, error) {
	pool.BlockNumber = blockNumber.Uint64()
	// 1. update inner pools fee and reserves for inner pools
	newInnerPools := make([]entity.Pool, len(innerPools))

	for i, innerPool := range innerPools {
		currentExtraInner := ExtraInner{}
		if err := json.Unmarshal([]byte(innerPool.Extra), &currentExtraInner); err != nil {
			return pool, err
		}
		currentExtraInner.ConversionFee = fees[i]

		extraBytes, err := json.Marshal(currentExtraInner)
		if err != nil {
			return innerPool, err
		}

		innerPool.Reserves = entity.PoolReserves{}
		for _, reserve := range reserveData[i] {
			innerPool.Reserves = append(innerPool.Reserves, reserve.String())
		}

		innerPool.Extra = string(extraBytes)
		innerPool.BlockNumber = blockNumber.Uint64()
		innerPool.Timestamp = time.Now().Unix()
		innerPool.SwapFee = float64(fees[i])

		newInnerPools[i] = innerPool
	}

	currentExtra := Extra{}
	if err := json.Unmarshal([]byte(pool.Extra), &currentExtra); err != nil {
		return pool, err
	}
	currentExtra.InnerPools = newInnerPools

	// 2. prepare and set state for PathFinder
	entityPoolByAnchor := make(map[string]*entity.Pool)
	for i, anchor := range anchors {
		entityPoolByAnchor[strings.ToLower(anchor.Hex())] = &newInnerPools[i]
	}
	convertibleTokenAnchors, err := getConvertibleTokensAnchorState(ctx, d.ethrpcClient, d.config.ConverterRegistry)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": d.config.DexID, "err": err}).
			Error("getConvertibleTokensAnchorState failed")

		return pool, err
	}
	currentExtra.InnerPoolByAnchor = entityPoolByAnchor
	currentExtra.AnchorsByConvertibleToken = convertibleTokenAnchors

	// 4. prepare tokens by lp address
	currentExtra.TokensByLpAddress = tokensByAnchor

	extraBytes, err := json.Marshal(currentExtra)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": d.config.DexID, "err": err}).
			Error("marshal extra failed")

		return pool, err
	}
	pool.Extra = string(extraBytes)

	return pool, nil
}

func (d *PoolTracker) getFee(ctx context.Context, pools []entity.Pool) ([]uint64, error) {
	fees := make([]uint32, len(pools))
	getFeeRequest := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i, pool := range pools {
		getFeeRequest.AddCall(&ethrpc.Call{
			ABI:    converterABI,
			Target: pool.Address,
			Method: converterGetFee,
			Params: nil,
		}, []interface{}{&fees[i]})
	}

	_, err := getFeeRequest.TryBlockAndAggregate()
	if err != nil {
		return nil, err
	}

	results := make([]uint64, len(fees))
	for i, fee := range fees {
		results[i] = uint64(fee)
	}
	return results, nil
}

func (d *PoolTracker) getReservesFromRPCNode(ctx context.Context, pools []entity.Pool) ([][]*big.Int, *big.Int, error) {
	reserves := make([][]*big.Int, len(pools))
	getReservesRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	for i, pool := range pools {
		reserves[i] = make([]*big.Int, len(pool.Tokens))
		for j, token := range pool.Tokens {
			getReservesRequest.AddCall(&ethrpc.Call{
				ABI:    converterABI,
				Target: pool.Address,
				Method: converterGetReserve,
				Params: []interface{}{common.HexToAddress(token.Address)},
			}, []interface{}{&reserves[i][j]})
		}
	}

	resp, err := getReservesRequest.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, err
	}

	return reserves, resp.BlockNumber, nil
}
