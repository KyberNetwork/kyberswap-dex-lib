package midas

import (
	"context"
	"errors"
	"github.com/samber/lo"
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

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
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
	if u.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	configByte, ok := bytesByPath[u.config.ConfigPath]
	if !ok {
		return nil, nil, errors.New("misconfigured config path")
	}

	var mTokenConfigs map[string]MTokenConfig
	if err := json.Unmarshal(configByte, &mTokenConfigs); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal config")
		return nil, nil, err
	}

	pools := make([]entity.Pool, 0)
	for mTokenSymbol, config := range mTokenConfigs {
		var (
			depositMTokenDataFeed common.Address
			redeemMTokenDataFeed  common.Address

			depositPaymentTokens []common.Address
			redeemPaymentTokens  []common.Address
		)
		if _, err := u.ethrpcClient.
			NewRequest().
			SetContext(ctx).
			AddCall(&ethrpc.Call{
				ABI:    DepositVaultABI,
				Target: config.DepositVault,
				Method: vaultGetPaymentTokensMethod,
			}, []any{&depositPaymentTokens}).
			AddCall(&ethrpc.Call{
				ABI:    DepositVaultABI,
				Target: config.DepositVault,
				Method: vaultMTokenDataFeedMethod,
			}, []any{&depositMTokenDataFeed}).
			AddCall(&ethrpc.Call{
				ABI:    RedemptionVaultABI,
				Target: config.RedemptionVault,
				Method: vaultGetPaymentTokensMethod,
			}, []any{&redeemPaymentTokens}).
			AddCall(&ethrpc.Call{
				ABI:    RedemptionVaultABI,
				Target: config.RedemptionVault,
				Method: vaultMTokenDataFeedMethod,
			}, []any{&redeemMTokenDataFeed}).
			Aggregate(); err != nil {
			logger.Errorf("failed to aggregate vaults %v, %v", config.DepositVault, config.RedemptionVault)
			return nil, nil, err
		}

		if depositMTokenDataFeed != redeemMTokenDataFeed {
			logger.Errorf("data feed mismatch for mToken %s, config %v", mTokenSymbol, config)
			continue
		}

		for _, token := range depositPaymentTokens {
			pool, err := u.initPool(ctx, true, config.DepositVault, config.MToken, token.String(),
				config.DepositVaultType)
			if err != nil {
				logger.Errorf("failed to initialize deposit pool")
				continue
			}
			pools = append(pools, *pool)
		}

		for _, token := range redeemPaymentTokens {
			pool, err := u.initPool(ctx, false, config.RedemptionVault, config.MToken, token.String(),
				config.RedemptionVaultType)
			if err != nil {
				logger.Errorf("failed to initialize redeem pool")
				continue
			}
			pools = append(pools, *pool)
		}
	}

	u.hasInitialized = true

	return pools, nil, nil
}

func (u *PoolsListUpdater) initPool(ctx context.Context, isDepositVault bool,
	vault, mToken, token string, vaultType string,
) (*entity.Pool, error) {
	var tokenConfig struct {
		DataFeed  common.Address
		Fee       *big.Int
		Allowance *big.Int
		Stable    bool
	}

	req := u.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    lo.Ternary(isDepositVault, DepositVaultABI, RedemptionVaultABI),
			Target: vault,
			Method: vaultTokensConfigMethod,
			Params: []any{common.HexToAddress(token)},
		}, []any{&tokenConfig})
	if _, err := req.Call(); err != nil {
		logger.Errorf("failed to get tokenConfigs, vault %v, token %v", vault, token)
		return nil, err
	}

	staticExtra, err := json.Marshal(StaticExtra{
		IsDepositVault: isDepositVault,
		VaultType:      vaultType,
	})
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:   strings.ToLower(vault),
		Exchange:  u.config.DexId,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{{
			Address:   strings.ToLower(mToken),
			Swappable: true,
		}, {
			Address:   strings.ToLower(token),
			Swappable: true,
		}},
		StaticExtra: string(staticExtra),
	}, nil
}
