package umbraedlmm

import (
	"context"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

// feeParamsRPC mirrors the DEPLOYED FeeHelper.FeeParameters tuple (7 fields, no protocolShare).
type feeParamsRPC struct {
	BaseFactor               uint16
	FilterPeriod             uint16
	DecayPeriod              uint16
	ReductionFactor          uint16
	VariableFeeControl       uint16
	MaxVolatilityAccumulator *big.Int // uint24
	MinSwapBps               uint16
}

type quoteStateRPC struct {
	FeeParams             feeParamsRPC
	VolatilityAccumulator *big.Int
	VolatilityReference   *big.Int
	LastVolatilityUpdate  *big.Int
	ScaleX                *big.Int
	ScaleY                *big.Int
	Factory               common.Address
}

type pairStatsRPC struct {
	ActiveId            *big.Int
	ReserveX            *big.Int
	ReserveY            *big.Int
	ProtocolFeesX       *big.Int
	ProtocolFeesY       *big.Int
	Volatility          *big.Int
	LastUpdateTimestamp *big.Int
}

type activeBinsRPC struct {
	BinIds      []*big.Int
	ReservesX   []*big.Int
	ReservesY   []*big.Int
	TotalShares []*big.Int
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("getting new pool state")

	var static StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &static); err != nil {
		return p, err
	}

	// All reads pinned to one block. The deployed pair routes reserves/bins/quotes to an extension,
	// so reserves/bins come via the PairViewer; fee + volatility state come via getQuoteState.
	var (
		activeID  *big.Int
		quote     quoteStateRPC
		stats     pairStatsRPC
		activeBin activeBinsRPC
	)
	resp, err := t.ethrpcClient.R().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairMethodGetActiveID}, []any{&activeID}).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairMethodGetQuoteState}, []any{&quote}).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairMethodGetPairStatistics}, []any{&stats}).
		AddCall(&ethrpc.Call{ABI: viewerABI, Target: t.config.ViewerAddress, Method: viewerMethodActiveBins, Params: []any{common.HexToAddress(p.Address)}}, []any{&activeBin}).
		TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	blockNumber := resp.BlockNumber

	var variableFeeCap uint16
	if _, err := t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(blockNumber).AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: t.config.FactoryAddress,
		Method: factoryMethodGetVariableFeeCap,
		Params: []any{static.BinStep, quote.FeeParams.BaseFactor},
	}, []any{&variableFeeCap}).Call(); err != nil {
		// getVariableFeeCap is best-effort (0 = no cap), matching the deployed try/catch.
		variableFeeCap = 0
	}

	// Bins from the viewer are already normalized (18-decimal); use directly, drop empties.
	bins := make([]Bin, 0, len(activeBin.BinIds))
	for i, id := range activeBin.BinIds {
		rx, _ := uint256.FromBig(activeBin.ReservesX[i])
		ry, _ := uint256.FromBig(activeBin.ReservesY[i])
		if (rx == nil || rx.IsZero()) && (ry == nil || ry.IsZero()) {
			continue
		}
		bins = append(bins, Bin{ID: uint32(id.Uint64()), ReserveX: orZero(rx), ReserveY: orZero(ry)})
	}
	sort.Slice(bins, func(i, j int) bool { return bins[i].ID < bins[j].ID })

	// Decay the accumulator to "now" (≈ tracked block) exactly as _getDecayedVolatility does.
	startVol := decayedVolatility(quote, uint64(time.Now().Unix()))

	extra := Extra{
		ActiveID: uint32(activeID.Uint64()),
		Bins:     bins,
		FeeParameters: FeeParameters{
			BaseFactor:               quote.FeeParams.BaseFactor,
			FilterPeriod:             quote.FeeParams.FilterPeriod,
			DecayPeriod:              quote.FeeParams.DecayPeriod,
			ReductionFactor:          quote.FeeParams.ReductionFactor,
			VariableFeeControl:       quote.FeeParams.VariableFeeControl,
			MaxVolatilityAccumulator: uint32(quote.FeeParams.MaxVolatilityAccumulator.Uint64()),
			MinSwapBps:               quote.FeeParams.MinSwapBps,
		},
		VariableFeeCap:        variableFeeCap,
		VolatilityAccumulator: startVol,
		VolatilityReference:   uint32(quote.VolatilityReference.Uint64()),
		NativeReserveX:        stats.ReserveX.String(),
		NativeReserveY:        stats.ReserveY.String(),
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{stats.ReserveX.String(), stats.ReserveY.String()}
	p.BlockNumber = blockNumber.Uint64()
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{"pool_id": p.Address, "bins": len(bins), "block": p.BlockNumber}).Info("finished getting new pool state")
	return p, nil
}

// decayedVolatility mirrors the viewer's _getDecayedVolatility: constant during the filter period,
// linear decay after, to zero past decayPeriod.
func decayedVolatility(q quoteStateRPC, now uint64) uint64 {
	vol := q.VolatilityAccumulator.Uint64()
	last := q.LastVolatilityUpdate.Uint64()
	if now <= last {
		return vol
	}
	delta := now - last
	if delta < uint64(q.FeeParams.FilterPeriod) {
		return vol
	}
	return applyVolatilityDecay(vol, delta, uint64(q.FeeParams.DecayPeriod))
}
