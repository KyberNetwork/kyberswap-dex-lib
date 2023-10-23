package vooi

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool": p.Address}).Debug("Start getting new pool state")
	defer func() {
		logger.
			WithFields(
				logger.Fields{
					"dex_id":      t.config.DexID,
					"pool":        p.Address,
					"duration_ms": time.Since(startTime).Milliseconds(),
				}).
			Debug("Finish getting new pool state")
	}()

	ctx = util.NewContextWithTimestamp(ctx)

	// Get lastIndex
	lastIndex, err := t.getLastIndex(ctx, p.Address)
	if err != nil {
		return entity.Pool{}, err
	}

	var (
		paused bool
		a      *big.Int
		lpFee  *big.Int
		assets = make([]Asset, lastIndex)
	)

	getPoolState := t.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < lastIndex; i++ {
		getPoolState.AddCall(
			&ethrpc.Call{
				ABI:    poolABI,
				Target: p.Address,
				Method: poolMethodIndexToAsset,
				Params: []interface{}{big.NewInt(int64(i))},
			}, []interface{}{&assets[i]})
	}

	getPoolState.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodPaused,
		Params: nil,
	}, []interface{}{&paused})

	getPoolState.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodA,
		Params: nil,
	}, []interface{}{&a})

	getPoolState.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: poolMethodLpFee,
		Params: nil,
	}, []interface{}{&lpFee})

	if _, err = getPoolState.TryAggregate(); err != nil {
		logger.
			WithFields(
				logger.Fields{
					"liquiditySource": DexTypeVooi,
					"poolAddress":     p.Address,
					"error":           err,
				}).
			Error("failed to get pool state")

		return entity.Pool{}, ErrFailedToGetPoolState
	}

	poolTokens := make([]*entity.PoolToken, 0, len(assets))
	reserves := make([]string, 0, len(assets))
	assetByToken := make(map[string]Asset, len(assets))
	indexByToken := make(map[string]int, len(assets))

	for i, asset := range assets {
		token := strings.ToLower(asset.Token.Hex())

		poolTokens = append(poolTokens, &entity.PoolToken{
			Address:   token,
			Decimals:  asset.Decimals,
			Swappable: true,
		})
		reserves = append(reserves, asset.Cash.String())
		assetByToken[token] = asset
		indexByToken[token] = i
	}

	poolExtra := PoolExtra{
		AssetByToken: assetByToken,
		IndexByToken: indexByToken,
		Paused:       paused,
		A:            a,
		LPFee:        lpFee,
	}

	poolExtraBytes, err := json.Marshal(poolExtra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Tokens = poolTokens
	p.Reserves = reserves
	p.Extra = string(poolExtraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) getLastIndex(ctx context.Context, address string) (int, error) {
	var lastIndex *big.Int

	getLastIndexRequest := t.ethrpcClient.NewRequest().SetContext(ctx)
	getLastIndexRequest.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: address,
		Method: poolMethodLastIndex,
		Params: nil,
	}, []interface{}{&lastIndex})

	if _, err := getLastIndexRequest.Call(); err != nil {
		logger.
			WithFields(
				logger.Fields{
					"liquiditySource": DexTypeVooi,
					"poolAddress":     address,
					"error":           err,
				}).
			Error("failed to get lastIndex")

		return 0, ErrFailedToGetLastIndex
	}

	return int(lastIndex.Int64()), nil
}
