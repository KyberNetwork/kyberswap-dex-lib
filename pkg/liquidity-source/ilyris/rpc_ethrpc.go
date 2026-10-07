package ilyris

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// ethrpcChain binds chainReader to ethrpc multicalls.
type ethrpcChain struct {
	client *ethrpc.Client
	lens   string
}

// lensPoolState mirrors BinPoolLens.PoolState field for field, in order.
type lensPoolState struct {
	Pool           common.Address
	TokenX         common.Address
	TokenY         common.Address
	DecimalsX      uint8
	DecimalsY      uint8
	BinStepBps     *big.Int
	SwapFeeBps     *big.Int
	ActiveId       *big.Int
	ActivePriceX18 *big.Int
	TotalFeeRate   *big.Int
	BaseFeeRate    *big.Int
	ReserveX       *big.Int
	ReserveY       *big.Int
	ScannedFrom    *big.Int
	ScannedTo      *big.Int
	PopulatedBins  *big.Int
	MarketGuard    common.Address
	TransferPolicy common.Address
	Owner          common.Address
}

type poolFeeConfig struct {
	BaseFactor               *big.Int
	FilterPeriod             uint16
	DecayPeriod              uint16
	ReductionFactor          uint16
	VariableFeeControl       *big.Int
	MaxVolatilityAccumulator *big.Int
	ProtocolShareBps         uint16
}

type poolFeeState struct {
	VolatilityAccumulator *big.Int
	VolatilityReference   *big.Int
	IdReference           *big.Int
	TimeLastUpdate        *big.Int
}

func (c *ethrpcChain) req(ctx context.Context, blockNumber uint64) *ethrpc.Request {
	req := c.client.R().SetContext(ctx)
	if blockNumber > 0 {
		req.SetBlockNumber(new(big.Int).SetUint64(blockNumber))
	}
	return req
}

func (c *ethrpcChain) PoolState(ctx context.Context, poolAddr string) (RawPoolState, error) {
	var (
		res struct{ State lensPoolState } // a single-tuple output unpacks into the first field
		fc  poolFeeConfig
		fs  poolFeeState
	)
	req := c.req(ctx, 0)
	req.AddCall(&ethrpc.Call{ABI: lensABI, Target: c.lens, Method: "getPoolState",
		Params: []any{common.HexToAddress(poolAddr), big.NewInt(0)}}, []any{&res})
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "feeConfig"}, []any{&fc})
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "feeState"}, []any{&fs})
	resp, err := req.Aggregate()
	if err != nil {
		return RawPoolState{}, err
	}
	header, err := c.client.GetETHClient().HeaderByNumber(ctx, resp.BlockNumber)
	if err != nil {
		return RawPoolState{}, err
	}

	st := &res.State
	return RawPoolState{
		BinStepBps:   uint32(st.BinStepBps.Uint64()),
		DecimalsX:    st.DecimalsX,
		DecimalsY:    st.DecimalsY,
		ActiveID:     int32(st.ActiveId.Int64()),
		TotalFeeRate: st.TotalFeeRate.Uint64(),
		Fee: FeeParams{
			BaseFactor:               uint32(fc.BaseFactor.Uint64()),
			FilterPeriod:             fc.FilterPeriod,
			DecayPeriod:              fc.DecayPeriod,
			ReductionFactor:          fc.ReductionFactor,
			VariableFeeControl:       uint32(fc.VariableFeeControl.Uint64()),
			MaxVolatilityAccumulator: uint32(fc.MaxVolatilityAccumulator.Uint64()),
			VolatilityAccumulator:    uint32(fs.VolatilityAccumulator.Uint64()),
			VolatilityReference:      uint32(fs.VolatilityReference.Uint64()),
			IDReference:              int32(fs.IdReference.Int64()),
			TimeLastUpdate:           fs.TimeLastUpdate.Uint64(),
		},
		MarketGuard:    st.MarketGuard,
		BlockNumber:    resp.BlockNumber.Uint64(),
		BlockTimestamp: header.Time,
	}, nil
}

