package poolmanager_test

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolfactory"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

const configFile = "../../config/files/dev/polygon.yaml"

func newMockPointerSwapPoolManager(configFile string) (*poolmanager.PointerSwapPoolManager, error) {
	ctx := context.TODO()
	configLoader, err := config.NewConfigLoader(configFile)
	if err != nil {
		return nil, err
	}

	cfg, err := configLoader.Get()
	if err != nil {
		return nil, err
	}

	if err = cfg.Validate(); err != nil {
		logger.Errorf(ctx, "failed to validate config, err: %v", err)
		panic(err)
	}

	routerRedisClient, err := redis.New(&cfg.Redis)
	if err != nil {
		return nil, err
	}

	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		logger.Errorf(ctx, "fail to init redis client to pool service")
		return nil, err
	}
	poolRepository, _ := pool.NewRedisRepository(poolRedisClient.Client, nil, cfg.Repository.Pool)
	poolRankRepository := poolrank.NewRedisRepository(routerRedisClient.Client, cfg.Repository.PoolRank)
	poolFactory := poolfactory.NewPoolFactory(cfg.UseCase.PoolFactory, nil, nil)
	return poolmanager.NewPointerSwapPoolManager(ctx, poolRepository, poolFactory, poolRankRepository, cfg.UseCase.PoolManager, nil)
}

func newMockPoolManager(configFile string) (*poolmanager.PoolManager, error) {
	ctx := context.TODO()
	configLoader, err := config.NewConfigLoader(configFile)
	if err != nil {
		return nil, err
	}

	cfg, err := configLoader.Get()
	if err != nil {
		return nil, err
	}

	if err = cfg.Validate(); err != nil {
		logger.Errorf(ctx, "failed to validate config, err: %v", err)
		panic(err)
	}

	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		logger.Errorf(ctx, "fail to init redis client to pool service")
		return nil, err
	}
	poolRepository, _ := pool.NewRedisRepository(poolRedisClient.Client, nil, cfg.Repository.Pool)
	poolFactory := poolfactory.NewPoolFactory(cfg.UseCase.PoolFactory, nil, nil)
	return poolmanager.NewPoolManager(poolRepository, poolFactory, cfg.UseCase.PoolManager), nil
}

func TestProfilePointerSwapPoolManager(t *testing.T) {
	t.Skip()
	for i := 0; i < 10; i++ {
		_, err := newMockPointerSwapPoolManager(configFile)
		require.Nil(t, err)
	}
}

func comparePoolManager(
	pointerSwapPoolManager *poolmanager.PointerSwapPoolManager,
	poolManager *poolmanager.PoolManager,
	addresses, dex []string,
) error {
	s1, err := pointerSwapPoolManager.GetStateByPoolAddresses(context.Background(), addresses, dex, common.Hash{})
	if err != nil {
		return errors.Wrap(err, "pointerSwapPoolManager")
	}
	s2, err := poolManager.GetStateByPoolAddresses(context.Background(), addresses, dex, common.Hash{})
	if err != nil {
		return errors.Wrap(err, "poolManager")
	}
	if len(s1.Pools) != len(s2.Pools) {
		panic(err)
	}
	return nil
}

func listAddresses(configFile, tokenIn, tokenOut string) ([]string, []string, error) {
	ctx := context.TODO()
	configLoader, err := config.NewConfigLoader(configFile)
	if err != nil {
		return nil, nil, err
	}

	cfg, err := configLoader.Get()
	if err != nil {
		return nil, nil, err
	}

	if err = cfg.Validate(); err != nil {
		logger.Errorf(ctx, "failed to validate config, err: %v", err)
		panic(err)
	}

	routerRedisClient, err := redis.New(&cfg.Redis)
	if err != nil {
		return nil, nil, err
	}
	poolRankRepository := poolrank.NewRedisRepository(routerRedisClient.Client, cfg.Repository.PoolRank)
	aggregatorConfig := cfg.UseCase.GetRoute.Aggregator
	poolAddress, err := poolRankRepository.FindBestPoolIDs(
		context.Background(), tokenIn, tokenOut,
		aggregatorConfig.GetBestPoolsOptions,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(poolAddress) == 0 {
		fmt.Printf("cannot find best pools tokenIn: %v tokenOut: %v\n", tokenIn, tokenOut)
	}
	return poolAddress, cfg.UseCase.GetRoute.AvailableSources, nil
}

func TestComparePoolManager(t *testing.T) {
	t.Skip()
	f, err := os.Open("./token_pairs.csv")
	require.Nil(t, err)
	defer f.Close()

	// read csv values using csv.Reader
	csvReader := csv.NewReader(f)
	data, err := csvReader.ReadAll()
	require.Nil(t, err)

	pointerSwapPoolManager, err := newMockPointerSwapPoolManager(configFile)
	require.Nil(t, err)
	poolManager, err := newMockPoolManager(configFile)
	require.Nil(t, err)

	for i := 1; i < len(data); i++ {
		poolAddress, dex, err := listAddresses(configFile, data[i][0], data[i][1])
		require.Nil(t, err)
		require.Nil(t, comparePoolManager(pointerSwapPoolManager, poolManager, poolAddress, dex))
	}
}
