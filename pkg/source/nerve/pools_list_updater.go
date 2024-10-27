package nerve

import (
	"context"
	"errors"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/bytedance/sonic"

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
		config:         cfg,
		ethrpcClient:   ethrpcClient,
		hasInitialized: false,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{
		"liquiditySource": DexTypeNerve,
		"kind":            "getNewPools",
	})
	if d.hasInitialized {
		log.Infof("initialized. Ignore making new pools ")
		return nil, nil, nil
	}

	if d.config.PoolPath == "" {
		log.Errorf("config pool path empty")
		return nil, nil, errors.New("config pool path empty")
	}

	byteValue, ok := BytesByPath[d.config.PoolPath]
	if !ok {
		log.Errorf("couldn't parse bytesByPath from poolPath")
		return nil, nil, errors.New("misconfigured pools")
	}

	var poolsItem []PoolItem
	if err := sonic.Unmarshal(byteValue, &poolsItem); err != nil {
		log.Errorf("failed to parse pools: err %v", err)
		return nil, nil, err
	}
	log.Infof("got %v pools from file: %s", len(poolsItem), d.config.PoolPath)

	var pools []entity.Pool
	for i := range poolsItem {
		var pool = poolsItem[i]
		var swapStorage SwapStorage

		rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    swapABI,
			Target: pool.ID,
			Method: methodGetSwapStorage,
			Params: nil,
		}, []interface{}{&swapStorage})

		if _, err := rpcRequest.Call(); err != nil {
			log.Errorf("failed to get swap storage, err: %v", err)
			return nil, nil, err
		}

		var tokens []*entity.PoolToken
		var reserves entity.PoolReserves
		var staticExtra PoolStaticExtra

		for _, item := range pool.Tokens {
			tokenModel := entity.PoolToken{
				Address:   item.Address,
				Weight:    1,
				Swappable: true,
			}
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, item.Precision)
			tokens = append(tokens, &tokenModel)
			reserves = append(reserves, reserveZero)
		}

		staticExtraBytes, err := sonic.Marshal(staticExtra)
		if err != nil {
			log.Errorf("error when marshal staticExtra: %v", err)
			return nil, nil, err
		}
		var newPool = entity.Pool{
			Address:     pool.ID,
			ReserveUsd:  0,
			SwapFee:     0,
			Exchange:    d.config.DexID,
			Type:        DexTypeNerve,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			StaticExtra: string(staticExtraBytes),
			Tokens:      tokens,
		}
		pools = append(pools, newPool)
	}
	d.hasInitialized = true
	return pools, nil, nil
}
