package rangepool

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	// PoolsListUpdaterMetadata persists how many pools we have already emitted, so
	// each GetNewPools call only initializes pools the factory has added since.
	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools discovers Range Pools on-chain from RangePoolFactory.getPools() (Range
// is not indexed by any subgraph — custom, non-canonical Vault). It reads
// getRangePoolImmutableData for each newly seen pool to build its tokens + StaticExtra,
// and advances the metadata offset so subsequent calls are incremental.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	startTime := time.Now()
	dexID := u.config.DexID

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	allPools, err := u.listPoolAddresses(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("getPools failed")
		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Warn("getOffset failed")
	}
	if offset > len(allPools) {
		offset = len(allPools)
	}

	batch := allPools[offset:]
	if limit := u.config.NewPoolLimit; limit > 0 && len(batch) > limit {
		batch = batch[:limit]
	}

	pools, err := u.initPools(ctx, batch)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("initPools failed")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := json.Marshal(PoolsListUpdaterMetadata{Offset: offset + len(batch)})
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.WithFields(logger.Fields{
		"dex_id":      dexID,
		"pools_len":   len(pools),
		"offset":      offset,
		"total":       len(allPools),
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) listPoolAddresses(ctx context.Context) ([]common.Address, error) {
	var pools []common.Address
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    rangeFactoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodGetPools,
	}, []any{&pools})
	if _, err := req.Call(); err != nil {
		return nil, err
	}
	return pools, nil
}

func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}
	var m PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &m); err != nil {
		return 0, err
	}
	return m.Offset, nil
}

// initPools reads getRangePoolImmutableData for each pool (tokens, scaling factors,
// normalized weights) and assembles the entity.Pool. Dynamic state (balances, fees,
// virtual balances, liveness) is left for the tracker; reserves start at "0" so the
// pool is treated as paused until first tracked.
func (u *PoolsListUpdater) initPools(ctx context.Context, addrs []common.Address) ([]entity.Pool, error) {
	if len(addrs) == 0 {
		return nil, nil
	}

	immutables := make([]rangePoolImmutableDataResult, len(addrs))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, addr := range addrs {
		req.AddCall(&ethrpc.Call{
			ABI:    rangePoolABI,
			Target: addr.Hex(),
			Method: poolMethodGetImmutableData,
		}, []any{&immutables[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	// Empty Extra placeholder (the tracker overwrites it with real state).
	emptyExtraBytes, err := json.Marshal(Extra{Extra: &shared.Extra{}})
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(addrs))
	for i, addr := range addrs {
		imm := immutables[i].Data

		tokens := make([]*entity.PoolToken, len(imm.Tokens))
		reserves := make(entity.PoolReserves, len(imm.Tokens))
		for j, t := range imm.Tokens {
			tokens[j] = &entity.PoolToken{Address: hexutil.Encode(t[:]), Swappable: true}
			reserves[j] = "0"
		}

		staticExtra := StaticExtra{
			StaticExtra: shared.StaticExtra{
				Hook: hexutil.Encode(HookAddress[:]),
				// Range has no ERC4626 buffers, but base.PoolSimulator.GetMetaInfo indexes
				// bufferTokens[tokenIndex], so it must be a len(tokens) slice of empty
				// strings (not nil) or GetMetaInfo panics.
				BufferTokens: make([]string, len(imm.Tokens)),
			},
			NormalizedWeights:     shared.FromBigs(imm.NormalizedWeights),
			DecimalScalingFactors: shared.FromBigs(imm.DecimalScalingFactors),
			FactoryAddress:        u.config.FactoryAddress,
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:     hexutil.Encode(addr[:]),
			Exchange:    u.config.DexID,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			Extra:       string(emptyExtraBytes),
			StaticExtra: string(staticExtraBytes),
		})
	}
	return pools, nil
}
