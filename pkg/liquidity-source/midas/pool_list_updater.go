package midas

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
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

	pools := make([]entity.Pool, 0)

	for mToken, config := range u.config.MTokens {
		var (
			paymentTokens  []common.Address
			mTokenDataFeed common.Address
		)
		if _, err := u.ethrpcClient.
			NewRequest().
			SetContext(ctx).
			AddCall(&ethrpc.Call{
				ABI:    DepositVaultABI,
				Target: config.DepositVault,
				Method: depositVaultGetPaymentTokensMethod,
			}, []any{&paymentTokens}).
			AddCall(&ethrpc.Call{
				ABI:    DepositVaultABI,
				Target: config.DepositVault,
				Method: depositVaultMTokenDataFeedMethod,
			}, []any{&mTokenDataFeed}).Aggregate(); err != nil {
			return nil, nil, err
		}

		for _, token := range paymentTokens {
			var tokenConfig struct {
				DataFeed  common.Address
				Fee       *big.Int
				Allowance *big.Int
				Stable    bool
			}
			if _, err := u.ethrpcClient.
				NewRequest().
				SetContext(ctx).
				AddCall(&ethrpc.Call{
					ABI:    DepositVaultABI,
					Target: config.DepositVault,
					Method: depositVaultTokensConfigMethod,
					Params: []any{token},
				}, []any{&tokenConfig}).Call(); err != nil {
				return nil, nil, err
			}

			staticExtra, err := json.Marshal(StaticExtra{
				MTokenDataFeed:      mTokenDataFeed,
				DataFeed:            tokenConfig.DataFeed,
				DepositVaultType:    config.DepositVaultType,
				DepositVault:        common.HexToAddress(config.DepositVault),
				RedemptionVaultType: config.RedemptionVaultType,
				RedemptionVault:     common.HexToAddress(config.RedemptionVault),
			})
			if err != nil {
				return nil, nil, err
			}

			pools = append(pools, entity.Pool{
				Address:   strings.Join([]string{strings.ToLower(mToken), strings.ToLower(token.String())}, "-"),
				Exchange:  u.config.DexId,
				Type:      DexType,
				Timestamp: time.Now().Unix(),
				Reserves:  entity.PoolReserves{"0", "0"},
				Tokens: []*entity.PoolToken{{
					Address:   strings.ToLower(mToken),
					Swappable: true,
				}, {
					Address:   strings.ToLower(token.String()),
					Swappable: true,
				}},
				StaticExtra: string(staticExtra),
			})
		}
	}

	u.hasInitialized = true

	return pools, nil, nil
}
