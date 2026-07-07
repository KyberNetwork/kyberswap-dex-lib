package v2

import (
	"context"
	"strings"
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
	var (
		dexID     = u.config.DexID
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Warn("getOffset failed")
	}

	markets, err := u.getAllMarkets(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID}).
			Error("getAllMarkets failed")

		return nil, metadataBytes, err
	}

	if offset >= len(markets) {
		return []entity.Pool{}, metadataBytes, nil
	}

	pools, err := u.initPools(ctx, markets[offset:])
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initPools failed")

		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + len(markets))
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
				"new_pools":   len(pools),
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getAllMarkets(ctx context.Context) ([]common.Address, error) {
	markets := make([]common.Address, 0)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    comptrollerABI,
		Target: u.config.Comptroller,
		Method: comptrollerMethodGetAllMarkets,
		Params: nil,
	}, []any{&markets})

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	return markets, nil
}

func (u *PoolsListUpdater) getUnderlyingTokens(ctx context.Context, markets []common.Address) ([]common.Address, error) {
	underlyingTokens := make([]common.Address, len(markets))

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i, market := range markets {
		req.AddCall(&ethrpc.Call{
			ABI:    cTokenABI,
			Target: market.Hex(),
			Method: cTokenMethodUnderlying,
			Params: nil,
		}, []any{&underlyingTokens[i]})
	}

	if _, err := req.TryAggregate(); err != nil {
		return nil, err
	}

	return underlyingTokens, nil
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

func (u *PoolsListUpdater) initPools(ctx context.Context, markets []common.Address) ([]entity.Pool, error) {
	underlyingTokens, err := u.getUnderlyingTokens(ctx, markets)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(markets))

	for i, market := range markets {
		cTokenAddr := hexutil.Encode(market[:])

		var underlyingTokenAddr string
		if underlyingTokens[i] == valueobject.AddrZero {
			underlyingTokenAddr = strings.ToLower(valueobject.WrappedNativeMap[u.config.ChainID])
		} else {
			underlyingTokenAddr = hexutil.Encode(underlyingTokens[i][:])
		}

		cToken := &entity.PoolToken{
			Address:   cTokenAddr,
			Swappable: true,
		}

		underlyingToken := &entity.PoolToken{
			Address:   underlyingTokenAddr,
			Swappable: true,
		}

		var newPool = entity.Pool{
			Address:   cTokenAddr,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens:    []*entity.PoolToken{cToken, underlyingToken},
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadata := PoolsListUpdaterMetadata{
		Offset: newOffset,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}
