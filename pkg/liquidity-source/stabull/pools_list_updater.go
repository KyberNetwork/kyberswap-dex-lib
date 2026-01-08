package stabull

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
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

	allPairsLength, err := d.getAllPairsLength(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID}).
			Error("getAllPairsLength failed")

		return nil, metadataBytes, err
	}

	offset, err := d.getOffset(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Warn("getOffset failed")
	}

	batchSize := d.getBatchSize(allPairsLength, d.config.NewPoolLimit, offset)

	pairAddresses, err := d.listPairAddresses(ctx, offset, batchSize)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("listPairAddresses failed")

		return nil, metadataBytes, err
	}

	pools, err := d.initPools(ctx, pairAddresses)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initPools failed")

		return nil, metadataBytes, err
	}

	newMetadataBytes, err := d.newMetadata(offset + batchSize)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("newMetadata failed")

		return nil, metadataBytes, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      dexID,
				"pools_len":   len(pools),
				"offset":      offset,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

// getAllPairsLength gets number of pools from the factory contract
// TODO: Stabull factory doesn't have curvesLength method - needs event-based discovery
// Pools should be discovered via NewCurve events instead of indexed enumeration
func (d *PoolsListUpdater) getAllPairsLength(ctx context.Context) (int, error) {
	// Stabull factory uses event-based pool discovery (NewCurve events)
	// There is no curvesLength() method in the factory contract
	// Return 0 for now - proper implementation should use GetNewPoolsType to discover via events
	logger.WithFields(logger.Fields{
		"dex": DexType,
	}).Warn("getAllPairsLength not supported - Stabull uses event-based pool discovery")

	return 0, nil
}

// getOffset gets index of the last pool that is fetched
func (d *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}

	return metadata.Offset, nil
}

// listPairAddresses lists addresses of pools from offset
// TODO: Stabull factory uses bytes32 IDs, not indexed enumeration
// Proper implementation should discover pools via NewCurve events
func (d *PoolsListUpdater) listPairAddresses(ctx context.Context, offset int, batchSize int) ([]common.Address, error) {
	// The curves() method takes bytes32 ID, not uint256 index
	// Pool discovery should be event-based using NewCurve events
	logger.WithFields(logger.Fields{
		"dex": DexType,
	}).Warn("listPairAddresses not supported - Stabull uses event-based pool discovery")

	return []common.Address{}, nil
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
		token0Address   common.Address
		token1Address   common.Address
		token0Decimals  uint8
		liquidityResult struct {
			Total      *big.Int   `json:"total_"`
			Individual []*big.Int `json:"individual_"`
		}
		curveResult struct {
			Alpha   *big.Int `json:"alpha_"`
			Beta    *big.Int `json:"beta_"`
			Delta   *big.Int `json:"delta_"`
			Epsilon *big.Int `json:"epsilon_"`
			Lambda  *big.Int `json:"lambda_"`
		}
	)

	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	// Fetch token0 address (base token) using numeraires(0)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodNumeraires,
		Params: []interface{}{big.NewInt(0)},
	}, []interface{}{&token0Address})

	// Fetch token1 address (USDC) using numeraires(1)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodNumeraires,
		Params: []interface{}{big.NewInt(1)},
	}, []interface{}{&token1Address})

	// Fetch reserves using liquidity() method
	// liquidity() returns (uint256 total_, uint256[] individual_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodLiquidity,
		Params: []interface{}{},
	}, []interface{}{&liquidityResult.Total, &liquidityResult.Individual})

	// Fetch curve parameters (alpha, beta, delta, epsilon, lambda)
	// viewCurve() returns (uint256 alpha_, uint256 beta_, uint256 delta_, uint256 epsilon_, uint256 lambda_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodViewCurve,
		Params: []interface{}{},
	}, []interface{}{&curveResult.Alpha, &curveResult.Beta, &curveResult.Delta, &curveResult.Epsilon, &curveResult.Lambda})

	// Fetch token0 decimals using ERC20 decimals() method
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: token0Address.Hex(),
		Method: abi.Erc20DecimalsMethod,
		Params: []interface{}{},
	}, []interface{}{&token0Decimals})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, err
	}

	// Build curve parameters (convert to strings for JSON)
	curveParams := CurveParameters{
		Alpha:   curveResult.Alpha.String(),
		Beta:    curveResult.Beta.String(),
		Delta:   curveResult.Delta.String(),
		Epsilon: curveResult.Epsilon.String(),
		Lambda:  curveResult.Lambda.String(),
	}

	// Token metadata: token1 is always USDC (6 decimals)
	// Token0 decimals fetched via ERC20 call
	// Fallback to 18 decimals if fetch fails
	if token0Decimals == 0 {
		token0Decimals = 18
	}

	extra := Extra{
		CurveParams:     curveParams,
		BaseOracleRate:  "", // Oracle rates built into viewOriginSwap
		QuoteOracleRate: "", // Not needed for simulation
		OracleRate:      "",
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}

	// Convert reserves to strings
	reserves := make([]string, len(liquidityResult.Individual))
	for i, reserve := range liquidityResult.Individual {
		reserves[i] = reserve.String()
	}

	// Token metadata
	// Stabull pools always have USDC as token1 (quote currency)
	// Token0 is the fiat-backed stablecoin (e.g., AUDS, NZDS, etc.)
	// Symbol is kept generic as ERC20 symbol() method is not always reliable
	tokens := []*entity.PoolToken{
		{
			Address:   strings.ToLower(token0Address.Hex()),
			Symbol:    "TOKEN0",       // Generic symbol (actual symbol can vary: AUDS, NZDS, etc.)
			Decimals:  token0Decimals, // Fetched via ERC20 decimals() call
			Swappable: true,
		},
		{
			Address:   strings.ToLower(token1Address.Hex()),
			Symbol:    "USDC", // All Stabull pools have USDC as quote token
			Decimals:  6,      // USDC always has 6 decimals
			Swappable: true,
		},
	}

	return &entity.Pool{
		Address:   strings.ToLower(poolAddress),
		Exchange:  d.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  reserves,
		Tokens:    tokens,
		Extra:     string(extraBytes),
	}, nil
}
