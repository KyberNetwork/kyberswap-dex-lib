package ondo_usdy

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client

		hasInitialized bool
	}
)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	pools, err := u.initPools(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to initPool")
		return nil, nil, err
	}

	logger.WithFields(logger.Fields{
		"dex_id": u.config.DexID,
	}).Info("finish fetching pools")

	return pools, nil, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context) ([]entity.Pool, error) {
	byteData, ok := bytesByPath[u.config.PoolPath]
	if !ok {
		logger.Errorf("misconfigured poolPath")
		return nil, errors.New("misconfigured poolPath")
	}

	var poolItems []PoolItem
	if err := json.Unmarshal(byteData, &poolItems); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal poolData")
		return nil, err
	}

	pools, err := u.processBatch(ctx, poolItems)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to processBatch")
		return nil, err
	}
	u.hasInitialized = true

	return pools, nil
}

func (u *PoolsListUpdater) processBatch(ctx context.Context, poolItems []PoolItem) ([]entity.Pool, error) {
	var pools = make([]entity.Pool, 0, len(poolItems))
	var rwaDynamicOracleAddresses = make([]string, 0, len(poolItems))

	for _, pool := range poolItems {
		var tokens = make([]*entity.PoolToken, 0, len(pool.Tokens))
		var reserves = make(entity.PoolReserves, 0, len(pool.Tokens))

		for _, token := range pool.Tokens {
			tokenEntity := entity.PoolToken{
				Address:   strings.ToLower(token.Address),
				Name:      token.Name,
				Symbol:    token.Symbol,
				Decimals:  token.Decimals,
				Weight:    18,
				Swappable: true,
			}
			tokens = append(tokens, &tokenEntity)
			reserves = append(reserves, defaultReserves)
		}

		poolEntity := entity.Pool{
			Address:   pool.ID,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  reserves,
			Tokens:    tokens,
		}

		rwaDynamicOracleAddresses = append(rwaDynamicOracleAddresses, pool.RWADynamicOracleAddress)
		pools = append(pools, poolEntity)
	}

	poolExtras, blockNumber, err := getExtra(
		ctx, u.config, u.ethrpcClient, pools, rwaDynamicOracleAddresses,
	)
	if err != nil {
		return nil, err
	}

	for i := range pools {
		extraBytes, err := json.Marshal(poolExtras[i])
		if err != nil {
			return nil, err
		}
		pools[i].Extra = string(extraBytes)
		pools[i].BlockNumber = blockNumber
	}

	return pools, nil
}

func getExtra(
	ctx context.Context,
	config *Config,
	client *ethrpc.Client,
	pools []entity.Pool,
	rwaDynamicOracleAddress []string,
) ([]PoolExtra, uint64, error) {
	paused := make([]bool, len(pools))
	oraclePriceData := make([]OraclePriceData, len(pools))
	totalShares := make([]*big.Int, len(pools))

	methodGetTotalShares := getMethodTotalShares(config.ChainID)

	calls := client.NewRequest().SetContext(ctx)
	for i := range pools {
		calls.AddCall(&ethrpc.Call{
			ABI:    rUSDYABI,
			Target: pools[i].Address,
			Method: rUSDYMethodPaused,
		}, []interface{}{&paused[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    rUSDYABI,
			Target: pools[i].Address,
			Method: methodGetTotalShares,
		}, []interface{}{&totalShares[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    rwaDynamicOracleABI,
			Target: rwaDynamicOracleAddress[i],
			Method: rwaDynamicOracleMethodGetPriceData,
		}, []interface{}{&oraclePriceData[i]})
	}

	resp, err := calls.Aggregate()
	if err != nil {
		return []PoolExtra{}, 0, err
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	var poolExtras []PoolExtra
	for i := range pools {
		poolExtras = append(poolExtras, PoolExtra{
			Paused:                  paused[i],
			TotalShares:             uint256.MustFromBig(totalShares[i]),
			OraclePrice:             uint256.MustFromBig(oraclePriceData[i].Price),
			PriceTimestamp:          oraclePriceData[i].Timestamp.Uint64(),
			RWADynamicOracleAddress: rwaDynamicOracleAddress[i],
		})
	}

	return poolExtras, resp.BlockNumber.Uint64(), nil
}

func getMethodTotalShares(chainID valueobject.ChainID) string {
	switch chainID {
	case valueobject.ChainIDEthereum:
		return rUSDYMethodTotalShares
	default:
		return rUSDYWMethodGetTotalShares
	}
}
