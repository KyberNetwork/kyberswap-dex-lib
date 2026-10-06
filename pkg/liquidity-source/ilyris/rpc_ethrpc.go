package ilyris

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

//go:embed abi/BinPoolLens.json
var lensABIJSON []byte

//go:embed abi/BinPool.json
var poolABIJSON []byte

//go:embed abi/BinFactory.json
var factoryABIJSON []byte

//go:embed abi/ContinuousMarketGuard.json
var guardABIJSON []byte

var (
	lensABI    abi.ABI
	poolABI    abi.ABI
	factoryABI abi.ABI
	guardABI   abi.ABI
)

func init() {
	// Panics on a malformed ABI, deliberately. A half-parsed ABI produces calls that encode
	// to the wrong selector and fail at runtime as "execution reverted" with no data --
	// indistinguishable from a genuinely reverting contract, and a long way from the cause.
	mustABI := func(raw []byte, into *abi.ABI) {
		parsed, err := abi.JSON(strings.NewReader(string(raw)))
		if err != nil {
			panic("ilyris: bad embedded ABI: " + err.Error())
		}
		*into = parsed
	}
	mustABI(lensABIJSON, &lensABI)
	mustABI(poolABIJSON, &poolABI)
	mustABI(factoryABIJSON, &factoryABI)
	mustABI(guardABIJSON, &guardABI)
}

// EthrpcChain binds chainReader to their client. It holds NO logic -- decisions live in the
// tracker, which is testable against a fake. Everything here is encode, call, decode.
type EthrpcChain struct {
	client *ethrpc.Client
	lens   string
}

func NewEthrpcChain(client *ethrpc.Client, lens string) *EthrpcChain {
	return &EthrpcChain{client: client, lens: lens}
}

var _ chainReader = (*EthrpcChain)(nil)

// lensPoolState mirrors BinPoolLens.PoolState field for field and IN ORDER. abi decoding is
// positional, so a reordered field here decodes silently into the wrong variable -- two
// addresses swapped would look like a valid pool with its tokens inverted.
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

// feeConfig / feeState getters; fields match the ABI output names.
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

func (c *EthrpcChain) PoolState(ctx context.Context, poolAddr string) (RawPoolState, error) {
	// A single-tuple output unpacks into the target's FIRST field, so wrap the struct.
	var res struct{ State lensPoolState }
	var fc poolFeeConfig
	var fs poolFeeState
	st := &res.State

	// One multicall, so getTotalFeeRate() and the fee state it derives from share a block.
	// Radius 0: bins come from the bitmap walk, not the lens window.
	req := c.client.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI: lensABI, Target: c.lens, Method: "getPoolState",
		Params: []any{common.HexToAddress(poolAddr), big.NewInt(0)},
	}, []any{&res})
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "feeConfig"}, []any{&fc})
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "feeState"}, []any{&fs})
	resp, err := req.Aggregate()
	if err != nil {
		return RawPoolState{}, err
	}

	out := RawPoolState{
		TokenX:       hexutil.Encode(st.TokenX[:]),
		TokenY:       hexutil.Encode(st.TokenY[:]),
		DecimalsX:    st.DecimalsX,
		DecimalsY:    st.DecimalsY,
		BinStepBps:   uint32(st.BinStepBps.Uint64()),
		ActiveID:     int32(st.ActiveId.Int64()),
		TotalFeeRate: st.TotalFeeRate.Uint64(),
		MarketGuard:  hexutil.Encode(st.MarketGuard[:]),
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
	}
	if resp.BlockNumber != nil {
		out.BlockNumber = resp.BlockNumber.Uint64()
		ts, err := c.headerTimestamp(ctx, resp.BlockNumber)
		if err != nil {
			return RawPoolState{}, err
		}
		out.BlockTimestamp = ts
	}
	return out, nil
}

// pinned starts a request at blockNumber (latest when 0).
func (c *EthrpcChain) pinned(ctx context.Context, blockNumber uint64) *ethrpc.Request {
	req := c.client.R().SetContext(ctx)
	if blockNumber > 0 {
		req.SetBlockNumber(new(big.Int).SetUint64(blockNumber))
	}
	return req
}

