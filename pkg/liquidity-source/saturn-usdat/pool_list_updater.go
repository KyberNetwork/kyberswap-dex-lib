package saturnusdat

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// PoolsListUpdater emits the single static USDC -> USDat pool once. Mutable
// state is a placeholder; the tracker fills it.
type PoolsListUpdater struct {
	config         *Config
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, _ *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg}
}

func (u *PoolsListUpdater) GetNewPools(_ context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	extraBytes, err := json.Marshal(Extra{
		AssetCap:          new(uint256.Int),
		AssetBalance:      new(uint256.Int),
		AssetDecimals:     DefaultDecimals,
		ExtensionDecimals: DefaultDecimals,
	})
	if err != nil {
		return nil, nil, err
	}
	staticExtraBytes, err := json.Marshal(StaticExtra{Facility: Facility, Asset: USDC})
	if err != nil {
		return nil, nil, err
	}
	u.hasInitialized = true

	return []entity.Pool{{
		Address:   USDat,
		Exchange:  u.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: USDC, Swappable: true},
			{Address: USDat, Swappable: true},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}}, nil, nil
}
