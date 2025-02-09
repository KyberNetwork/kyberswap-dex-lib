package skysavings

import (
	"context"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/savingsdai"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/savingsusds"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	initialized  bool
}

var _ = poollist.RegisterFactoryCE(savingsdai.DexType, NewPoolListUpdater)
var _ = poollist.RegisterFactoryCE(savingsusds.DexType, NewPoolListUpdater)

func NewPoolListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolListUpdater {
	return &PoolListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(_ context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexType": u.config.DexID,
	}).Infof("Start updating pools list ...")
	defer func() {
		logger.WithFields(logger.Fields{
			"dexType": u.config.DexID,
		}).Infof("Finish updating pools list.")
	}()

	if u.initialized {
		logger.WithFields(logger.Fields{
			"dexType": u.config.DexID,
		}).Infof("Pools have been initialized.")
		return nil, metadataBytes, nil
	}

	staticExtra := StaticExtra{
		Pot:               u.config.Pot,
		SavingsRateSymbol: u.config.SavingsRateSymbol,
	}

	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return nil, nil, err
	}

	pool := entity.Pool{
		Address:  u.config.SavingsToken,
		Exchange: u.config.DexID,
		Type:     u.config.DexID,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{
				Address:   strings.ToLower(u.config.DepositToken),
				Swappable: true,
			},
			{
				Address:   strings.ToLower(u.config.SavingsToken),
				Swappable: true,
			},
		},
		StaticExtra: string(staticExtraBytes),
	}

	u.initialized = true

	return []entity.Pool{pool}, metadataBytes, nil
}
