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

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools discovers the current registry; balances are loaded by the tracker.
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

	if len(liveTokens) < 1 || len(liveTokens) > maxTokens {
		return nil, metadataBytes, ErrInvalidState
	}
	for i, address := range liveTokens {
		if address == (common.Address{}) {
			return nil, metadataBytes, ErrInvalidState
		}
		for j := 0; j < i; j++ {
			if address == liveTokens[j] {
				return nil, metadataBytes, ErrInvalidState
			}
		}
	}

	staticExtra, err := json.Marshal(StaticExtra{PoolId: u.config.PoolId})
	if err != nil {
		return nil, metadataBytes, err
	}

	poolTokens := make([]*entity.PoolToken, len(liveTokens))
	reserves := make(entity.PoolReserves, len(liveTokens))
	for i, t := range liveTokens {
		poolTokens[i] = &entity.PoolToken{Address: hexutil.Encode(t[:]), Swappable: true}
		reserves[i] = "0"
	}

	pool := entity.Pool{
		Address:     hexutil.Encode(common.FromHex(u.config.Proxy)),
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
