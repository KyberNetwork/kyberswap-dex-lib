package everlongclamm

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

// GetNewPools lists the single configured CLAMM pool. The pool set is fixed per
// deployment (the ALM is the sole LP of exactly one pool), so this initializes once;
// it still verifies the pool exists on-chain before listing it.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	var (
		slot0       slot0Raw
		poolID             = common.HexToHash(u.config.PoolID)
		blockNumber uint64 = 0
	)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    poolManagerABI,
		Target: u.config.PoolManager,
		Method: poolManagerMethodGetSlot0,
		Params: []any{poolID},
	}, []any{&slot0})
	resp, err := req.Aggregate()
	if err != nil {
		return nil, nil, err
	}
	if resp.BlockNumber != nil {
		blockNumber = resp.BlockNumber.Uint64()
	}
	if slot0.SqrtPriceX96 == nil || slot0.SqrtPriceX96.Sign() == 0 {
		return nil, nil, ErrPoolNotInitialized
	}

	staticExtra, err := json.Marshal(StaticExtra{
		PoolManager: strings.ToLower(u.config.PoolManager),
		ALM:         strings.ToLower(u.config.ALM),
		Router:      strings.ToLower(u.config.Router),
		Hook:        strings.ToLower(u.config.Hook),
		Fee:         u.config.Fee,
		Parameters:  strings.ToLower(u.config.Parameters),
		TickSpacing: tickSpacingFromParameters(u.config.Parameters),
	})
	if err != nil {
		return nil, nil, err
	}

	u.hasInitialized = true

	return []entity.Pool{
		{
			Address:   strings.ToLower(u.config.PoolID),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(u.config.Currency0), Swappable: true},
				{Address: strings.ToLower(u.config.Currency1), Swappable: true},
			},
			Reserves:    entity.PoolReserves{"0", "0"},
			StaticExtra: string(staticExtra),
			Extra:       "{}",
			BlockNumber: blockNumber,
		},
	}, nil, nil
}

// tickSpacingFromParameters unpacks the int24 tick spacing from bits [16,40) of
// PoolKey.parameters (Infinity CLPoolParametersHelper layout).
func tickSpacingFromParameters(parameters string) int {
	raw := new(big.Int).SetBytes(common.HexToHash(parameters).Bytes())
	spacing := int(new(big.Int).And(new(big.Int).Rsh(raw, 16), big.NewInt(0xffffff)).Int64())
	if spacing >= 1<<23 { // sign-extend int24
		spacing -= 1 << 24
	}
	return spacing
}
