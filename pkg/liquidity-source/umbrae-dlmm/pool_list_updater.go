package umbraedlmm

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type PoolsListUpdaterMetadata struct {
	Offset int `json:"offset"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	startTime := time.Now()
	logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Info("Started getting new pools")

	total, err := u.allPairsLength(ctx)
	if err != nil {
		return nil, metadataBytes, err
	}

	offset := u.getOffset(metadataBytes)
	limit := u.config.NewPoolLimit
	if limit <= 0 || offset+limit > total {
		limit = total - offset
	}
	if limit <= 0 {
		return nil, metadataBytes, nil
	}

	addresses, err := u.listPairAddresses(ctx, offset, limit)
	if err != nil {
		return nil, metadataBytes, err
	}

	pools, err := u.initPools(ctx, addresses)
	if err != nil {
		return nil, metadataBytes, err
	}

	newMeta, err := json.Marshal(PoolsListUpdaterMetadata{Offset: offset + limit})
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.WithFields(logger.Fields{
		"dex_id": u.config.DexID, "pools_len": len(pools), "offset": offset,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Finished getting new pools")

	return pools, newMeta, nil
}

func (u *PoolsListUpdater) allPairsLength(ctx context.Context) (int, error) {
	var length *big.Int
	if _, err := u.ethrpcClient.R().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodAllPairsLength,
	}, []any{&length}).Call(); err != nil {
		return 0, err
	}
	return int(length.Int64()), nil
}

func (u *PoolsListUpdater) getOffset(metadataBytes []byte) int {
	if len(metadataBytes) == 0 {
		return 0
	}
	var m PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &m); err != nil {
		return 0
	}
	return m.Offset
}

func (u *PoolsListUpdater) listPairAddresses(ctx context.Context, offset, limit int) ([]common.Address, error) {
	out := make([]common.Address, limit)
	req := u.ethrpcClient.R().SetContext(ctx)
	for i := 0; i < limit; i++ {
		req.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryMethodAllPairs,
			Params: []any{big.NewInt(int64(offset + i))},
		}, []any{&out[i]})
	}
	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}
	var addresses []common.Address
	for i, ok := range resp.Result {
		if ok {
			addresses = append(addresses, out[i])
		}
	}
	return addresses, nil
}

// initPools reads the immutable pair config (tokens, binStep, decimals) into StaticExtra. Reserves
// are left at 0; the tracker fills bins and reserves later.
func (u *PoolsListUpdater) initPools(ctx context.Context, addresses []common.Address) ([]entity.Pool, error) {
	type pairInfo struct {
		tokenX, tokenY common.Address
		binStep        uint16
		dec            decimalsResult
	}
	infos := make([]pairInfo, len(addresses))

	req := u.ethrpcClient.R().SetContext(ctx)
	for i, addr := range addresses {
		target := addr.Hex()
		req.AddCall(&ethrpc.Call{ABI: pairABI, Target: target, Method: pairMethodTokenX}, []any{&infos[i].tokenX}).
			AddCall(&ethrpc.Call{ABI: pairABI, Target: target, Method: pairMethodTokenY}, []any{&infos[i].tokenY}).
			AddCall(&ethrpc.Call{ABI: pairABI, Target: target, Method: pairMethodBinStep}, []any{&infos[i].binStep}).
			AddCall(&ethrpc.Call{ABI: pairABI, Target: target, Method: pairMethodGetDecimals}, []any{&infos[i].dec})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(addresses))
	for i, addr := range addresses {
		staticExtra, err := json.Marshal(StaticExtra{
			BinStep:   infos[i].binStep,
			DecimalsX: infos[i].dec.DecimalsX,
			DecimalsY: infos[i].dec.DecimalsY,
			Router:    u.config.RouterAddress,
		})
		if err != nil {
			return nil, err
		}
		pools = append(pools, entity.Pool{
			Address:   strings.ToLower(addr.Hex()),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(infos[i].tokenX.Hex()), Swappable: true},
				{Address: strings.ToLower(infos[i].tokenY.Hex()), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtra),
		})
	}
	return pools, nil
}
