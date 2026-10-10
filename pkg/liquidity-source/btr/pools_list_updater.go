package btr

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/samber/lo"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

// factoryABI is the slice of the PoolFactory the lister reads.
var factoryABI = lo.Must(abi.JSON(strings.NewReader(`[
 {"inputs":[{"name":"pool","type":"address"}],"name":"isOfficialPool","outputs":[{"name":"","type":"bool"}],"stateMutability":"view","type":"function"},
 {"inputs":[{"name":"pool","type":"address"}],"name":"getPoolTokens","outputs":[{"name":"","type":"address[]"}],"stateMutability":"view","type":"function"}
]`)))

// PoolsListUpdater discovers the BTR core. The core is a single multi-asset
// book (`btr-core`), not one pool per pair, so it yields one entity.Pool whose
// Tokens are the roster read from the factory (`getPoolTokens`).
type PoolsListUpdater struct {
	config *Config
	client *ethrpc.Client
}

func NewPoolsListUpdater(cfg *Config, client *ethrpc.Client) *PoolsListUpdater {
	if cfg == nil {
		cfg = NewConfig(ChainIDMonad)
	}
	if cfg.Factory == "" {
		cfg.Factory = PoolFactory
	}
	return &PoolsListUpdater{config: cfg, client: client}
}

// GetNewPools validates the pinned core against the factory and reads its token
// roster. BTR cores are CREATE3 and permissioned, so the lister pins the known
// core instead of scanning factory logs. Any non-empty metadata means the core
// is already listed.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadata []byte) ([]entity.Pool, []byte, error) {
	if len(metadata) > 0 {
		return nil, metadata, nil
	}
	if u.client == nil {
		return nil, nil, ErrBTRBadState
	}
	core := common.HexToAddress(CreditPoolCore)

	var isPool bool
	var tokenAddrs []common.Address
	if _, err := u.client.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI: factoryABI, Target: u.config.Factory, Method: "isOfficialPool", Params: []any{core},
		}, []any{&isPool}).
		AddCall(&ethrpc.Call{
			ABI: factoryABI, Target: u.config.Factory, Method: "getPoolTokens", Params: []any{core},
		}, []any{&tokenAddrs}).
		Aggregate(); err != nil {
		return nil, nil, err
	}
	if !isPool {
		return nil, nil, errors.New("btr: pinned core is not an official pool")
	}
	if len(tokenAddrs) < 2 {
		return nil, nil, ErrBTRBadState
	}

	tokens := lo.Map(tokenAddrs, func(a common.Address, _ int) *entity.PoolToken {
		return &entity.PoolToken{Address: hexutil.Encode(a[:]), Swappable: true}
	})
	return []entity.Pool{{
		Address:   hexutil.Encode(core[:]),
		Exchange:  DexType,
		Type:      DexType,
		Tokens:    tokens,
		Reserves:  lo.Times(len(tokens), func(int) string { return "0" }),
		Timestamp: time.Now().Unix(),
	}}, []byte{1}, nil
}
