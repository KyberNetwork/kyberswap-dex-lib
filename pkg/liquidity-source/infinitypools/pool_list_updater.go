package infinitypools

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
)

type Config struct {
	DexID          string         `json:"dexId"`
	FactoryAddress common.Address `json:"factoryAddress"`
	QuoterAddress  common.Address `json:"quoterAddress"`
}

type PoolListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

type PoolListUpdaterMetadata struct {
	IsInited bool `json:"offset"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolListUpdater {
	return &PoolListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}

}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata PoolListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}

		if metadata.IsInited {
			return nil, metadataBytes, nil
		}
	}

	poolAddresses := []string{
		"0xc3a51f01bc43b1a41b1a1ccaa64c0578cf40ba1f",
		"0x2175a80b99ff2e945ccce92fd0365f0cb5c5e98d",
	}
	poolEntities := make([]entity.Pool, 0, len(poolAddresses))

	for _, poolAddress := range poolAddresses {
		pool, err := u.getPoolEntity(ctx, poolAddress)
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": poolAddress,
				"err":         err,
			}).Errorf("[%s] failed to get pool entity", DexType)
			return nil, metadataBytes, err
		}

		poolEntities = append(poolEntities, pool)
	}

	metadata.IsInited = true
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		logger.WithFields(logger.Fields{
			"err": err,
		}).Errorf("[%s] failed to marshal metadata", DexType)
		return nil, metadataBytes, err
	}

	return poolEntities, metadataBytes, nil
}

func (u *PoolListUpdater) getPoolEntity(ctx context.Context, poolAddress string) (entity.Pool, error) {
	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	var poolInfoResponse [3]any

	req.AddCall(&ethrpc.Call{
		ABI:    infinityPoolABI,
		Target: poolAddress,
		Method: "getPoolInfo",
		Params: nil,
	}, []interface{}{&poolInfoResponse})

	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": poolAddress,
			"err":         err,
		}).Errorf("[%s] failed to get pool info", DexType)
		return entity.Pool{}, err
	}

	token0 := poolInfoResponse[0].(common.Address)
	token1 := poolInfoResponse[1].(common.Address)
	splits := poolInfoResponse[2].(*big.Int)

	var balanceToken0 *big.Int
	var balanceToken1 *big.Int

	req = u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: token0.String(),
		Method: "balanceOf",
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&balanceToken0})

	req.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: token1.String(),
		Method: "balanceOf",
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&balanceToken1})

	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": poolAddress,
			"err":         err,
		}).Errorf("[%s] failed to get pool balance", DexType)
		return entity.Pool{}, err
	}

	extra := Extra{
		Splits:         splits,
		FactoryAddress: u.cfg.FactoryAddress,
		QuoterAddress:  u.cfg.QuoterAddress,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": poolAddress,
			"err":         err,
		}).Errorf("[%s] failed to marshal extra", DexType)
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:   poolAddress,
		Exchange:  u.cfg.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{balanceToken0.String(), balanceToken1.String()},
		Tokens: []*entity.PoolToken{
			{Address: strings.ToLower(token0.String()), Swappable: true},
			{Address: strings.ToLower(token1.String()), Swappable: true},
		},
		Extra: string(extraBytes),
	}, nil
}
