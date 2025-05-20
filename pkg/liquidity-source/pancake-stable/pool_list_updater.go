package pancakestable

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
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

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	dexID := u.config.DexID

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	allPoolsLength, err := u.getAllPoolsLength(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID}).
			Error("allPoolsLength failed")

		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Warn("getOffset failed")
	}

	batchSize := u.getBatchSize(allPoolsLength, u.config.NewPoolLimit, offset)

	poolAddresses, err := u.listPoolAddresses(ctx, offset, batchSize)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("listPoolAddresses failed")

		return nil, metadataBytes, err
	}

	pools, err := u.initPools(ctx, poolAddresses)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initPools failed")

		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + batchSize)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("newMetadata failed")

		return nil, metadataBytes, err
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getAllPoolsLength(ctx context.Context) (int, error) {
	var allPoolsLength *big.Int

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodPairLength,
		Params: nil,
	}, []any{&allPoolsLength})

	if _, err := req.Call(); err != nil {
		return 0, err
	}

	return int(allPoolsLength.Int64()), nil
}

func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}

	return metadata.Offset, nil
}

func (u *PoolsListUpdater) listPoolAddresses(ctx context.Context, offset int, batchSize int) ([]common.Address, error) {
	result := make([]common.Address, batchSize)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i := 0; i < batchSize; i++ {
		index := big.NewInt(int64(offset + i))

		req.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryMethodSwapPairContract,
			Params: []any{index},
		}, []any{&result[i]})
	}

	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}

	var poolAddresses []common.Address
	for i, isSuccess := range resp.Result {
		if !isSuccess {
			continue
		}

		poolAddresses = append(poolAddresses, result[i])
	}

	return poolAddresses, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	tokensByPool, err := u.listPoolTokens(ctx, poolAddresses)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolAddresses))
	for i, poolAddress := range poolAddresses {
		numCoins := len(tokensByPool[i])
		staticPoolData, err := u.getPoolStaticData(ctx, poolAddress.Hex(), numCoins)
		if err != nil {
			return nil, err
		}

		extraBytes, err := json.Marshal(&staticPoolData)
		if err != nil {
			return nil, err
		}

		var tokens = make([]*entity.PoolToken, 0, numCoins)
		for j := range numCoins {
			tokens = append(tokens, &entity.PoolToken{
				Address:   strings.ToLower(tokensByPool[i][j].Hex()),
				Swappable: true,
			})
		}

		var newPool = entity.Pool{
			Address:     strings.ToLower(poolAddress.Hex()),
			Exchange:    u.config.DexID,
			Type:        curve.PoolTypeBase,
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{"0", "0"},
			Tokens:      tokens,
			StaticExtra: string(extraBytes),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}

func (u *PoolsListUpdater) listPoolTokens(ctx context.Context, poolAddresses []common.Address) ([][]common.Address, error) {
	var (
		req        = u.ethrpcClient.NewRequest().SetContext(ctx)
		totalCoins = make([]*big.Int, len(poolAddresses))
	)

	for i, poolAddress := range poolAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress.Hex(),
			Method: poolMethodNCoins,
			Params: nil,
		}, []any{&totalCoins[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	var (
		req2             = u.ethrpcClient.NewRequest().SetContext(ctx)
		listTokensResult = make([][]common.Address, 0, len(poolAddresses))
	)

	for i, poolAddress := range poolAddresses {
		totalCoins := totalCoins[i]
		if totalCoins != nil && totalCoins.Sign() > 0 {
			var tokens = make([]common.Address, totalCoins.Int64())

			for j := range totalCoins.Int64() {
				req2.AddCall(&ethrpc.Call{
					ABI:    poolABI,
					Target: poolAddress.Hex(),
					Method: poolMethodCoins,
					Params: []any{big.NewInt(j)},
				}, []any{&tokens[j]})
			}

			listTokensResult = append(listTokensResult, tokens)
		}
	}
	if _, err := req2.Aggregate(); err != nil {
		return nil, err
	}

	return listTokensResult, nil
}

func (d *PoolsListUpdater) getPoolStaticData(
	ctx context.Context,
	poolAddress string,
	numCoins int,
) (StaticExtra, error) {
	req := d.ethrpcClient.NewRequest().SetContext(ctx)

	var (
		precisionMultipliers = make([]*big.Int, numCoins)
		rates                = make([]*big.Int, numCoins)
		lpToken              common.Address
	)

	for i := range numCoins {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodPrecisionMul,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&precisionMultipliers[i]})

		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodRates,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&rates[i]})
	}

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodToken,
		Params: nil,
	}, []any{&lpToken})

	if _, err := req.Aggregate(); err != nil {
		return StaticExtra{}, err
	}

	rateStrings := lo.Map(rates, func(item *big.Int, _ int) string {
		return item.String()
	})

	precisionStrings := lo.Map(precisionMultipliers, func(item *big.Int, _ int) string {
		return item.String()
	})

	var staticExtra = StaticExtra{
		LpToken:              lpToken.Hex(),
		PrecisionMultipliers: precisionStrings,
		Rates:                rateStrings,
		APrecision:           "1",
	}

	return staticExtra, nil
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadata := PoolsListUpdaterMetadata{
		Offset: newOffset,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}

func (u *PoolsListUpdater) getBatchSize(length int, limit int, offset int) int {
	if offset == length {
		return 0
	}

	if offset+limit >= length {
		if offset > length {
			logger.WithFields(logger.Fields{
				"dex":    u.config.DexID,
				"offset": offset,
				"length": length,
			}).Warn("[getBatchSize] offset is greater than length")
		}
		return max(length-offset, 0)
	}

	return limit
}
