package thogprop

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// ErrTokenOrderMismatch signals that getTokens(poolId) returned an address
// order different from tokenTable -- every priceWord/priceShift/categoryShift
// in math.go is keyed by that fixed index, so a mismatch here would silently
// corrupt every quote. Stop rather than guess.
var ErrTokenOrderMismatch = errors.New("thog-prop: getTokens() order does not match the documented token table")

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools registers the single fixed 8-token pool ThogAMM's proxy
// serves. getPoolIds()/getTokens() are read live and cross-checked against
// the configured pool ID and the hardcoded tokenTable order -- this is a
// one-time discovery gate, not a per-refresh check (pool_tracker.go handles
// ongoing state).
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if len(metadataBytes) != 0 {
		// Already discovered in a prior run; this is a single fixed pool.
		return nil, metadataBytes, nil
	}

	var poolIds [][32]byte
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: thogAMMABI, Target: u.config.Proxy, Method: "getPoolIds"}, []any{&poolIds}).
		Call(); err != nil {
		return nil, metadataBytes, err
	}

	found := false
	for _, id := range poolIds {
		if strings.EqualFold(hexutil.Encode(id[:]), u.config.PoolId) {
			found = true
			break
		}
	}
	if !found {
		return nil, metadataBytes, errors.New("thog-prop: configured poolId not present in live getPoolIds()")
	}

	poolIdBytes := common.HexToHash(u.config.PoolId)
	var liveTokens []common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: thogAMMABI, Target: u.config.Proxy, Method: "getTokens", Params: []any{poolIdBytes}}, []any{&liveTokens}).
		Call(); err != nil {
		return nil, metadataBytes, err
	}

	want := tokenAddresses()
	if len(liveTokens) != len(want) {
		return nil, metadataBytes, ErrTokenOrderMismatch
	}
	for i, addr := range liveTokens {
		if !strings.EqualFold(hexutil.Encode(addr[:]), want[i]) {
			return nil, metadataBytes, ErrTokenOrderMismatch
		}
	}

	staticExtra, err := json.Marshal(StaticExtra{PoolId: u.config.PoolId})
	if err != nil {
		return nil, metadataBytes, err
	}

	poolTokens := make([]*entity.PoolToken, len(tokenTable))
	reserves := make(entity.PoolReserves, len(tokenTable))
	for i, t := range tokenTable {
		poolTokens[i] = &entity.PoolToken{Address: t.Address, Swappable: true}
		reserves[i] = "0"
	}

	pool := entity.Pool{
		Address:     strings.ToLower(u.config.Proxy),
		Exchange:    u.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      poolTokens,
		StaticExtra: string(staticExtra),
	}

	newMetadataBytes, err := json.Marshal(map[string]bool{"discovered": true})
	if err != nil {
		return nil, metadataBytes, err
	}
	return []entity.Pool{pool}, newMetadataBytes, nil
}
