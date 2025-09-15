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
	"github.com/samber/lo"

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
			return nil, nil, err
		}

		if depositMTokenDataFeed != redeemMTokenDataFeed {
			logger.Errorf("data feed mismatch for mToken %s, deposit vault %s, redemption vault %s",
				mToken, config.DepositVault, config.RedemptionVault)
			continue
		}

		uniqueTokens := make([]common.Address, 0, len(depositPaymentTokens)+len(redeemPaymentTokens))
		uniqueTokens = append(uniqueTokens, depositPaymentTokens...)
		uniqueTokens = append(uniqueTokens, redeemPaymentTokens...)
		uniqueTokens = lo.Uniq(uniqueTokens)

		for _, token := range uniqueTokens {
			var tokenConfig struct {
				DataFeed  common.Address
				Fee       *big.Int
				Allowance *big.Int
				Stable    bool
			}

			canDeposit := lo.Contains(depositPaymentTokens, token)
			canRedeem := lo.Contains(redeemPaymentTokens, token)

			req := u.ethrpcClient.
				NewRequest().
				SetContext(ctx).
				AddCall(&ethrpc.Call{
					ABI:    DepositVaultABI,
					Target: config.DepositVault,
					Method: vaultTokensConfigMethod,
					Params: []any{token},
				}, []any{&tokenConfig})

			if _, err := req.Aggregate(); err != nil {
				return nil, nil, err
			}

			staticExtra, err := json.Marshal(StaticExtra{
				MTokenDataFeed: depositMTokenDataFeed.String(),
				DataFeed:       tokenConfig.DataFeed.String(),

				CanDeposit: canDeposit,
				CanRedeem:  canRedeem,

				DepositVaultType:    config.DepositVaultType,
				DepositVault:        config.DepositVault,
				RedemptionVaultType: config.RedemptionVaultType,
				RedemptionVault:     config.RedemptionVault,
			})
			if err != nil {
				return nil, nil, err
			}

			pools = append(pools, entity.Pool{
				Address: strings.Join([]string{
					strings.ToLower(mToken),
					strings.ToLower(token.String()),
				}, "-"),
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
