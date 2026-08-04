package metronomeswap

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

// PoolsListUpdaterMetadata tracks how far into PoolRegistry.getPools()'s result we've already
// materialized entity.Pool records. getPools() returns the full current list every call (it's a
// registry, not a paginated event log), and pools are only ever appended by id — offset-based
// diffing is enough to pick up newly-registered pools without re-processing known ones.
type PoolsListUpdaterMetadata struct {
	Offset int `json:"offset"`
}

type PoolsListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{"dexId": u.cfg.DexID})

	var metadata PoolsListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			log.Warnf("metronome-swap: parse metadata failed: %v", err)
		}
	}

	var poolAddrs []common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    poolRegistryABI,
		Target: u.cfg.PoolRegistry,
		Method: poolRegistryMethodGetPools,
	}, []any{&poolAddrs}).Call(); err != nil {
		log.Errorf("metronome-swap: getPools failed: %v", err)
		return nil, metadataBytes, err
	}

	if metadata.Offset >= len(poolAddrs) {
		return nil, metadataBytes, nil
	}
	newPoolAddrs := poolAddrs[metadata.Offset:]

	pools, err := u.newPools(ctx, newPoolAddrs)
	if err != nil {
		log.Errorf("metronome-swap: build pools failed: %v", err)
		return nil, metadataBytes, err
	}

	newMeta, err := json.Marshal(PoolsListUpdaterMetadata{Offset: len(poolAddrs)})
	if err != nil {
		return pools, metadataBytes, nil
	}

	return pools, newMeta, nil
}

// newPools resolves swappable synthetic tokens for every pool address in one shot: 3 batched
// rpc roundtrips total (getDebtTokens, syntheticToken, decimals), regardless of how many pools
// are being built — NOT 3 roundtrips per pool. swap() takes ISyntheticToken instances, NOT the
// IDebtToken instances Pool.getDebtTokens() returns directly — each debt token maps 1:1 to its
// synthetic token via DebtToken.syntheticToken().
//
// A pool with fewer than 2 synthetic tokens has no swappable pair (CanSwapTo/CanSwapFrom are
// always empty for it) and is skipped — tracking it would burn an rpc-heavy refresh cycle every
// run for a pool that can never route a swap.
func (u *PoolsListUpdater) newPools(ctx context.Context, poolAddrs []common.Address) ([]entity.Pool, error) {
	debtTokensPerPool := make([][]common.Address, len(poolAddrs))
	reqDebtTokens := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, poolAddr := range poolAddrs {
		reqDebtTokens.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: hexutil.Encode(poolAddr[:]),
			Method: poolMethodGetDebtTokens,
		}, []any{&debtTokensPerPool[i]})
	}
	if _, err := reqDebtTokens.Aggregate(); err != nil {
		return nil, err
	}

	// debtTokenOffsets[i] is where pool i's debt tokens start in the flattened slices below.
	debtTokenOffsets := make([]int, len(poolAddrs))
	totalDebtTokens := 0
	for i, debtTokens := range debtTokensPerPool {
		debtTokenOffsets[i] = totalDebtTokens
		totalDebtTokens += len(debtTokens)
	}

	syntheticTokens := make([]common.Address, totalDebtTokens)
	reqSynthetic := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, debtTokens := range debtTokensPerPool {
		for j, debtToken := range debtTokens {
			idx := debtTokenOffsets[i] + j
			reqSynthetic.AddCall(&ethrpc.Call{
				ABI:    debtTokenABI,
				Target: hexutil.Encode(debtToken[:]),
				Method: debtTokenMethodSyntheticToken,
			}, []any{&syntheticTokens[idx]})
		}
	}
	if totalDebtTokens > 0 {
		if _, err := reqSynthetic.Aggregate(); err != nil {
			return nil, err
		}
	}

	// decimals() is widened to uint256 in synthetic_token.json (ABI hygiene rule) — decode
	// into *big.Int, not *uint8, to match the widened type.
	decimalsRaw := make([]*big.Int, totalDebtTokens)
	reqDecimals := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, synth := range syntheticTokens {
		reqDecimals.AddCall(&ethrpc.Call{
			ABI:    syntheticTokenABI,
			Target: hexutil.Encode(synth[:]),
			Method: syntheticTokenMethodDecimals,
		}, []any{&decimalsRaw[i]})
	}
	if totalDebtTokens > 0 {
		if _, err := reqDecimals.Aggregate(); err != nil {
			return nil, err
		}
	}

	poolRegistry := strings.ToLower(u.cfg.PoolRegistry)
	staticExtraBytes, err := json.Marshal(StaticExtra{PoolRegistry: poolRegistry})
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolAddrs))
	for i, poolAddr := range poolAddrs {
		nTokens := len(debtTokensPerPool[i])
		if nTokens < 2 {
			continue // no swappable pair — CanSwapTo/CanSwapFrom would always be empty
		}

		base := debtTokenOffsets[i]
		tokens := make([]*entity.PoolToken, nTokens)
		reserves := make(entity.PoolReserves, nTokens)
		for j := range nTokens {
			tokens[j] = &entity.PoolToken{
				Address:   strings.ToLower(hexutil.Encode(syntheticTokens[base+j][:])),
				Decimals:  uint8(decimalsRaw[base+j].Uint64()),
				Swappable: true,
			}
			reserves[j] = "0"
		}

		pools = append(pools, entity.Pool{
			Address:     strings.ToLower(hexutil.Encode(poolAddr[:])),
			Exchange:    u.cfg.DexID,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Tokens:      tokens,
			Reserves:    reserves,
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}

	return pools, nil
}
