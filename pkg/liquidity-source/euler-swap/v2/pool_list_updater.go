package v2

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type (
	PoolsListUpdater struct {
		config       *shared.Config
		ethrpcClient *ethrpc.Client
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
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

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadata []byte) ([]entity.Pool, []byte, error) {
	var offset int
	if len(metadata) > 0 {
		var m PoolsListUpdaterMetadata
		if err := json.Unmarshal(metadata, &m); err != nil {
			return nil, nil, err
		}
		offset = m.Offset
	}

	length, err := u.getPoolsLength(ctx)
	if err != nil {
		return nil, nil, err
	}

	if offset >= length {
		return nil, metadata, nil
	}

	batchSizeToUse := shared.BatchSize
	if offset+batchSizeToUse > length {
		batchSizeToUse = length - offset
	}

	poolAddresses, err := u.listPoolAddresses(ctx, offset, batchSizeToUse)
	if err != nil {
		return nil, nil, err
	}

	pools, err := u.initPools(ctx, poolAddresses)
	if err != nil {
		return nil, nil, err
	}

	newMetadata, err := json.Marshal(PoolsListUpdaterMetadata{
		Offset: offset + len(pools),
	})
	if err != nil {
		return nil, nil, err
	}

	return pools, newMetadata, nil
}

func (u *PoolsListUpdater) getPoolsLength(ctx context.Context) (int, error) {
	var length *big.Int
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    registryABI,
		Target: u.config.FactoryAddress,
		Method: shared.FactoryMethodPoolsLength,
	}, []any{&length})

	if _, err := req.Call(); err != nil {
		return 0, err
	}

	return int(length.Int64()), nil
}

func (u *PoolsListUpdater) listPoolAddresses(ctx context.Context, offset, limit int) ([]common.Address, error) {
	var poolAddresses []common.Address
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    registryABI,
		Target: u.config.FactoryAddress,
		Method: shared.FactoryMethodPoolsSlice,
		Params: []any{big.NewInt(int64(offset)), big.NewInt(int64(limit))},
	}, []any{&poolAddresses})

	if _, err := req.Call(); err != nil {
		return nil, err
	}

	return poolAddresses, nil
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

func (u *PoolsListUpdater) initPools(ctx context.Context, poolAddresses []common.Address) ([]entity.Pool, error) {
	tokensByPool, err := u.listPoolTokens(ctx, poolAddresses)
	if err != nil {
		return nil, err
	}

	numPools := len(poolAddresses)
	staticParams := make([]StaticParamsRPC, numPools)
	dynamicParams := make([]DynamicParamsRPC, numPools)
	evcAddresses := make([]common.Address, numPools)

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i, poolAddress := range poolAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress.Hex(),
			Method: shared.PoolMethodGetStaticParams,
		}, []any{&staticParams[i]})

		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress.Hex(),
			Method: shared.PoolMethodGetDynamicParams,
		}, []any{&dynamicParams[i]})

		req.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress.Hex(),
			Method: shared.PoolMethodEVC,
		}, []any{&evcAddresses[i]})
	}

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, numPools)

	for i, poolAddress := range poolAddresses {
		staticExtra := StaticExtra{
			SupplyVault0: staticParams[i].Data.SupplyVault0.Hex(),
			SupplyVault1: staticParams[i].Data.SupplyVault1.Hex(),
			EulerAccount: staticParams[i].Data.EulerAccount.Hex(),
			EVC:          evcAddresses[i].Hex(),
		}

		if !valueobject.IsZeroAddress(staticParams[i].Data.BorrowVault0) {
			staticExtra.BorrowVault0 = staticParams[i].Data.BorrowVault0.Hex()
		}
		if !valueobject.IsZeroAddress(staticParams[i].Data.BorrowVault1) {
			staticExtra.BorrowVault1 = staticParams[i].Data.BorrowVault1.Hex()
		}
		if !valueobject.IsZeroAddress(staticParams[i].Data.FeeRecipient) {
			staticExtra.FeeRecipient = staticParams[i].Data.FeeRecipient.Hex()
		}

		staticExtraBytes, err := json.Marshal(&staticExtra)
		if err != nil {
			return nil, err
		}

		dParams := DynamicParams{
			EquilibriumReserve0: uint256.MustFromBig(dynamicParams[i].Data.EquilibriumReserve0),
			EquilibriumReserve1: uint256.MustFromBig(dynamicParams[i].Data.EquilibriumReserve1),
			MinReserve0:         uint256.MustFromBig(dynamicParams[i].Data.MinReserve0),
			MinReserve1:         uint256.MustFromBig(dynamicParams[i].Data.MinReserve1),
			PriceX:              uint256.MustFromBig(dynamicParams[i].Data.PriceX),
			PriceY:              uint256.MustFromBig(dynamicParams[i].Data.PriceY),
			ConcentrationX:      uint256.NewInt(dynamicParams[i].Data.ConcentrationX),
			ConcentrationY:      uint256.NewInt(dynamicParams[i].Data.ConcentrationY),
			Fee0:                uint256.NewInt(dynamicParams[i].Data.Fee0),
			Fee1:                uint256.NewInt(dynamicParams[i].Data.Fee1),
			Expiration:          dynamicParams[i].Data.Expiration.Uint64(),
			SwapHookedOps:       dynamicParams[i].Data.SwapHookedOperations,
			SwapHook:            dynamicParams[i].Data.SwapHook.Hex(),
		}

		extra := Extra{
			Pause:         1, // unlocked
			DynamicParams: dParams,
		}

		extraBytes, err := json.Marshal(&extra)
		if err != nil {
			return nil, err
		}

		var tokens []*entity.PoolToken
		tokens = append(tokens, &entity.PoolToken{
			Address:   strings.ToLower(tokensByPool[i][0].Hex()),
			Swappable: true,
		})
		tokens = append(tokens, &entity.PoolToken{
			Address:   strings.ToLower(tokensByPool[i][1].Hex()),
			Swappable: true,
		})

		newPool := entity.Pool{
			Address:     strings.ToLower(poolAddress.Hex()),
			Exchange:    u.config.DexID,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      tokens,
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