func (c *ethrpcChain) ScanBitmap(ctx context.Context, poolAddr string, blockNumber uint64, words []int32,
	probes []binProbe) ([]*big.Int, []binProbeResult, error) {
	bits := make([]*big.Int, len(words))
	next := make([]struct {
		Found bool
		Id    *big.Int
	}, len(probes))
	req := c.req(ctx, blockNumber)
	for i, w := range words {
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "getBitmapWord",
			Params: []any{big.NewInt(int64(w))}}, []any{&bits[i]})
	}
	for i, p := range probes {
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "nextNonEmptyBin",
			Params: []any{big.NewInt(int64(p.From)), p.XForY}}, []any{&next[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, nil, err
	}
	out := make([]binProbeResult, len(next))
	for i, n := range next {
		out[i] = binProbeResult{Found: n.Found, ID: int32(n.Id.Int64())}
	}
	return bits, out, nil
}

func (c *ethrpcChain) BinReserves(ctx context.Context, poolAddr string, blockNumber uint64,
	ids []int32) ([]RawBin, error) {
	res := make([]struct {
		ReservesX, ReservesY, TotalShares []*big.Int
	}, (len(ids)+binReservesChunk-1)/binReservesChunk)
	if len(res) == 0 {
		return nil, nil
	}
	req := c.req(ctx, blockNumber)
	for i := range res {
		chunk := ids[i*binReservesChunk : min((i+1)*binReservesChunk, len(ids))]
		params := make([]*big.Int, len(chunk))
		for j, id := range chunk {
			params[j] = big.NewInt(int64(id))
		}
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "getBinReserves",
			Params: []any{params}}, []any{&res[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	out := make([]RawBin, 0, len(ids))
	for i, r := range res {
		for j := range r.ReservesX {
			out = append(out, RawBin{ID: ids[i*binReservesChunk+j], ReserveX: r.ReservesX[j], ReserveY: r.ReservesY[j]})
		}
	}
	return out, nil
}

// GuardState uses tryAggregate: a reverting checkSwap is the signal, not an RPC failure.
func (c *ethrpcChain) GuardState(ctx context.Context, guard, poolAddr string, activeID int32,
	blockNumber uint64) (RawGuardState, error) {
	var (
		paused    bool
		freezeEnd uint64
		none      struct{}
		pool, id  = common.HexToAddress(poolAddr), big.NewInt(int64(activeID))
		one       = big.NewInt(1)
	)
	req := c.req(ctx, blockNumber)
	req.AddCall(&ethrpc.Call{ABI: guardABI, Target: guard, Method: "swapsPaused"}, []any{&paused})
	req.AddCall(&ethrpc.Call{ABI: guardABI, Target: guard, Method: "freezeEnd"}, []any{&freezeEnd})
	for _, xForY := range []bool{true, false} {
		req.AddCall(&ethrpc.Call{ABI: guardABI, Target: guard, Method: "checkSwap",
			Params: []any{pool, xForY, one, one, id, id}}, []any{&none})
	}
	resp, err := req.TryAggregate()
	if err != nil {
		return RawGuardState{}, err
	}
	g := RawGuardState{SwapsPaused: paused, FreezeEnd: freezeEnd}
	for _, ok := range resp.Result {
		g.SwapsPaused = g.SwapsPaused || !ok // unreadable flag or reverting checkSwap: fail closed
	}
	return g, nil
}

func (c *ethrpcChain) FactoryPools(ctx context.Context, factory string, offset, limit int) ([]FactoryPool, int, error) {
	var length *big.Int
	if _, err := c.req(ctx, 0).AddCall(&ethrpc.Call{ABI: factoryABI, Target: factory,
		Method: "allPoolsLength"}, []any{&length}).Call(); err != nil {
		return nil, 0, err
	}
	total := int(length.Int64())
	if offset >= total {
		return nil, total, nil
	}
	end := min(offset+limit, total)
	addrs := make([]common.Address, end-offset)
	req := c.req(ctx, 0)
	for i := range addrs {
		req.AddCall(&ethrpc.Call{ABI: factoryABI, Target: factory, Method: "allPools",
			Params: []any{big.NewInt(int64(offset + i))}}, []any{&addrs[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, total, err
	}
	tokens := make([][2]common.Address, len(addrs))
	req = c.req(ctx, 0)
	for i, a := range addrs {
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: a.Hex(), Method: "tokenX"}, []any{&tokens[i][0]})
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: a.Hex(), Method: "tokenY"}, []any{&tokens[i][1]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, total, err
	}
	out := make([]FactoryPool, len(addrs))
	for i, a := range addrs {
		out[i] = FactoryPool{Address: hexutil.Encode(a[:]),
			TokenX: hexutil.Encode(tokens[i][0][:]), TokenY: hexutil.Encode(tokens[i][1][:])}
	}
	return out, total, nil
}
