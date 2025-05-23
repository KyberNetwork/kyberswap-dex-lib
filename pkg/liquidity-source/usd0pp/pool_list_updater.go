package usd0pp

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client

		hasInitialized bool
	}
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	startTime := time.Now()
	u.hasInitialized = true
	logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Debug("Start getting new pools")

	extra, blockNumber, err := getExtra(ctx, u.ethrpcClient)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      DexType,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return []entity.Pool{
		{
			Address:   USD0PP,
			Reserves:  []string{defaultReserves, defaultReserves},
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(USD0),
					Symbol:    "USD0",
					Decimals:  18,
					Swappable: true,
				},
				{
					Address:   strings.ToLower(USD0PP),
					Symbol:    "USD0++",
					Decimals:  18,
					Swappable: true,
				},
			},
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

func getExtra(ctx context.Context, client *ethrpc.Client) (PoolExtra, uint64, error) {
	var (
		paused    bool
		endTime   *big.Int
		startTime *big.Int
	)

	calls := client.NewRequest()
	calls.SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    usd0ppABI,
		Target: USD0PP,
		Method: usd0ppMethodGetStartTime,
		Params: []interface{}{},
	}, []interface{}{&startTime})
	calls.AddCall(&ethrpc.Call{
		ABI:    usd0ppABI,
		Target: USD0PP,
		Method: usd0ppMethodGetEndTime,
		Params: []interface{}{},
	}, []interface{}{&endTime})
	calls.AddCall(&ethrpc.Call{
		ABI:    usd0ppABI,
		Target: USD0PP,
		Method: usd0ppMethodPaused,
		Params: []interface{}{},
	}, []interface{}{&paused})

	resp, err := calls.Aggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	return PoolExtra{
		Paused:    paused,
		StartTime: int64(startTime.Uint64()),
		EndTime:   int64(endTime.Uint64()),
	}, resp.BlockNumber.Uint64(), nil
}
