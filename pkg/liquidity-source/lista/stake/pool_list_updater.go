package stake

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config         *Config
	tracker        *PoolTracker
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	tracker, _ := NewPoolTracker(config, ethrpcClient)
	return &PoolsListUpdater{config: config, tracker: tracker}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}
	u.hasInitialized = true

	pools := make([]entity.Pool, 0, len(u.config.Pools))
	for _, item := range u.config.Pools {
		// The graph always stores the wrapped form, even when the deposit side was configured
		// native -- but IsNativeUnderlying remembers which it was, so the simulator can report
		// native support truthfully instead of assuming it from the token address alone.
		isNativeUnderlying := valueobject.IsNative(item.UnderlyingToken)
		underlyingToken := item.UnderlyingToken
		if isNativeUnderlying {
			underlyingToken = valueobject.WrapNativeLower(underlyingToken, u.config.ChainID)
		}

		staticExtraBytes, err := json.Marshal(StaticExtra{IsNativeUnderlying: isNativeUnderlying})
		if err != nil {
			return nil, nil, err
		}

		newPool := entity.Pool{
			Address:     strings.ToLower(item.PoolAddress),
			Exchange:    u.config.DexID,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    entity.PoolReserves{"0", "0"},
			StaticExtra: string(staticExtraBytes),
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(underlyingToken), Swappable: true},
				{Address: strings.ToLower(item.ShareToken), Swappable: true},
			},
		}

		newPool, err = u.tracker.GetNewPoolState(ctx, newPool, pool.GetNewPoolStateParams{})
		if err != nil {
			logger.WithFields(logger.Fields{"error": err, "pool_id": newPool.Address}).
				Errorf("failed to get initial lista-stake pool state")
			return nil, nil, err
		}

		pools = append(pools, newPool)
	}

	return pools, nil, nil
}
