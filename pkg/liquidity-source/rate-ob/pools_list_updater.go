package rateob

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolListUpdater {
	return &PoolListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools lists the engine's pairs from its OrderbookFactory (allPairsLength / allPairs), each
// pair being one Orderbook contract. Pairs are directional: A/B and B/A are separate books.
func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	var factory, weth common.Address
	var nativeScale, length *big.Int
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: u.config.Engine, Method: "orderbookFactory"}, []any{&factory}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: u.config.Engine, Method: "WETH"}, []any{&weth}).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: u.config.Engine, Method: "nativeScale"}, []any{&nativeScale}).
		Aggregate(); err != nil {
		return nil, metadataBytes, err
	}
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: rateABI, Target: factory.Hex(), Method: "allPairsLength"}, []any{&length}).
		Call(); err != nil {
		return nil, metadataBytes, err
	}
	total := int(length.Int64())
	if total <= metadata.Offset {
		return nil, metadataBytes, nil
	}

	end := min(total, metadata.Offset+pairBatchSize)
	pairs := make([]common.Address, end-metadata.Offset)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := range pairs {
		req.AddCall(&ethrpc.Call{ABI: rateABI, Target: factory.Hex(), Method: "allPairs",
			Params: []any{big.NewInt(int64(metadata.Offset + i))}}, []any{&pairs[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, metadataBytes, err
	}

	type pairInfo struct {
		BaseQuote struct {
			Base  common.Address
			Quote common.Address
		}
		Probe *big.Int
	}
	infos := make([]pairInfo, len(pairs))
	req = u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, pair := range pairs {
		req.AddCall(&ethrpc.Call{ABI: rateABI, Target: pair.Hex(), Method: "getBaseQuote"},
			[]any{&infos[i].BaseQuote}).
			// convert(1e8, 1e18, base->quote) recovers Orderbook's private decDiff and its direction,
			// as BandPool.initialize does.
			AddCall(&ethrpc.Call{ABI: rateABI, Target: pair.Hex(), Method: "convert",
				Params: []any{big.NewInt(1e8), big.NewInt(1e18), true}}, []any{&infos[i].Probe})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, metadataBytes, err
	}

	unwraps := nativeScale.Sign() == 0
	wethHex := hexutil.Encode(weth[:])
	pools := make([]entity.Pool, 0, len(pairs))
	for i, pair := range pairs {
		decDiff, baseBQuote, err := decDiffOf(infos[i].Probe)
		if err != nil {
			log.Warn().Str("dexID", u.config.DexID).Str("pair", pair.Hex()).Err(err).Msg("skip pair")
			continue
		}
		base, quote := hexutil.Encode(infos[i].BaseQuote.Base[:]), hexutil.Encode(infos[i].BaseQuote.Quote[:])
		staticExtra := StaticExtra{Engine: strings.ToLower(u.config.Engine), DecDiff: decDiff, BaseBQuote: baseBQuote}
		if unwraps && (base == wethHex || quote == wethHex) {
			staticExtra.NativeOut = wethHex
		}
		staticExtraBytes, _ := json.Marshal(staticExtra)
		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(pair[:]),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: base, Swappable: true},
				{Address: quote, Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}
	log.Info().Str("dexID", u.config.DexID).Int("count", len(pools)).Int("total", total).Msg("fetched new pairs")

	metadata.Offset = end
	metadataBytes, _ = json.Marshal(metadata)
	return pools, metadataBytes, nil
}

// decDiffOf reads convert(1e8, 1e18, true): 1e18/decDiff when base has more decimals, 1e18*decDiff
// otherwise (decDiff 1 when they are equal).
func decDiffOf(probe *big.Int) (uint64, bool, error) {
	e18 := big.NewInt(1e18)
	var q, r big.Int
	switch probe.Cmp(e18) {
	case 0:
		return 1, false, nil
	case 1:
		q.QuoRem(probe, e18, &r)
		if r.Sign() != 0 || !q.IsUint64() {
			return 0, false, errors.Errorf("unexpected convert probe %s", probe)
		}
		return q.Uint64(), false, nil
	default:
		if probe.Sign() == 0 {
			return 0, false, errors.Errorf("unexpected convert probe %s", probe)
		}
		q.QuoRem(e18, probe, &r)
		if r.Sign() != 0 {
			return 0, false, errors.Errorf("unexpected convert probe %s", probe)
		}
		return q.Uint64(), true, nil
	}
}
