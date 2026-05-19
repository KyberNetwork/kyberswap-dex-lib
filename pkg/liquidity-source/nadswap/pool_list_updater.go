package nadswap

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, c *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: c}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	start := time.Now()
	dexID := u.config.DexID

	total, err := u.getAllPairsLength(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Error("getAllPairsLength failed")
		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": dexID, "err": err}).Warn("getOffset failed")
	}

	batchSize := u.getBatchSize(total, u.config.NewPoolLimit, offset)
	if batchSize == 0 {
		return nil, metadataBytes, nil
	}

	pairs, err := u.listPairAddresses(ctx, offset, batchSize)
	if err != nil {
		return nil, metadataBytes, err
	}

	pools, err := u.initPools(ctx, pairs)
	if err != nil {
		return nil, metadataBytes, err
	}

	newMeta, err := json.Marshal(PoolsListUpdaterMetadata{Offset: offset + batchSize})
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.WithFields(logger.Fields{
		"dex_id": dexID, "pools_len": len(pools), "offset": offset, "duration_ms": time.Since(start).Milliseconds(),
	}).Info("Finished getting new pools")

	return pools, newMeta, nil
}

func (u *PoolsListUpdater) getAllPairsLength(ctx context.Context) (int, error) {
	var length *big.Int
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodAllPairsLength,
	}, []any{&length})
	if _, err := req.Call(); err != nil {
		return 0, err
	}
	return int(length.Int64()), nil
}

func (u *PoolsListUpdater) getOffset(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	var m PoolsListUpdaterMetadata
	if err := json.Unmarshal(b, &m); err != nil {
		return 0, err
	}
	return m.Offset, nil
}

func (u *PoolsListUpdater) getBatchSize(length, limit, offset int) int {
	if offset >= length {
		return 0
	}
	if offset+limit >= length {
		return length - offset
	}
	return limit
}

func (u *PoolsListUpdater) listPairAddresses(ctx context.Context, offset, batchSize int) ([]common.Address, error) {
	results := make([]common.Address, batchSize)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := 0; i < batchSize; i++ {
		req.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryMethodAllPairs,
			Params: []any{big.NewInt(int64(offset + i))},
		}, []any{&results[i]})
	}
	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}
	pairs := make([]common.Address, 0, batchSize)
	for i, ok := range resp.Result {
		if ok {
			pairs = append(pairs, results[i])
		}
	}
	return pairs, nil
}

// pairInfo holds per-pair multicall results.
type pairInfo struct {
	Token0     common.Address
	Token1     common.Address
	Reserve0   *big.Int
	Reserve1   *big.Int
	Timestamp  uint32
	HasFeeCfg  bool
	QuoteToken common.Address
	CreatorFee uint16
	DexProtFee uint16
}

func (u *PoolsListUpdater) initPools(ctx context.Context, pairs []common.Address) ([]entity.Pool, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	infos := make([]pairInfo, len(pairs))
	reservesResults := make([]reservesRPCResult, len(pairs))

	var feeCollector common.Address

	// 1) token0 / token1 / getReserves (all required; use Aggregate)
	reqA := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodFeeCollector,
	}, []any{&feeCollector})
	for i, p := range pairs {
		reqA.AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Hex(), Method: pairMethodToken0,
		}, []any{&infos[i].Token0})
		reqA.AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Hex(), Method: pairMethodToken1,
		}, []any{&infos[i].Token1})
		reqA.AddCall(&ethrpc.Call{
			ABI: pairABI, Target: p.Hex(), Method: pairMethodGetReserves,
		}, []any{&reservesResults[i]})
	}
	if _, err := reqA.Aggregate(); err != nil {
		return nil, err
	}
	for i := range pairs {
		infos[i].Reserve0 = reservesResults[i].Reserve0
		infos[i].Reserve1 = reservesResults[i].Reserve1
		infos[i].Timestamp = reservesResults[i].BlockTimestampLast
	}

	// 2) FeeCollector.getFeeConfig(pair) — revert-tolerant (TryAggregate)
	// NOTE: getFeeConfig returns a single tuple output. go-ethereum's ABI Copy uses
	// copyAtomic for single-output methods, which sets dst.Field(0). So we must wrap
	// our target struct in another struct (FeeCfg field at index 0).
	type feeCfgRaw struct {
		BaseToken            common.Address
		QuoteToken           common.Address
		CreatorFeeRate       uint16
		CurveProtocolFeeRate uint16
		DexProtocolFeeRate   uint16
	}
	type feeCfgWrapper struct {
		FeeCfg feeCfgRaw
	}
	rawCfg := make([]feeCfgWrapper, len(pairs))
	reqB := u.ethrpcClient.NewRequest().SetContext(ctx)
	feeCollectorStr := hexutil.Encode(feeCollector[:])
	for i, p := range pairs {
		reqB.AddCall(&ethrpc.Call{
			ABI:    feeCollectorABI,
			Target: feeCollectorStr,
			Method: feeCollectorMethodGetFeeConfig,
			Params: []any{p},
		}, []any{&rawCfg[i]})
	}
	respB, err := reqB.TryAggregate()
	if err != nil {
		return nil, err
	}
	for i, ok := range respB.Result {
		if ok {
			infos[i].HasFeeCfg = true
			infos[i].QuoteToken = rawCfg[i].FeeCfg.QuoteToken
			infos[i].CreatorFee = rawCfg[i].FeeCfg.CreatorFeeRate
			infos[i].DexProtFee = rawCfg[i].FeeCfg.DexProtocolFeeRate
		}
	}

	pools := make([]entity.Pool, 0, len(pairs))
	now := time.Now().Unix()
	for i, p := range pairs {
		r0u, _ := uint256.FromBig(infos[i].Reserve0)
		r1u, _ := uint256.FromBig(infos[i].Reserve1)
		extra := Extra{Reserve0: r0u, Reserve1: r1u, BlockTimestampLast: infos[i].Timestamp}
		se := StaticExtra{
			IsMemePair:         infos[i].HasFeeCfg,
			QuoteToken:         infos[i].QuoteToken,
			CreatorFeeRate:     infos[i].CreatorFee,
			DexProtocolFeeRate: infos[i].DexProtFee,
		}
		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return nil, err
		}
		seBytes, err := json.Marshal(se)
		if err != nil {
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(p[:]),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves: []string{
				infos[i].Reserve0.String(),
				infos[i].Reserve1.String(),
			},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(infos[i].Token0[:]), Swappable: true},
				{Address: hexutil.Encode(infos[i].Token1[:]), Swappable: true},
			},
			Extra:       string(extraBytes),
			StaticExtra: string(seBytes),
		})
	}

	return pools, nil
}
