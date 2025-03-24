package lending

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/llamma"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       shared.Config
	ethrpcClient *ethrpc.Client
	client       *resty.Client
	logger       logger.Logger
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *shared.Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	client := resty.New().
		SetBaseURL(config.HTTPConfig.BaseURL).
		SetTimeout(config.HTTPConfig.Timeout.Duration).
		SetRetryCount(config.HTTPConfig.RetryCount)

	lg := logger.WithFields(logger.Fields{
		"dexId":   config.DexID,
		"dexType": DexType,
	})

	return &PoolsListUpdater{
		config:       *config,
		ethrpcClient: ethrpcClient,
		client:       client,
		logger:       lg,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	u.logger.Infof("Start updating pools list...")
	defer func() {
		u.logger.Infof("Finish updating pools list.")
	}()

	// Pool list doesn't get changed often, so only fetch after some minutes
	now := time.Now().UTC()
	var metadata shared.PoolListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			u.logger.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to unmarshal metadataBytes")
			return nil, nil, err
		}

		if now.Sub(metadata.LastRun) < u.config.FetchPoolsMinDuration.Duration {
			u.logger.Debugf("skip fetching new pool %v %v", now, metadata.LastRun)
			return nil, metadataBytes, nil
		}
	}
	metadata.LastRun = now

	lendingVaults, err := u.getLendingVaults(ctx)
	if err != nil {
		return nil, nil, err
	}

	pools, err := u.initPools(ctx, lendingVaults)
	if err != nil {
		return nil, nil, err
	}

	return pools, metadata.ToBytes(), nil
}

func (u *PoolsListUpdater) getLendingVaults(ctx context.Context) ([]LendingVault, error) {
	req := u.client.R().SetContext(ctx)
	var result GetLendingVaultsResult
	resp, err := req.SetResult(&result).Get(fmt.Sprintf(getLendingVaultsEndpoint, u.config.ChainCode))
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() || !result.Success {
		return nil, nil
	}

	return result.Data.LendingVaultData, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, lendingVaults []LendingVault) ([]entity.Pool, error) {
	calls := u.ethrpcClient.NewRequest().SetContext(ctx)
	aCoefficients := make([]*big.Int, len(lendingVaults))
	for i := 0; i < len(lendingVaults); i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    llamma.CurveLlammaABI,
			Target: lendingVaults[i].AmmAddress,
			Method: llamma.LlammaMethodA,
		}, []interface{}{&aCoefficients[i]})
	}
	if _, err := calls.Aggregate(); err != nil {
		return nil, err
	}

	var pools = make([]entity.Pool, 0, len(lendingVaults))
	for i, vault := range lendingVaults {
		borrowedToken := vault.Assets["borrowed"]
		collateralToken := vault.Assets["collateral"]

		staticExtraBytes, err := json.Marshal(llamma.StaticExtra{
			A:             uint256.MustFromBig(aCoefficients[i]),
			UseDynamicFee: true,
		})
		if err != nil {
			u.logger.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to marshal staticExtra")
			return nil, err
		}

		newPool := entity.Pool{
			Address:   strings.ToLower(vault.AmmAddress),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  make([]string, 0, len(vault.Assets)),
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(borrowedToken.Address),
					Symbol:    borrowedToken.Symbol,
					Decimals:  borrowedToken.Decimals,
					Swappable: true,
				},
				{
					Address:   strings.ToLower(collateralToken.Address),
					Symbol:    collateralToken.Symbol,
					Decimals:  collateralToken.Decimals,
					Swappable: true,
				},
			},
			StaticExtra: string(staticExtraBytes),
		}
		pools = append(pools, newPool)
	}

	return pools, nil
}
