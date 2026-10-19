package upshiftusdf

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// PoolsListUpdater emits the single static USDC -> USDF pool once. Mutable state is a
// placeholder; the tracker fills it.
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
		Apr: new(uint256.Int), TokenPrice: new(uint256.Int), MinDeposit: new(uint256.Int),
		MaxSupply: new(uint256.Int), TotalSupply: new(uint256.Int),
	})
	if err != nil {
		return nil, nil, err
	}
	u.hasInitialized = true

	return []entity.Pool{{
		Address:   Vault,
		Exchange:  u.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  []string{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: USDC, Swappable: true},
			{Address: USDF, Swappable: true},
		},
		Extra: string(extraBytes),
	}}, nil, nil
}
