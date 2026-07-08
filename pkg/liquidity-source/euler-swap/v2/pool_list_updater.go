package v2

import (
	"context"
	"math/big"
	"slices"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

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
	// addresses seen at the tail of the registry's pool list. The registry
	// stores pools in an OpenZeppelin EnumerableSet: unregistering a pool swaps
	// the last element into the removed slot, reshuffling indices, so a numeric
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
	var metadata PoolsListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, nil, err
		}
	}

	length, err := u.getPoolsLength(ctx)
	if err != nil {
		return nil, nil, err
	}

	if length == 0 {
		return nil, metadataBytes, nil
	}

	tailSize := min(shared.BackupTailWindow, length)
	tailAddresses, err := u.listPoolAddresses(ctx, length-tailSize, tailSize)
	if err != nil {
		return nil, nil, err
	}

	newAddresses := make([]common.Address, 0)
	for _, addr := range tailAddresses {
		if !slices.Contains(metadata.LastPools, addr) {
			newAddresses = append(newAddresses, addr)
		}
	}

	metadata.LastPools = tailAddresses

	if len(newAddresses) == 0 {
		newMetadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return nil, metadataBytes, err
		}
		return nil, newMetadataBytes, nil
	}

	pools, err := u.initPools(ctx, newAddresses)
	if err != nil {
		return nil, nil, err
	}

	newMetadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
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

func (u *PoolsListUpdater) listPoolAddresses(ctx context.Context, offset, count int) ([]common.Address, error) {
	var poolAddresses []common.Address
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    registryABI,
		Target: u.config.FactoryAddress,
		Method: shared.FactoryMethodPoolsSlice,
		Params: []any{big.NewInt(int64(offset)), big.NewInt(int64(offset + count))},
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
		staticExtra := buildStaticExtra(staticParams[i].Data, evcAddresses[i])

		staticExtraBytes, err := json.Marshal(&staticExtra)
		if err != nil {
			return nil, err
		}

		extra := Extra{
			Pause:         1, // unlocked
			DynamicParams: buildDynamicParams(dynamicParams[i].Data),
		}

		extraBytes, err := json.Marshal(&extra)
		if err != nil {
			return nil, err
		}

		var tokens []*entity.PoolToken
		tokens = append(tokens, &entity.PoolToken{
			Address:   hexutil.Encode(tokensByPool[i][0][:]),
			Swappable: true,
		})
		tokens = append(tokens, &entity.PoolToken{
			Address:   hexutil.Encode(tokensByPool[i][1][:]),
			Swappable: true,
		})

		newPool := entity.Pool{
			Address:     hexutil.Encode(poolAddress[:]),
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
