package saddle

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:         cfg,
		ethrpcClient:   ethrpcClient,
		hasInitialized: false,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		return nil, nil, nil
	}

	pools, err := d.initPools(ctx)
	if err != nil {
		return nil, nil, err
	}

	return pools, nil, nil
}

func (d *PoolsListUpdater) initPools(ctx context.Context) ([]entity.Pool, error) {
	if d.config.PoolPath == "" {
		return nil, nil
	}

	byteData, ok := bytesByPath[d.config.PoolPath]
	if !ok {
		logger.Errorf("misconfigured poolPath")
		return nil, errors.New("misconfigured poolPath")
	}
	var poolItems []PoolItem
	if err := json.Unmarshal(byteData, &poolItems); err != nil {
		logger.WithFields(logger.Fields{
			"path":  d.config.PoolPath,
			"error": err,
		}).Errorf("failed to unmarshal the pool path config file")

		return nil, err
	}

	logger.Infof("[%s] got %v from pool path config file", d.config.DexID, len(poolItems))

	pools, err := d.processBatch(ctx, poolItems)
	if err != nil {
		return nil, err
	}
	d.hasInitialized = true

	return pools, nil
}

func (d *PoolsListUpdater) processBatch(ctx context.Context, poolItems []PoolItem) ([]entity.Pool, error) {
	var swapStorages = make([]SwapStorage, len(poolItems))

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i := range poolItems {
		calls.AddCall(&ethrpc.Call{
			ABI:    swapFlashLoanABI,
			Target: poolItems[i].ID,
			Method: poolMethodSwapStorage,
			Params: nil,
		}, []interface{}{&swapStorages[i]})
	}
	if _, err := calls.TryAggregate(); err != nil {
		logger.Errorf("failed to try aggregate call with error %v", err)
		return nil, err
	}

	var pools = make([]entity.Pool, 0, len(poolItems))
	for i, pool := range poolItems {
		var tokens = make([]*entity.PoolToken, 0, len(pool.Tokens))
		var reserves = make([]string, 0, len(pool.Tokens))
		var precisionMultipliers = make([]string, 0, len(pool.Tokens))

		for _, token := range pool.Tokens {
			tokenModel := entity.PoolToken{
				Address:   token.Address,
				Weight:    defaultWeight,
				Swappable: true,
			}
			tokens = append(tokens, &tokenModel)
			precisionMultipliers = append(precisionMultipliers, token.Precision)
			reserves = append(reserves, zeroSrting)
		}

		staticExtra := StaticExtra{
			LpToken:              strings.ToLower(swapStorages[i].LpToken.Hex()),
			PrecisionMultipliers: precisionMultipliers,
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": pool.ID,
				"error":       err,
			}).Errorf("failed to marshal static extra data")
			return nil, err
		}

		var newPool = entity.Pool{
			Address:     strings.ToLower(pool.ID),
			Exchange:    d.config.DexID,
			Type:        DexTypeSaddle,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
