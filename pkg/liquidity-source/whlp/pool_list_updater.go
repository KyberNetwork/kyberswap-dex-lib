package whlp

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	hasInit      bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(_ context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInit {
		return nil, nil, nil
	}
	u.hasInit = true

	staticExtraBytes, _ := json.Marshal(StaticExtra{
		Accountant: common.HexToAddress(u.config.AccountantAddress),
		Depositor:  common.HexToAddress(u.config.DepositorAddress),
		QuoteAsset: common.HexToAddress(u.config.QuoteAssetAddress),
	})

	return []entity.Pool{{
		Address:  u.config.VaultAddress,
		Exchange: string(valueobject.ExchangeWhlp),
		Type:     DexType,
		Timestamp: time.Now().Unix(),
		Reserves: []string{unlimitedReserve, unlimitedReserve},
		Tokens: []*entity.PoolToken{
			{Address: u.config.VaultAddress, Decimals: 6, Symbol: "WHLP"},
			{Address: u.config.QuoteAssetAddress, Decimals: 6, Symbol: "USDT0"},
		},
		StaticExtra: string(staticExtraBytes),
		Extra:       "{}",
	}}, nil, nil
}
