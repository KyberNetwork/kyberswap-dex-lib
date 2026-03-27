package v2

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
	logger.Info("started getting new pools")

	allPairsLength, err := u.getAllPairsLength(ctx)
	if err != nil {
		logger.Error("getAllPairsLength failed")
		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.Warn("getOffset failed")
	}

	batchSize := u.getBatchSize(allPairsLength, u.config.NewPoolLimit, offset)
	if batchSize == 0 {
		return nil, metadataBytes, nil
	}

	pairAddresses, err := u.listPairAddresses(ctx, offset, batchSize)
	if err != nil {
		return nil, metadataBytes, err
	}

	pools, err := u.initPools(ctx, pairAddresses)
	if err != nil {
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + batchSize)
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.Info("finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getAllPairsLength(ctx context.Context) (int, error) {
	var allPairsLength *big.Int

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.Factory,
		Method: "allPairsLength",
	}, []any{&allPairsLength})
	if _, err := req.Call(); err != nil {
		return 0, err
	}

	return int(allPairsLength.Int64()), nil
}

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

func (u *PoolsListUpdater) listPairAddresses(ctx context.Context, offset int, batchSize int) ([]common.Address, error) {
	addresses := make([]common.Address, batchSize)
	req := u.ethrpcClient.R().SetContext(ctx)
	for i := 0; i < batchSize; i++ {
		req.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.config.Factory,
			Method: "pairs",
			Params: []any{big.NewInt(int64(offset + i))},
		}, []any{&addresses[i]})
	}
	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}

	var pairAddresses []common.Address
	for i, isSuccess := range resp.Result {
		if !isSuccess {
			continue
		}

		pairAddresses = append(pairAddresses, addresses[i])
	}

	return pairAddresses, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, pairAddresses []common.Address) ([]entity.Pool, error) {
	agentTokens, assetTokens, routers, bondings, err := u.getPairsInfo(ctx, pairAddresses)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(pairAddresses))

	for i, pairAddress := range pairAddresses {
		token0 := &entity.PoolToken{Address: hexutil.Encode(agentTokens[i][:]), Swappable: true}
		token1 := &entity.PoolToken{Address: hexutil.Encode(assetTokens[i][:]), Swappable: true}

		staticExtra, err := json.Marshal(&StaticExtra{
			Bonding: bondings[i].Hex(),
			Router:  routers[i].Hex(),
		})
		if err != nil {
			return nil, err
		}

		var newPool = entity.Pool{
			Address:     hexutil.Encode(pairAddress[:]),
			Exchange:    string(u.config.DexId),
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{"0", "0"},
			Tokens:      []*entity.PoolToken{token0, token1},
			StaticExtra: string(staticExtra),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}

func (u *PoolsListUpdater) getPairsInfo(
	ctx context.Context,
	pairAddresses []common.Address,
) ([]common.Address, []common.Address, []common.Address, []common.Address, error) {
	agentTokens := make([]common.Address, len(pairAddresses))
	assetTokens := make([]common.Address, len(pairAddresses))
	routers := make([]common.Address, len(pairAddresses))

	req := u.ethrpcClient.R().SetContext(ctx)
	for i, pairAddress := range pairAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddress.Hex(),
			Method: "tokenA",
		}, []any{&agentTokens[i]}).AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddress.Hex(),
			Method: "tokenB",
		}, []any{&assetTokens[i]}).AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddress.Hex(),
			Method: "router",
		}, []any{&routers[i]})
	}
	_, err := req.Aggregate()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	bondings := make([]common.Address, len(pairAddresses))
	req = u.ethrpcClient.R().SetContext(ctx)
	for i, router := range routers {
		req.AddCall(&ethrpc.Call{
			ABI:    routerABI,
			Target: router.Hex(),
			Method: "bondingV4",
		}, []any{&bondings[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, nil, nil, nil, err
	}

	return agentTokens, assetTokens, routers, bondings, nil
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadataBytes, err := json.Marshal(PoolsListUpdaterMetadata{Offset: newOffset})
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}

func (u *PoolsListUpdater) getBatchSize(length int, limit int, offset int) int {
	if offset == length {
		return 0
	}

	if offset+limit >= length {
		return max(length-offset, 0)
	}

	return limit
}
