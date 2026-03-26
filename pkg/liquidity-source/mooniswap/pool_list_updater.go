package mooniswap

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{"dex_id": u.config.DexId}).Info("started getting new pools")

	allPoolAddresses, err := u.getAllPools(ctx)
	if err != nil {
		return nil, metadataBytes, err
	}

	var metadata PoolsListUpdaterMetadata
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &metadata)
	}

	if metadata.TotalPools == len(allPoolAddresses) {
		return nil, metadataBytes, nil
	}

	pools, err := u.initPools(ctx, allPoolAddresses)
	if err != nil {
		return nil, metadataBytes, err
	}

	newMetadata, _ := json.Marshal(PoolsListUpdaterMetadata{
		TotalPools: len(allPoolAddresses),
	})

	logger.WithFields(logger.Fields{"pools_len": len(pools)}).Info("finished getting new pools")

	return pools, newMetadata, nil
}

func (u *PoolsListUpdater) getAllPools(ctx context.Context) ([]common.Address, error) {
	var poolAddresses []common.Address
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodGetAllPools,
	}, []any{&poolAddresses})

	if _, err := req.Call(); err != nil {
		return nil, err
	}

	return poolAddresses, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, addresses []common.Address) ([]entity.Pool, error) {
	token0List := make([]common.Address, len(addresses))
	token1List := make([]common.Address, len(addresses))

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, addr := range addresses {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: addr.Hex(),
			Method: poolMethodToken0,
		}, []any{&token0List[i]})
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: addr.Hex(),
			Method: poolMethodToken1,
		}, []any{&token1List[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(addresses))
	for i, addr := range addresses {
		token0 := hexutil.Encode(token0List[i][:])
		token1 := hexutil.Encode(token1List[i][:])

		if valueobject.IsNative(token0) || valueobject.IsNative(token1) {
			continue
		}

		staticExtraBytes, _ := json.Marshal(StaticExtra{
			IsNativeToken0: valueobject.IsZero(token0),
			IsNativeToken1: valueobject.IsZero(token1),
		})

		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(addr[:]),
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: valueobject.ZeroToWrappedLower(token0, u.config.ChainId), Swappable: true},
				{Address: valueobject.ZeroToWrappedLower(token1, u.config.ChainId), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}

	return pools, nil
}
