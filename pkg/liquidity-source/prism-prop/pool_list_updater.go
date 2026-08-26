package prismprop

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools re-lists every pair on every run: getSupportedPairs is a single
// cheap call, and prism-prop has no pair-added event to track a cursor from
// (see titan-prop's KnownVenues cursor for the pattern this would follow if
// that changes). The pool-service layer dedupes by address, so re-listing an
// already-known pair is a no-op.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	var pairs []Pair
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: u.config.RouterAddress,
		Method: methodGetSupportedPairs,
	}, []any{&pairs}).Aggregate(); err != nil {
		return nil, nil, err
	}

	router := common.HexToAddress(u.config.RouterAddress)
	staticExtraBytes, err := json.Marshal(StaticExtra{RouterAddress: hexutil.Encode(router[:])})
	if err != nil {
		return nil, nil, err
	}

	pools := make([]entity.Pool, 0, len(pairs))
	for _, pair := range pairs {
		token0, token1 := hexutil.Encode(pair.Token0[:]), hexutil.Encode(pair.Token1[:])
		pools = append(pools, entity.Pool{
			Address:  poolAddress(token0, token1),
			Exchange: u.config.DexID,
			Type:     DexType,
			Tokens: []*entity.PoolToken{
				{Address: token0, Swappable: true},
				{Address: token1, Swappable: true},
			},
			Reserves:    entity.PoolReserves{"0", "0"},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
			Timestamp:   time.Now().Unix(),
		})
	}
	return pools, nil, nil
}

// poolAddress is synthetic: prism-prop has one router per chain quoting
// every pair, so a fixed "prism" namespace plus the pair already
// disambiguates every pool -- prefixing with the full router address isn't
// needed for uniqueness within this exchange, only across exchanges, which
// the "prism" literal already covers.
func poolAddress(token0, token1 string) string {
	return "prismprop_" + token0 + "_" + token1
}
