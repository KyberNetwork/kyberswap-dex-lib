package upshiftvaultv2

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// PoolsListUpdater emits one [asset, LP token] pool per configured vault, once. Mutable state is a
// placeholder; the tracker fills it.
type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	assets := make([]common.Address, len(u.config.Vaults))
	lps := make([]common.Address, len(u.config.Vaults))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, vault := range u.config.Vaults {
		req.AddCall(&ethrpc.Call{ABI: vaultABI, Target: vault, Method: vaultMethodAsset}, []any{&assets[i]}).
			AddCall(&ethrpc.Call{ABI: vaultABI, Target: vault, Method: vaultMethodLP}, []any{&lps[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(Extra{
		Supply: new(uint256.Int), Valuation: new(uint256.Int), TotalAssets: new(uint256.Int), MaxDeposit: new(uint256.Int), DepositCap: new(uint256.Int),
	})
	if err != nil {
		return nil, nil, err
	}

	pools := make([]entity.Pool, len(u.config.Vaults))
	for i, vault := range u.config.Vaults {
		vaultAddr := common.HexToAddress(vault)
		pools[i] = entity.Pool{
			Address:   hexutil.Encode(vaultAddr[:]),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(assets[i][:]), Swappable: true},
				{Address: hexutil.Encode(lps[i][:]), Swappable: true},
			},
			Extra: string(extraBytes),
		}
	}
	u.hasInitialized = true

	return pools, nil, nil
}
