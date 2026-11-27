package mhocdesk

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

// Metadata is the set of configured pools already emitted, by lower-case address.
type Metadata struct {
	Seen map[string]bool `json:"seen"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools emits each pool once, with its immutables (BASE, QUOTE, PRICE_SCALE, K_SCALE, token decimals). The
// pools are the configured addresses plus, when a registry is configured, every address its pools() returns at the
// time of the call, so a pair the maker lists later is picked up on the next run. A pool removed from the registry
// is not withdrawn here: once the maker stops pushing prices to it, its card expires and it quotes nothing.
// Reserves start at 0; the tracker fills the state on its first refresh.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var md Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &md); err != nil {
			return nil, metadataBytes, err
		}
	}
	if md.Seen == nil {
		md.Seen = map[string]bool{}
	}
	candidates := u.config.Pools
	if common.IsHexAddress(u.config.Registry) {
		var listed []common.Address
		if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
			ABI: registryABI, Target: u.config.Registry, Method: "pools"}, []any{&listed}).Call(); err != nil {
			return nil, metadataBytes, err
		}
		candidates = append(append([]string(nil), candidates...),
			lo.Map(listed, func(a common.Address, _ int) string { return a.Hex() })...)
	}
	var addrs []string
	queued := map[string]bool{}
	for _, a := range candidates {
		a = strings.ToLower(a)
		if common.IsHexAddress(a) && !md.Seen[a] && !queued[a] {
			addrs = append(addrs, a)
			queued[a] = true
		}
	}
	if len(addrs) == 0 {
		return nil, metadataBytes, nil
	}

	n := len(addrs)
	bases, quotes := make([]common.Address, n), make([]common.Address, n)
	scales, kScales := make([]*big.Int, n), make([]*big.Int, n)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, a := range addrs {
		req.AddCall(&ethrpc.Call{ABI: propAMMABI, Target: a, Method: "BASE"}, []any{&bases[i]})
		req.AddCall(&ethrpc.Call{ABI: propAMMABI, Target: a, Method: "QUOTE"}, []any{&quotes[i]})
		req.AddCall(&ethrpc.Call{ABI: propAMMABI, Target: a, Method: "PRICE_SCALE"}, []any{&scales[i]})
		req.AddCall(&ethrpc.Call{ABI: propAMMABI, Target: a, Method: "K_SCALE"}, []any{&kScales[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, metadataBytes, err
	}
	bd, qd := make([]uint8, n), make([]uint8, n)
	req = u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := range addrs {
		req.AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: bases[i].Hex(), Method: abi.Erc20DecimalsMethod}, []any{&bd[i]})
		req.AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: quotes[i].Hex(), Method: abi.Erc20DecimalsMethod}, []any{&qd[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, metadataBytes, err
	}

	pools := make([]entity.Pool, 0, n)
	now := time.Now().Unix()
	for i, a := range addrs {
		se, err := json.Marshal(StaticExtra{BaseDecimals: bd[i], QuoteDecimals: qd[i], PriceScale: scales[i].String(),
			KScale: uint32(kScales[i].Uint64())})
		if err != nil {
			return nil, metadataBytes, err
		}
		pools = append(pools, entity.Pool{
			Address:   a,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(bases[i][:]), Decimals: bd[i], Swappable: true},
				{Address: hexutil.Encode(quotes[i][:]), Decimals: qd[i], Swappable: true},
			},
			StaticExtra: string(se),
		})
		md.Seen[a] = true
	}
	newMd, err := json.Marshal(md)
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, newMd, nil
}
