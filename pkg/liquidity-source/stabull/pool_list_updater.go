package stabull

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type PoolsListUpdaterMetadata struct {
	Offset int `json:"offset"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = d.config.DexID
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	// Stabull uses factory.getCurve(base, quote) to discover pools
	// We query the factory with known token pairs to find all deployed pools
	pairAddresses, err := d.discoverPoolsFromFactory(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("discoverPoolsFromFactory failed")

		return nil, metadataBytes, err
	}

	pools, err := d.initPools(ctx, pairAddresses)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initPools failed")

		return nil, metadataBytes, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      dexID,
				"pools_len":   len(pools),
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, metadataBytes, nil
}

// discoverPoolsFromFactory queries factory NewCurve events to find all deployed pools
func (d *PoolsListUpdater) discoverPoolsFromFactory(ctx context.Context) ([]common.Address, error) {
	logger.WithFields(logger.Fields{
		"dex":     DexType,
		"factory": d.config.FactoryAddress,
		"from":    d.config.FromBlock,
	}).Info("discovering pools from NewCurve events")

	// Query factory logs for NewCurve events
	// Event signature: NewCurve(address indexed caller, bytes32 indexed id, address indexed curve)
	// Topic: 0xe7a19de9e8788cc07c144818f2945144acd6234f790b541aa1010371c8b2a73b
	fromBlock := new(big.Int).SetUint64(d.config.FromBlock)

	// Use eth_getLogs directly via eth client
	query := ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   nil, // latest
		Addresses: []common.Address{common.HexToAddress(d.config.FactoryAddress)},
		Topics: [][]common.Hash{
			{common.HexToHash(newCurveTopic)}, // Event signature
		},
	}

	logs, err := d.ethrpcClient.GetETHClient().FilterLogs(ctx, query)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dex":   DexType,
			"error": err,
		}).Error("failed to query NewCurve events")
		return nil, fmt.Errorf("failed to query NewCurve events: %w", err)
	}

	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"eventsFound": len(logs),
	}).Info("fetched NewCurve events")

	// Extract pool addresses from events
	var poolAddresses []common.Address
	poolSet := make(map[common.Address]struct{}) // Deduplicate

	for _, log := range logs {
		// The third indexed parameter (curve address) is in Topics[3]
		if len(log.Topics) < 4 {
			logger.WithFields(logger.Fields{
				"dex":       DexType,
				"topicsLen": len(log.Topics),
			}).Warn("invalid NewCurve event: not enough topics")
			continue
		}

		// Topics[0] = event signature
		// Topics[1] = caller (indexed)
		// Topics[2] = id (indexed)
		// Topics[3] = curve address (indexed)
		poolAddress := common.BytesToAddress(log.Topics[3].Bytes())

		if poolAddress == (common.Address{}) {
			continue
		}

		// Deduplicate
		if _, exists := poolSet[poolAddress]; !exists {
			poolSet[poolAddress] = struct{}{}
			poolAddresses = append(poolAddresses, poolAddress)
		}
	}

	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"pools_found": len(poolAddresses),
	}).Info("discovered pools from NewCurve events")

	return poolAddresses, nil
}

func (d *PoolsListUpdater) initPools(ctx context.Context, pairAddresses []common.Address) ([]entity.Pool, error) {
	var pools []entity.Pool
	for _, poolAddress := range pairAddresses {
		pool, err := d.getNewPool(ctx, poolAddress.Hex())
		if err != nil {
			logger.WithFields(logger.Fields{
				"dex":         DexType,
				"poolAddress": poolAddress.Hex(),
				"error":       err,
			}).Warn("failed to fetch pool")
			continue
		}
		pools = append(pools, *pool)
	}

	return pools, nil
}

func (d *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadata := PoolsListUpdaterMetadata{
		Offset: newOffset,
	}

	return json.Marshal(metadata)
}

func (d *PoolsListUpdater) getBatchSize(length int, limit int, offset int) int {
	if length <= 0 {
		return 0
	}

	if offset >= length {
		return 0
	}

	if limit <= 0 {
		limit = length
	}

	batchSize := min(length-offset, limit)

	return batchSize
}

func (d *PoolsListUpdater) getNewPool(ctx context.Context, poolAddress string) (*entity.Pool, error) {
	var (
		token0Address       common.Address
		token1Address       common.Address
		token0Decimals      uint8
		token1Decimals      uint8
		assimilator0Address common.Address
		assimilator1Address common.Address
		oracle0Address      common.Address
		oracle1Address      common.Address
	)

	// Batch 1: Fetch token addresses from pool
	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodNumeraires,
		Params: []interface{}{big.NewInt(0)},
	}, []interface{}{&token0Address})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodNumeraires,
		Params: []interface{}{big.NewInt(1)},
	}, []interface{}{&token1Address})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token addresses: %w", err)
	}

	// Batch 2: Fetch token decimals and assimilator addresses
	rpcRequest2 := d.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest2.AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: token0Address.Hex(),
		Method: abi.Erc20DecimalsMethod,
		Params: []interface{}{},
	}, []interface{}{&token0Decimals})

	rpcRequest2.AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: token1Address.Hex(),
		Method: abi.Erc20DecimalsMethod,
		Params: []interface{}{},
	}, []interface{}{&token1Decimals})

	rpcRequest2.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodAssimilator,
		Params: []interface{}{token0Address},
	}, []interface{}{&assimilator0Address})

	rpcRequest2.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodAssimilator,
		Params: []interface{}{token1Address},
	}, []interface{}{&assimilator1Address})

	_, err = rpcRequest2.Aggregate()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token decimals and assimilators: %w", err)
	}

	// Batch 3: Fetch oracle addresses from assimilators
	rpcRequest3 := d.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest3.AddCall(&ethrpc.Call{
		ABI:    assimilatorABI,
		Target: assimilator0Address.Hex(),
		Method: assimilatorMethodOracle,
		Params: []interface{}{},
	}, []interface{}{&oracle0Address})

	rpcRequest3.AddCall(&ethrpc.Call{
		ABI:    assimilatorABI,
		Target: assimilator1Address.Hex(),
		Method: assimilatorMethodOracle,
		Params: []interface{}{},
	}, []interface{}{&oracle1Address})

	_, err = rpcRequest3.Aggregate()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch oracle addresses: %w", err)
	}

	// Fallback to default decimals if needed
	if token0Decimals == 0 {
		token0Decimals = 18
	}
	if token1Decimals == 0 {
		token1Decimals = 6 // USDC default
	}

	// Build Extra with oracle addresses
	// Note: Reserves and curve params will be fetched by pool_tracker
	extra := Extra{
		BaseOracleAddress:  strings.ToLower(oracle0Address.Hex()),
		QuoteOracleAddress: strings.ToLower(oracle1Address.Hex()),
		CurveParams:        CurveParameters{}, // Empty, will be populated by pool_tracker
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal extra: %w", err)
	}

	// StaticExtra with assimilator addresses (won't change)
	staticExtra := map[string]interface{}{
		"baseAssimilator":  strings.ToLower(assimilator0Address.Hex()),
		"quoteAssimilator": strings.ToLower(assimilator1Address.Hex()),
	}

	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal static extra: %w", err)
	}

	// Token metadata
	tokens := []*entity.PoolToken{
		{
			Address:   strings.ToLower(token0Address.Hex()),
			Symbol:    "TOKEN0",
			Decimals:  token0Decimals,
			Swappable: true,
		},
		{
			Address:   strings.ToLower(token1Address.Hex()),
			Symbol:    "TOKEN1",
			Decimals:  token1Decimals,
			Swappable: true,
		},
	}

	// Return pool WITHOUT reserves (pool_tracker will fetch them)
	// Reserves are set to ["0", "0"] as placeholders
	return &entity.Pool{
		Address:     strings.ToLower(poolAddress),
		Exchange:    d.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    []string{reserveZero, reserveZero}, // Placeholder, pool_tracker will update
		Tokens:      tokens,
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}, nil
}
