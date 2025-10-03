package cusd

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	var assets []common.Address
	if _, err := u.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    capTokenABI,
			Target: u.config.Vault,
			Method: capTokenAssetsMethod,
		}, []any{&assets}).
		Call(); err != nil {
		logger.Errorf("failed to get assets")
		return nil, nil, err
	}

	tokens := make([]*entity.PoolToken, 0, len(assets)+1)

	for _, token := range assets {
		tokens = append(tokens, &entity.PoolToken{
			Address:   strings.ToLower(token.String()),
			Swappable: true,
		})
	}
	tokens = append(tokens, &entity.PoolToken{
		Address:   strings.ToLower(u.config.Vault),
		Swappable: true,
	})

	reserves := lo.Times(len(tokens), func(_ int) string { return "0" })

	return []entity.Pool{
		{
			Address:   strings.ToLower(u.config.Vault),
			Exchange:  u.config.DexId,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  reserves,
			Tokens:    tokens,
		},
	}, nil, nil
}
