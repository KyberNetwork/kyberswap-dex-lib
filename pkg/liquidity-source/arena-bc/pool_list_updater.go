package arenabc

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
		logger       logger.Logger
	}

	PoolsListUpdaterMetadata struct {
		Offset uint64 `json:"offset"`
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
		logger:       logger.WithFields(logger.Fields{"dex_id": cfg.DexId}),
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var startTime = time.Now()
	u.logger.Info("started getting new pools")

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		u.logger.Warn("failed to getOffset")
	}

	tokenIdentifier, err := u.getTokenIdentifier(ctx)
	if err != nil {
		u.logger.Error("failed to getTokenIdentifier")
		return nil, metadataBytes, err
	}

	// TokenManager use 1-based indexing
	if offset >= tokenIdentifier.Uint64() {
		u.logger.Info("no new pools")
		return nil, metadataBytes, nil
	}

	newOffset := min(tokenIdentifier.Uint64(), offset+uint64(u.config.NewPoolLimit))
	pools, err := u.initPools(ctx, offset, newOffset)
	if err != nil {
		u.logger.Error("failed to initPools")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(newOffset)
	if err != nil {
		return nil, metadataBytes, err
	}

	u.logger.WithFields(logger.Fields{
		"new_pools":   len(pools),
		"offset":      offset,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, offset, newOffset uint64) ([]entity.Pool, error) {
	if offset >= newOffset {
		return nil, nil
	}

	size := int(newOffset - offset)
	tokenParametersList := make([]TokenParametersResult, size)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := offset; i < newOffset; i++ {
		req.AddCall(&ethrpc.Call{
			ABI:    tokenManagerABI,
			Target: u.config.TokenManager,
			Method: tokenManagerMethodTokenParams,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&tokenParametersList[i-offset]})
	}

	_, err := req.Aggregate()
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, size)

	for i, v := range tokenParametersList {
		if v.LpDeployed {
			logger.Infof("pool already deployed: token_id=%d, pair_address=%s", uint64(i)+offset, v.PairAddress.String())
			continue
		}

		wrappedNativeToken := &entity.PoolToken{
			Address:   strings.ToLower(valueobject.WrappedNativeMap[u.config.ChainId]),
			Swappable: true,
		}

		cToken := &entity.PoolToken{
			Address:   strings.ToLower(v.TokenContractAddress.Hex()),
			Swappable: true,
		}

		staticExtraBytes, _ := json.Marshal(StaticExtra{
			ChainId:      u.config.ChainId,
			TokenManager: u.config.TokenManager,
			TokenId:      big.NewInt(int64(i) + int64(offset)),
		})

		pool := entity.Pool{
			// Use token contract address instead of pool address to avoid conflict with arenadex-v2 pools
			// when pools migrated to uniswap-v2.
			Address:     strings.ToLower(v.TokenContractAddress.Hex()),
			Exchange:    u.config.DexId,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{"0", "0"},
			Tokens:      []*entity.PoolToken{wrappedNativeToken, cToken},
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, pool)
	}

	return pools, nil
}

func (u *PoolsListUpdater) getTokenIdentifier(ctx context.Context) (*big.Int, error) {
	var tokenIdentifier *big.Int
	req := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    tokenManagerABI,
			Target: u.config.TokenManager,
			Method: tokenManagerMethodTokenIdentifier,
		}, []any{&tokenIdentifier})

	if _, err := req.Call(); err != nil {
		return nil, err
	}

	return tokenIdentifier, nil
}

func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (uint64, error) {
	if len(metadataBytes) == 0 {
		return initialTokenId, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return initialTokenId, err
	}

	return metadata.Offset, nil
}

func (u *PoolsListUpdater) newMetadata(newOffset uint64) ([]byte, error) {
	metadataBytes, err := json.Marshal(PoolsListUpdaterMetadata{Offset: newOffset})
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}
