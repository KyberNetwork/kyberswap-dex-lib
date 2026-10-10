package btr

import (
	"context"
	"errors"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

// PoolsListUpdater discovers BTR cores from the PoolFactory on Monad. BTR's
// core is a single multi-asset book (`btr-core`), not one pool per pair, so
// this returns one entity.Pool whose Tokens list is the on-chain roster read
// from the factory (`getPoolTokens`), never a hard-coded list.
type PoolsListUpdater struct {
	config *Config
	client *ethrpc.Client
}

func NewPoolsListUpdater(cfg *Config, client *ethrpc.Client) *PoolsListUpdater {
	if cfg == nil {
		cfg = NewConfig(ChainIDMonad)
	}
	return &PoolsListUpdater{config: cfg, client: client}
}

// GetNewPools validates the pinned core against the factory and reads its token
// roster. BTR cores are CREATE3 and permissioned; the adapter pins the known
// core address and validates it rather than scanning every factory log.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.client == nil {
		return nil, nil, ErrBTRBadState
	}
	core := common.HexToAddress(CreditPoolCore)
	factory := u.config.Factory
	if factory == "" {
		factory = PoolFactory
	}

	var isPool bool
	if _, err := u.client.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI: factoryABI, Target: factory, Method: "isOfficialPool",
		Params: []any{core},
	}, []any{&isPool}).Call(); err != nil {
		return nil, nil, err
	}
	if !isPool {
		return nil, nil, errors.New("btr: pinned core is not an official pool")
	}

	var tokenAddrs []common.Address
	if _, err := u.client.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI: factoryABI, Target: factory, Method: "getPoolTokens",
		Params: []any{core},
	}, []any{&tokenAddrs}).Call(); err != nil {
		return nil, nil, err
	}
	if len(tokenAddrs) < 2 {
		return nil, nil, ErrBTRBadState
	}

	decimals := make([]uint8, len(tokenAddrs))
	req := u.client.NewRequest().SetContext(ctx)
	for i, addr := range tokenAddrs {
		req.AddCall(&ethrpc.Call{
			ABI: erc20ABI, Target: addr.Hex(), Method: "decimals",
		}, []any{&decimals[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, nil, err
	}

	tokens := make([]*entity.PoolToken, len(tokenAddrs))
	roster := make([]string, len(tokenAddrs))
	for i, addr := range tokenAddrs {
		roster[i] = lowerAddr(addr.Hex())
		tokens[i] = &entity.PoolToken{Address: addr.Hex(), Decimals: decimals[i], Swappable: true}
	}

	static, err := json.Marshal(StaticExtra{
		Pool:      core.Hex(),
		Factory:   factory,
		Tokens:    roster,
		BaseToken: "",
		Router:    RouterAddress,
	})
	if err != nil {
		return nil, nil, err
	}

	return []entity.Pool{{
		Address:     core.Hex(),
		Exchange:    DexType,
		Type:        DexType,
		Tokens:      tokens,
		Reserves:    make([]string, len(tokens)),
		StaticExtra: string(static),
		Timestamp:   time.Now().Unix(),
	}}, nil, nil
}

// ---- ABIs used only by discovery ----

var (
	factoryABI = mustABI(`[
	 {"inputs":[{"internalType":"address","name":"pool","type":"address"}],"name":"isOfficialPool","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
	 {"inputs":[{"internalType":"address","name":"pool","type":"address"}],"name":"getPoolTokens","outputs":[{"internalType":"address[]","name":"","type":"address[]"}],"stateMutability":"view","type":"function"}
	]`)
	erc20ABI = mustABI(`[{"inputs":[],"name":"decimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}]`)
)
