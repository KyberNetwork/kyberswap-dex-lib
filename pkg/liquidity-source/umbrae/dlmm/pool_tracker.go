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

// feeParamsRPC mirrors the deployed FeeHelper.FeeParameters tuple (7 fields, no protocolShare).
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
	VolatilityAccumulator *big.Int // uint128
	VolatilityReference   *big.Int // uint24 — the anchor bin id in V2
	LastVolatilityUpdate  *big.Int // uint40
	ScaleX                *big.Int // 10^(18-decimalsX)
	ScaleY                *big.Int // 10^(18-decimalsY)
	Factory               common.Address
}

type reservesRPC struct {
	ReserveX *big.Int
	ReserveY *big.Int
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

	// All reads pinned to one block. Fee + volatility state come via getQuoteState; native total
	// reserves via getReserves; per-bin reserves via the PairViewer.
	var (
		activeID  *big.Int
		quote     quoteStateRPC
		reserves  reservesRPC
		activeBin activeBinsRPC
	)
	resp, err := t.ethrpcClient.R().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairMethodGetActiveID}, []any{&activeID}).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairMethodGetQuoteState}, []any{&quote}).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: p.Address, Method: pairMethodGetReserves}, []any{&reserves}).
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
		// Best-effort fallback, mirroring the viewer's try/catch. NOTE: in V2 a zero cap pins the
		// variable fee to zero (#147), so log it — a silent 0 here under-quotes the fee.
		logger.WithFields(logger.Fields{"pool_id": p.Address, "error": err}).
			Warn("getVariableFeeCap failed; defaulting to 0 (variable fee pinned to zero)")
		variableFeeCap = 0
	}

	// The V2 viewer reports bin reserves in NATIVE decimals (the V1 viewer reported 18-decimal
	// normalized — this flipped in the redeploy). The swap math runs on normalized values, and bin
	// reserves only ever change by whole multiples of the scale factors, so scaling back up is
	// exact. Drop empty bins.
	scaleX, _ := uint256.FromBig(quote.ScaleX)
	scaleY, _ := uint256.FromBig(quote.ScaleY)
	bins := make([]Bin, 0, len(activeBin.BinIds))
	for i, id := range activeBin.BinIds {
		rx, _ := uint256.FromBig(activeBin.ReservesX[i])
		ry, _ := uint256.FromBig(activeBin.ReservesY[i])
		if (rx == nil || rx.IsZero()) && (ry == nil || ry.IsZero()) {
			continue
		}
		bins = append(bins, Bin{
			ID:       uint32(id.Uint64()),
			ReserveX: new(uint256.Int).Mul(orZero(rx), scaleX),
			ReserveY: new(uint256.Int).Mul(orZero(ry), scaleY),
		})
	}
	sort.Slice(bins, func(i, j int) bool { return bins[i].ID < bins[j].ID })

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
		VolatilityAccumulator: quote.VolatilityAccumulator.Uint64(),
		VolatilityReference:   uint32(quote.VolatilityReference.Uint64()),
		LastVolatilityUpdate:  quote.LastVolatilityUpdate.Uint64(),
		Timestamp:             uint64(time.Now().Unix()),
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{reserves.ReserveX.String(), reserves.ReserveY.String()}
	p.BlockNumber = blockNumber.Uint64()
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{"pool_id": p.Address, "bins": len(bins), "block": p.BlockNumber}).Info("finished getting new pool state")
	return p, nil
}
