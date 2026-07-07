package v1

import (
	"context"
	"math/big"
	"slices"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		config       *shared.Config
		ethrpcClient *ethrpc.Client
	}

	// PoolsListUpdaterMetadata tracks the last shared.BackupTailWindow pool
	// addresses seen at the tail of the factory's pool list. The factory stores
	// pools in an OpenZeppelin EnumerableSet: unregistering a pool swaps the
	// last element into the removed slot, reshuffling indices, so a numeric
	// offset can't be trusted to page through the list safely. This is only a
	// backup for the primary PoolFactory block-subscription flow though, so it
	// just needs to notice pools appended since the last check, not replay the
	// whole history.
	PoolsListUpdaterMetadata struct {
		LastPools []common.Address `json:"lp"`
	}
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *shared.Config,
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

	metadata, err := u.getMetadata(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Warn("getMetadata failed")
	}

	allPoolsLength, err := u.getAllPoolsLength(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID}).
			Error("allPoolsLength failed")

		return nil, metadataBytes, err
	}

	if allPoolsLength == 0 {
		logger.
			WithFields(logger.Fields{"dex_id": dexID}).
			Warn("no pools found")

		return nil, metadataBytes, nil
	}

	tailSize := min(shared.BackupTailWindow, allPoolsLength)
	tailAddresses, err := u.listPoolAddresses(ctx, allPoolsLength-tailSize, tailSize)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("listPoolAddresses failed")

		return nil, metadataBytes, err
	}

	newAddresses := make([]common.Address, 0)
	for _, addr := range tailAddresses {
		if !slices.Contains(metadata.LastPools, addr) {
			newAddresses = append(newAddresses, addr)
		}
	}

	metadata.LastPools = tailAddresses

	if len(newAddresses) == 0 {
		newMetadataBytes, err := u.newMetadata(metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
		return nil, newMetadataBytes, nil
	}

	pools, err := u.initPools(ctx, newAddresses)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initPools failed")

		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(metadata)
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
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) listPoolAddresses(ctx context.Context, offset, batchSize int) ([]common.Address, error) {
	result := []common.Address{}

	startIdx := big.NewInt(int64(offset))
	endIdx := big.NewInt(int64(offset + batchSize))

	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: "poolsSlice", // Still hardcoded if not in shared, check if it should be
		Params: []any{startIdx, endIdx},
	}, []any{&result})

	_, err := req.Aggregate()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	tokensByPool, err := u.listPoolTokens(ctx, poolAddresses)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolAddresses))

	for i, poolAddress := range poolAddresses {
		staticPoolData, err := getPoolStaticData(ctx, u.ethrpcClient, poolAddress.Hex())
		if err != nil {
			return nil, err
		}

		extraBytes, err := json.Marshal(&staticPoolData)
		if err != nil {
			return nil, err
		}

		token0 := &entity.PoolToken{
			Address:   hexutil.Encode(tokensByPool[i][0][:]),
			Swappable: true,
		}

		token1 := &entity.PoolToken{
			Address:   hexutil.Encode(tokensByPool[i][1][:]),
			Swappable: true,
		}

		var newPool = entity.Pool{
			Address:     hexutil.Encode(poolAddress[:]),
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

func (u *PoolsListUpdater) listPoolTokens(ctx context.Context, poolAddresses []common.Address) ([][2]common.Address, error) {
	var poolTokens = make([][2]common.Address, len(poolAddresses))

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i, poolAddress := range poolAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress.Hex(),
			Method: shared.PoolMethodGetAssets,
		}, []any{&poolTokens[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	return poolTokens, nil
}

// getPoolStaticData fetches a pool's immutable params directly, so it can be
// reused both by the batch backfill (PoolsListUpdater) and the per-pool decode
// on the PoolDeployed event (PoolFactory).
func getPoolStaticData(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	poolAddress string,
) (StaticExtra, error) {
	var (
		params ParamsRPC
		evc    common.Address
	)

	req := ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: shared.PoolMethodGetParams,
	}, []any{&params})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: shared.PoolMethodEVC,
	}, []any{&evc})

	_, err := req.Aggregate()
	if err != nil {
		return StaticExtra{}, err
	}

	poolData := StaticExtra{
		Vault0:               params.Data.Vault0.Hex(),
		Vault1:               params.Data.Vault1.Hex(),
		EulerAccount:         params.Data.EulerAccount.Hex(),
		EquilibriumReserve0:  uint256.MustFromBig(params.Data.EquilibriumReserve0),
		EquilibriumReserve1:  uint256.MustFromBig(params.Data.EquilibriumReserve1),
		PriceX:               uint256.MustFromBig(params.Data.PriceX),
		PriceY:               uint256.MustFromBig(params.Data.PriceY),
		Fee:                  uint256.MustFromBig(params.Data.Fee),
		ProtocolFee:          uint256.MustFromBig(params.Data.ProtocolFee),
		ConcentrationX:       uint256.MustFromBig(params.Data.ConcentrationX),
		ConcentrationY:       uint256.MustFromBig(params.Data.ConcentrationY),
		ProtocolFeeRecipient: params.Data.ProtocolFeeRecipient,
		EVC:                  evc.Hex(),
	}

	return poolData, nil
}

func (u *PoolsListUpdater) getAllPoolsLength(ctx context.Context) (int, error) {
	var allPoolsLength *big.Int

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: shared.FactoryMethodPoolsLength,
	}, []any{&allPoolsLength})

	if _, err := req.Call(); err != nil {
		return 0, err
	}

	return int(allPoolsLength.Int64()), nil
}

func (u *PoolsListUpdater) newMetadata(metadata PoolsListUpdaterMetadata) ([]byte, error) {
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}

func (u *PoolsListUpdater) getMetadata(metadataBytes []byte) (PoolsListUpdaterMetadata, error) {
	if len(metadataBytes) == 0 {
		return PoolsListUpdaterMetadata{}, nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return PoolsListUpdaterMetadata{}, err
	}

	return metadata, nil
}