func (c *EthrpcChain) ScanBitmap(ctx context.Context, poolAddr string, blockNumber uint64, words []int32,
	probes []BinProbe) ([]*big.Int, []BinProbeResult, error) {
	bits := make([]*big.Int, len(words))
	next := make([]struct {
		Found bool
		Id    *big.Int
	}, len(probes))
	req := c.pinned(ctx, blockNumber)
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
	out := make([]BinProbeResult, len(next))
	for i, n := range next {
		out[i] = BinProbeResult{Found: n.Found, ID: int32(n.Id.Int64())}
	}
	return bits, out, nil
}

// binReservesChunk bounds one getBinReserves call; all chunks share one multicall.
const binReservesChunk = 512

func (c *EthrpcChain) BinReserves(ctx context.Context, poolAddr string, blockNumber uint64,
	ids []int32) ([]RawBin, error) {
	type reserves struct {
		ReservesX   []*big.Int
		ReservesY   []*big.Int
		TotalShares []*big.Int
	}
	res := make([]reserves, (len(ids)+binReservesChunk-1)/binReservesChunk)
	req := c.pinned(ctx, blockNumber)
	for i := range res {
		chunk := ids[i*binReservesChunk : min((i+1)*binReservesChunk, len(ids))]
		params := make([]*big.Int, len(chunk))
		for j, id := range chunk {
			params[j] = big.NewInt(int64(id))
		}
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "getBinReserves",
			Params: []any{params}}, []any{&res[i]})
	}
	if len(res) > 0 {
		if _, err := req.Aggregate(); err != nil {
			return nil, err
		}
	}
	out := make([]RawBin, 0, len(ids))
	for i, r := range res {
		for j := range r.ReservesX {
			out = append(out, RawBin{ID: ids[i*binReservesChunk+j], ReserveX: r.ReservesX[j], ReserveY: r.ReservesY[j]})
		}
	}
	return out, nil
}

func (c *EthrpcChain) GuardState(ctx context.Context, guard string, blockNumber uint64) (RawGuardState, error) {
	var paused bool
	var freezeEnd uint64

	req := c.pinned(ctx, blockNumber)
	req.AddCall(&ethrpc.Call{ABI: guardABI, Target: guard, Method: "swapsPaused"}, []any{&paused})
	req.AddCall(&ethrpc.Call{ABI: guardABI, Target: guard, Method: "freezeEnd"}, []any{&freezeEnd})
	resp, err := req.Aggregate()
	if err != nil {
		return RawGuardState{}, err
	}
	out := RawGuardState{SwapsPaused: paused, FreezeEnd: freezeEnd, BlockNumber: blockNumber}
	if resp != nil && resp.BlockNumber != nil {
		out.BlockNumber = resp.BlockNumber.Uint64()
	}
	return out, nil
}

// headerTimestamp reads the pinned block's timestamp via eth_getBlockByNumber
// (HeaderByNumber, no transactions). GetCurrentBlockTimestamp is the *current*
// block and would leave a historical pin with Time=0, which makes any nonzero
// freezeEnd look frozen forever.
func (c *EthrpcChain) headerTimestamp(ctx context.Context, blockNumber *big.Int) (uint64, error) {
	if c.client == nil {
		return 0, fmt.Errorf("ilyris: no ethrpc client for block timestamp")
	}
	eth := c.client.GetETHClient()
	if eth == nil {
		return 0, fmt.Errorf("ilyris: no eth client for block timestamp")
	}
	hdr, err := eth.HeaderByNumber(ctx, blockNumber)
	if err != nil {
		return 0, err
	}
	if hdr == nil {
		return 0, fmt.Errorf("ilyris: missing header for block %s", blockNumber)
	}
	return hdr.Time, nil
}

func (c *EthrpcChain) FactoryPools(ctx context.Context, factory string, offset, limit int) ([]string, int, error) {
	var length *big.Int
	req := c.client.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{ABI: factoryABI, Target: factory, Method: "allPoolsLength"}, []any{&length})
	if _, err := req.Aggregate(); err != nil {
		return nil, 0, err
	}
	total := int(length.Int64())
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}

	addrs := make([]common.Address, end-offset)
	batch := c.client.R().SetContext(ctx)
	for i := offset; i < end; i++ {
		batch.AddCall(&ethrpc.Call{
			ABI: factoryABI, Target: factory, Method: "allPools",
			Params: []any{big.NewInt(int64(i))},
		}, []any{&addrs[i-offset]})
	}
	if _, err := batch.Aggregate(); err != nil {
		return nil, total, err
	}

	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, strings.ToLower(a.Hex()))
	}
	return out, total, nil
}
