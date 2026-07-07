package poe

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

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
	logger.Infof("started getting new pools")

	poolsLength, err := u.getPoolsLength(ctx)
	if err != nil {
		logger.Errorf("getPoolsLength failed: %v", err)
		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.Warnf("getOffset failed: %v", err)
	}

	batchSize := u.getBatchSize(poolsLength, u.config.NewPoolLimit, offset)
	if batchSize == 0 {
		return nil, metadataBytes, nil
	}

	addresses, err := u.listPoolAddresses(ctx, offset, batchSize)
	if err != nil {
		logger.Errorf("listPoolAddresses failed: %v", err)
		return nil, metadataBytes, err
	}

	pools, err := u.initPools(ctx, addresses)
	if err != nil {
		logger.Errorf("initPools failed: %v", err)
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := json.Marshal(PoolsListUpdaterMetadata{Offset: offset + batchSize})
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.Infof("finished getting %d new pools", len(pools))

	return pools, newMetadataBytes, nil
}

// getPoolsLength gets the number of pools registered in the factory.
func (u *PoolsListUpdater) getPoolsLength(ctx context.Context) (int, error) {
	var poolsLength *big.Int

	req := u.ethrpcClient.R().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryMethodGetPoolsLength,
		}, []any{&poolsLength})

	if _, err := req.Call(); err != nil {
		return 0, err
	}

	return int(poolsLength.Int64()), nil
}

// getOffset gets the index of the last pool that was fetched.
func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}

	return metadata.Offset, nil
}

// getBatchSize caps the number of pools fetched in one run to config.NewPoolLimit.
func (u *PoolsListUpdater) getBatchSize(length, limit, offset int) int {
	if offset >= length {
		return 0
	}

	if limit <= 0 || offset+limit > length {
		return length - offset
	}

	return limit
}

// listPoolAddresses lists pool addresses from the factory starting at offset.
func (u *PoolsListUpdater) listPoolAddresses(ctx context.Context, offset, batchSize int) ([]common.Address, error) {
	addresses := make([]common.Address, batchSize)

	req := u.ethrpcClient.R().SetContext(ctx)
	for i := range batchSize {
		req.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryMethodGetPoolAt,
			Params: []any{big.NewInt(int64(offset + i))},
		}, []any{&addresses[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	return addresses, nil
}

// initPools fetches each pool's token order via getTokens() (order is fixed
// at creation and must be read, never assumed) and builds entity.Pool.
func (u *PoolsListUpdater) initPools(ctx context.Context, addresses []common.Address) ([]entity.Pool, error) {
	infos := make([]struct {
		TokenX common.Address
		TokenY common.Address
	}, len(addresses))

	req := u.ethrpcClient.R().SetContext(ctx)
	for i, addr := range addresses {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: addr.Hex(),
			Method: poolMethodGetTokens,
		}, []any{&infos[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(addresses))
	for i, addr := range addresses {
		info := infos[i]

		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(addr[:]),
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(info.TokenX[:]), Swappable: true},
				{Address: hexutil.Encode(info.TokenY[:]), Swappable: true},
			},
			Extra: "{}",
		})
	}

	return pools, nil
}
