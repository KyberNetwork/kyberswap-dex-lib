package musd

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client

		hasInitialized bool
	}
)

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
			Address:   MUSD,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{defaultReserves, defaultReserves},
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(MUSD),
					Symbol:    "mUSD",
					Decimals:  18,
					Name:      "Mantle USD",
					Swappable: true,
				},
				{
					Address:   strings.ToLower(USDY),
					Symbol:    "USDY",
					Decimals:  18,
					Name:      "Ondo U.S. Dollar Yield",
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
		paused          bool
		oraclePriceData OraclePriceData
	)

	calls := client.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    mUSDABI,
		Target: MUSD,
		Method: mUSDMethodPaused,
		Params: []interface{}{},
	}, []interface{}{&paused})
	calls.AddCall(&ethrpc.Call{
		ABI:    rwaDynamicOracleABI,
		Target: RWADynamicOracle,
		Method: rwaDynamicOracleMethodGetPriceData,
		Params: []interface{}{},
	}, []interface{}{&oraclePriceData})

	resp, err := calls.Aggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	return PoolExtra{
		Paused:         paused,
		OraclePrice:    uint256.MustFromBig(oraclePriceData.Price),
		PriceTimestamp: oraclePriceData.Timestamp.Uint64(),
	}, resp.BlockNumber.Uint64(), nil
}
