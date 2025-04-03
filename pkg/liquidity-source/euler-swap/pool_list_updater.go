package eulerswap

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
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
	var (
		dexID     = u.config.DexID
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	ctx = util.NewContextWithTimestamp(ctx)

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

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      dexID,
				"pools_len":   len(pools),
				"offset":      offset,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getAllPoolsLength(ctx context.Context) (int, error) {
	var allPoolsLength *big.Int

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodAllPoolsLength,
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

	var metadata uniswapv2.PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}

	return metadata.Offset, nil
}

func (u *PoolsListUpdater) listPoolAddresses(ctx context.Context, offset int, batchSize int) ([]common.Address, error) {
	result := make([]common.Address, batchSize)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i := range batchSize {
		index := big.NewInt(int64(offset + i))

		req.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryMethodAllPools,
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
	token0List, token1List, err := u.listPoolTokens(ctx, poolAddresses)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolAddresses))

	for i, poolAddress := range poolAddresses {
		staticPoolData, err := u.getPoolStaticData(ctx, poolAddress.Hex())
		if err != nil {
			return nil, err
		}

		extraBytes, err := json.Marshal(&staticPoolData)
		if err != nil {
			return nil, err
		}

		token0 := &entity.PoolToken{
			Address:   strings.ToLower(token0List[i].Hex()),
			Swappable: true,
		}

		token1 := &entity.PoolToken{
			Address:   strings.ToLower(token1List[i].Hex()),
			Swappable: true,
		}

		var newPool = entity.Pool{
			Address:     strings.ToLower(poolAddress.Hex()),
			Exchange:    u.config.DexID,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{"0", "0"},
			Tokens:      []*entity.PoolToken{token0, token1},
			StaticExtra: string(extraBytes),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}

func (u *PoolsListUpdater) listPoolTokens(ctx context.Context, poolAddresses []common.Address) ([]common.Address, []common.Address, error) {
	var (
		listToken0Result = make([]common.Address, len(poolAddresses))
		listToken1Result = make([]common.Address, len(poolAddresses))
	)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i, poolAddress := range poolAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress.Hex(),
			Method: poolMethodAsset0,
			Params: nil,
		}, []any{&listToken0Result[i]})

		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress.Hex(),
			Method: poolMethodAsset1,
			Params: nil,
		}, []any{&listToken1Result[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		return nil, nil, err
	}

	return listToken0Result, listToken1Result, nil
}

func (d *PoolsListUpdater) getPoolStaticData(
	ctx context.Context,
	poolAddress string,
) (StaticExtra, error) {
	req := d.ethrpcClient.NewRequest().SetContext(ctx)

	var (
		eulerAccount        common.Address
		vault0              common.Address
		vault1              common.Address
		equilibriumReserve0 *big.Int
		equilibriumReserve1 *big.Int
		priceX              *big.Int
		priceY              *big.Int
		concentrationX      *big.Int
		concentrationY      *big.Int
		feeMultiplier       *big.Int
	)

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodEulerAccount,
		Params: nil,
	}, []any{&eulerAccount})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodVault0,
		Params: nil,
	}, []any{&vault0})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodVault1,
		Params: nil,
	}, []any{&vault1})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodVault0,
		Params: nil,
	}, []any{&vault0})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodEquilibriumReserve0,
		Params: nil,
	}, []any{&equilibriumReserve0})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodEquilibriumReserve1,
		Params: nil,
	}, []any{&equilibriumReserve1})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodPriceX,
		Params: nil,
	}, []any{&priceX})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodPriceY,
		Params: nil,
	}, []any{&priceY})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodConcentrationX,
		Params: nil,
	}, []any{&concentrationX})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodConcentrationY,
		Params: nil,
	}, []any{&concentrationY})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodFeeMultiplier,
		Params: nil,
	}, []any{&feeMultiplier})

	_, err := req.Aggregate()
	if err != nil {
		return StaticExtra{}, err
	}

	poolData := StaticExtra{
		Vault0:              vault0.Hex(),
		Vault1:              vault1.Hex(),
		EulerAccount:        eulerAccount.Hex(),
		FeeMultiplier:       uint256.MustFromBig(feeMultiplier),
		EquilibriumReserve0: uint256.MustFromBig(equilibriumReserve0),
		EquilibriumReserve1: uint256.MustFromBig(equilibriumReserve1),
		PriceX:              uint256.MustFromBig(priceX),
		PriceY:              uint256.MustFromBig(priceY),
		ConcentrationX:      uint256.MustFromBig(concentrationX),
		ConcentrationY:      uint256.MustFromBig(concentrationY),
	}

	return poolData, nil
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadata := uniswapv2.PoolsListUpdaterMetadata{
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
