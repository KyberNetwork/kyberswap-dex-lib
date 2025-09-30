package midas

import (
	"context"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

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
	pools := make([]entity.Pool, 0)
	for mToken, config := range u.config.MTokens {
		var (
			dvMTokenDataFeed common.Address
			rvMTokenDataFeed common.Address

			dvTokens []common.Address
			rvTokens []common.Address
		)
		if _, err := u.ethrpcClient.
			NewRequest().
			SetContext(ctx).
			AddCall(&ethrpc.Call{
				ABI:    DepositVaultABI,
				Target: config.Dv,
				Method: vGetPaymentTokensMethod,
			}, []any{&dvTokens}).
			AddCall(&ethrpc.Call{
				ABI:    DepositVaultABI,
				Target: config.Dv,
				Method: vMTokenDataFeedMethod,
			}, []any{&dvMTokenDataFeed}).
			AddCall(&ethrpc.Call{
				ABI:    RedemptionVaultABI,
				Target: config.Rv,
				Method: vGetPaymentTokensMethod,
			}, []any{&rvTokens}).
			AddCall(&ethrpc.Call{
				ABI:    RedemptionVaultABI,
				Target: config.Rv,
				Method: vMTokenDataFeedMethod,
			}, []any{&rvMTokenDataFeed}).
			Aggregate(); err != nil {
			logger.Errorf("failed to aggregate vaults %v, %v", config.Dv, config.Rv)
			return nil, nil, err
		}

		if dvMTokenDataFeed != rvMTokenDataFeed {
			logger.Errorf("data feed mismatch for mToken %s, config %v", mToken, config)
			continue
		}

		if len(dvTokens) > 0 {
			dvPool, err := u.initPool(true, config.Dv, mToken, dvTokens, config.DvType)
			if err == nil {
				pools = append(pools, *dvPool)
			}
		}

		if len(rvTokens) > 0 {
			rvPool, err := u.initPool(false, config.Rv, mToken, rvTokens, config.RvType)
			if err == nil {
				pools = append(pools, *rvPool)
			}
		}
	}

	return pools, nil, nil
}

func (u *PoolsListUpdater) initPool(isDv bool, vault, mToken string, paymentTokens []common.Address,
	vaultType VaultType) (*entity.Pool, error) {
	tokens := make([]*entity.PoolToken, 0, len(paymentTokens)+1)
	tokens = append(tokens, &entity.PoolToken{
		Address:   strings.ToLower(mToken),
		Swappable: true,
	})
	for _, token := range paymentTokens {
		tokens = append(tokens, &entity.PoolToken{
			Address:   strings.ToLower(token.String()),
			Swappable: true,
		})
	}

	reserves := lo.Times(len(tokens), func(_ int) string { return "0" })

	staticExtra, err := json.Marshal(StaticExtra{
		IsDv:      isDv,
		VaultType: vaultType,
	})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:     strings.ToLower(vault),
		Exchange:    u.config.DexId,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      tokens,
		StaticExtra: string(staticExtra),
	}, nil
}
