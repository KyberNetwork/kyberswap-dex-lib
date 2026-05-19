package unipool

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

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	startTime := time.Now()
	dexID := u.config.DexID

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	allPairsLength, err := u.getAllPairsLength(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("getAllPairsLength failed")
		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Warn("getOffset failed")
	}

	batchSize := u.getBatchSize(allPairsLength, u.config.NewPoolLimit, offset)

	pairAddresses, err := u.listPairAddresses(ctx, offset, batchSize)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("listPairAddresses failed")
		return nil, metadataBytes, err
	}

	pools, err := u.initPools(ctx, pairAddresses)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("initPools failed")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + batchSize)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("newMetadata failed")
		return nil, metadataBytes, err
	}

	logger.WithFields(logger.Fields{
		"dex_id":      dexID,
		"pools_len":   len(pools),
		"offset":      offset,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getAllPairsLength(ctx context.Context) (int, error) {
	var allPairsLength *big.Int
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    uniPoolFactoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodGetAllPairsLength,
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
	var m PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &m); err != nil {
		return 0, err
	}
	return m.Offset, nil
}

func (u *PoolsListUpdater) listPairAddresses(ctx context.Context, offset, batchSize int) ([]common.Address, error) {
	results := make([]common.Address, batchSize)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < batchSize; i++ {
		req.AddCall(&ethrpc.Call{
			ABI:    uniPoolFactoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryMethodGetPairAtIndex,
			Params: []any{big.NewInt(int64(offset + i))},
		}, []any{&results[i]})
	}
	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}

	pairs := make([]common.Address, 0, batchSize)
	for i, ok := range resp.Result {
		if !ok {
			continue
		}
		pairs = append(pairs, results[i])
	}
	return pairs, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, pairAddresses []common.Address) ([]entity.Pool, error) {
	if len(pairAddresses) == 0 {
		return nil, nil
	}

	tokenResults := make([]tokensABI, len(pairAddresses))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, addr := range pairAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    uniPoolPairABI,
			Target: addr.Hex(),
			Method: pairMethodGetTokens,
		}, []any{&tokenResults[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	staticExtra := StaticExtra{FactoryAddress: u.config.FactoryAddress}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return nil, err
	}

	extra := zeroExtra()
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(pairAddresses))
	for i, pairAddr := range pairAddresses {
		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(pairAddr[:]),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(tokenResults[i].Token0[:]), Swappable: true},
				{Address: hexutil.Encode(tokenResults[i].Token1[:]), Swappable: true},
			},
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		})
	}
	return pools, nil
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	return json.Marshal(PoolsListUpdaterMetadata{Offset: newOffset})
}

func (u *PoolsListUpdater) getBatchSize(length, limit, offset int) int {
	if offset >= length {
		return 0
	}
	if offset+limit >= length {
		return length - offset
	}
	return limit
}
