package pancakeprop

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	cfg    *Config
	client *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, client *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg, client: client}
}

type tokenPair struct {
	Token0 common.Address
	Token1 common.Address
}

// GetNewPools lists one pool per entry pair. The pricing module only appends pairs, so the
// pair id (index + 1) is stable; metadata holds the number of pairs already emitted.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadata []byte) ([]entity.Pool, []byte, error) {
	var count int
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &count); err != nil {
			return nil, metadata, err
		}
	}

	var pricing, vault common.Address
	var pairs []tokenPair
	req := u.client.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{ABI: entryABI, Target: u.cfg.Entry, Method: methodPricing}, []any{&pricing})
	req.AddCall(&ethrpc.Call{ABI: entryABI, Target: u.cfg.Entry, Method: methodVault}, []any{&vault})
	req.AddCall(&ethrpc.Call{ABI: entryABI, Target: u.cfg.Entry, Method: methodGetSupportedPairs}, []any{&pairs})
	if _, err := req.Aggregate(); err != nil {
		return nil, metadata, err
	}
	if count >= len(pairs) {
		return nil, metadata, nil
	}

	entry := strings.ToLower(u.cfg.Entry)
	pools := make([]entity.Pool, 0, len(pairs)-count)
	for i := count; i < len(pairs); i++ {
		staticExtra, err := json.Marshal(StaticExtra{
			Entry:   entry,
			Pricing: hexutil.Encode(pricing[:]),
			Vault:   hexutil.Encode(vault[:]),
			PairID:  uint64(i + 1),
		})
		if err != nil {
			return nil, metadata, err
		}
		token0, token1 := pairs[i].Token0, pairs[i].Token1
		// A pair has no address of its own; derive a unique one from the entry and its tokens.
		hash := crypto.Keccak256(common.HexToAddress(entry).Bytes(), token0.Bytes(), token1.Bytes())
		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(hash[12:]),
			Exchange:  u.cfg.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(token0[:]), Swappable: true},
				{Address: hexutil.Encode(token1[:]), Swappable: true},
			},
			Reserves:    entity.PoolReserves{"0", "0"},
			Extra:       "{}",
			StaticExtra: string(staticExtra),
		})
	}

	next, err := json.Marshal(len(pairs))
	return pools, next, err
}
