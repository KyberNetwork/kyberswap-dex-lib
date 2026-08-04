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

	pools := make([]entity.Pool, 0, len(newPoolAddrs))
	for _, poolAddr := range newPoolAddrs {
		p, err := u.newPool(ctx, poolAddr)
		if err != nil {
			log.Errorf("metronome-swap: build pool %s failed: %v", poolAddr, err)
			return nil, metadataBytes, err
		}
		pools = append(pools, p)
	}

	newMeta, err := json.Marshal(PoolsListUpdaterMetadata{Offset: len(poolAddrs)})
	if err != nil {
		return pools, metadataBytes, nil
	}

	return pools, newMeta, nil
}

// newPool resolves a Pool's swappable synthetic tokens. swap() takes ISyntheticToken
// instances, NOT the IDebtToken instances Pool.getDebtTokens() returns directly — each debt
// token maps 1:1 to its synthetic token via DebtToken.syntheticToken().
func (u *PoolsListUpdater) newPool(ctx context.Context, poolAddr common.Address) (entity.Pool, error) {
	poolStr := hexutil.Encode(poolAddr[:])

	var debtTokens []common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolStr,
		Method: poolMethodGetDebtTokens,
	}, []any{&debtTokens}).Call(); err != nil {
		return entity.Pool{}, err
	}

	syntheticTokens := make([]common.Address, len(debtTokens))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, debtToken := range debtTokens {
		req.AddCall(&ethrpc.Call{
			ABI:    debtTokenABI,
			Target: hexutil.Encode(debtToken[:]),
			Method: debtTokenMethodSyntheticToken,
		}, []any{&syntheticTokens[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return entity.Pool{}, err
	}

	// decimals() is widened to uint256 in synthetic_token.json (ABI hygiene rule) — decode
	// into *big.Int, not *uint8, to match the widened type.
	decimalsRaw := make([]*big.Int, len(syntheticTokens))
	req2 := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, synth := range syntheticTokens {
		req2.AddCall(&ethrpc.Call{
			ABI:    syntheticTokenABI,
			Target: hexutil.Encode(synth[:]),
			Method: syntheticTokenMethodDecimals,
		}, []any{&decimalsRaw[i]})
	}
	if _, err := req2.Aggregate(); err != nil {
		return entity.Pool{}, err
	}

	tokens := make([]*entity.PoolToken, len(syntheticTokens))
	reserves := make(entity.PoolReserves, len(syntheticTokens))
	for i, synth := range syntheticTokens {
		tokens[i] = &entity.PoolToken{
			Address:   strings.ToLower(hexutil.Encode(synth[:])),
			Decimals:  uint8(decimalsRaw[i].Uint64()),
			Swappable: true,
		}
		reserves[i] = "0"
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{PoolRegistry: strings.ToLower(u.cfg.PoolRegistry)})
	if err != nil {
		return entity.Pool{}, err
	}

	return entity.Pool{
		Address:     strings.ToLower(poolStr),
		Exchange:    u.cfg.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Tokens:      tokens,
		Reserves:    reserves,
		Extra:       "{}",
		StaticExtra: string(staticExtraBytes),
	}, nil
}
