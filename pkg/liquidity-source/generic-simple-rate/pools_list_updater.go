package generic_simple_rate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	if d.config.ABIJsonString != "" {
		ABI, err := abi.JSON(bytes.NewReader([]byte(d.config.ABIJsonString)))
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to parse ABI")
			return nil, nil, err
		}
		abiMap[d.config.DexID] = ABI
	}

	pools, err := d.initPools()
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to initPool")
		return nil, nil, err
	}
	logger.WithFields(logger.Fields{"pool": pools}).Info("finish fetching pools")

	return pools, nil, nil
}

func (d *PoolsListUpdater) initPools() ([]entity.Pool, error) {
	if d.config.Pools == "" {
		logger.Errorf("misconfigured pool")
		return nil, errors.New("misconfigured pool")
	}

	var poolItems []PoolItem
	if err := json.Unmarshal([]byte(d.config.Pools), &poolItems); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal pool")
		return nil, err
	}

	pools, err := d.processBatch(poolItems)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to processBatch")
		return nil, err
	}
	d.hasInitialized = true

	return pools, nil
}

func (d *PoolsListUpdater) processBatch(poolItems []PoolItem) ([]entity.Pool, error) {
	var pools = make([]entity.Pool, 0, len(poolItems))

	for _, pool := range poolItems {
		var err error
		var poolEntity entity.Pool

		poolEntity, err = d.getNewPool(&pool)

		if err != nil {
			return nil, err
		}

		pools = append(pools, poolEntity)
	}

	return pools, nil
}

func (d *PoolsListUpdater) getNewPool(pool *PoolItem) (entity.Pool, error) {
	var tokens = make([]*entity.PoolToken, 0, len(pool.Tokens))
	var reserves = make(entity.PoolReserves, 0, len(pool.Tokens))

	for _, token := range pool.Tokens {
		tokenEntity := entity.PoolToken{
			Address:   strings.ToLower(token.Address),
			Name:      token.Name,
			Symbol:    token.Symbol,
			Decimals:  token.Decimals,
			Weight:    defaultTokenWeight,
			Swappable: true,
		}
		tokens = append(tokens, &tokenEntity)
		reserves = append(reserves, defaultReserves)
	}

	var (
		paused bool
		rate   *big.Int
	)

	req := d.ethrpcClient.R()
	if d.config.PausedMethod != "" {
		req.AddCall(&ethrpc.Call{
			ABI:    getABI(pool.Exchange),
			Target: pool.Address,
			Method: d.config.PausedMethod,
			Params: []interface{}{},
		}, []interface{}{&paused})
	}

	if d.config.IsRateUpdatable {
		req.AddCall(&ethrpc.Call{
			ABI:    abiMap[pool.Exchange],
			Target: pool.Address,
			Method: d.config.RateMethod,
			Params: []interface{}{},
		}, []interface{}{&rate})
	} else {
		rate = d.config.DefaultRate
	}

	if len(req.Calls) > 0 {
		_, err := req.Aggregate()
		if err != nil {
			return entity.Pool{}, err
		}
	}

	defaultGas := DefaultGas
	if d.config.DefaultGas != nil {
		defaultGas = d.config.DefaultGas.Int64()
	}

	poolExtraBytes, err := json.Marshal(PoolExtra{
		Rate:       uint256.MustFromBig(rate),
		RateUnit:   uint256.MustFromBig(d.config.RateUnit),
		Paused:     paused,
		DefaultGas: defaultGas,
	})
	if err != nil {
		return entity.Pool{}, err
	}

	poolEntity := entity.Pool{
		Address:   pool.Address,
		Exchange:  d.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  reserves,
		Tokens:    tokens,
		Extra:     string(poolExtraBytes),
	}

	return poolEntity, nil
}

func getABI(exchange string) abi.ABI {
	if ABI, ok := abiMap[exchange]; ok {
		return ABI
	}
	return rateABI
}
