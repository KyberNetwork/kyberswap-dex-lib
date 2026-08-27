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
)

//go:embed abi/BinPoolLens.json
var lensABIJSON []byte

//go:embed abi/BinFactory.json
var factoryABIJSON []byte

//go:embed abi/ContinuousMarketGuard.json
var guardABIJSON []byte

var (
	lensABI    abi.ABI
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

type lensBinInfo struct {
	Id          *big.Int
	ReserveX    *big.Int
	ReserveY    *big.Int
	TotalShares *big.Int
	PriceX18    *big.Int
}

func (c *EthrpcChain) PoolState(ctx context.Context, poolAddr string, radius uint32) (RawPoolState, error) {
	var st lensPoolState
	var bins []lensBinInfo

	req := c.client.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: c.lens,
		Method: "getPoolState",
		// radius is uint24 on chain. abi.Pack takes it as *big.Int; passing a uint16 here
		// encodes a different type and the call reverts with EMPTY data, which reads as a
		// broken lens rather than a wrong ABI.
		Params: []any{common.HexToAddress(poolAddr), big.NewInt(int64(radius))},
	}, []any{&st})

	resp, err := req.Aggregate()
	if err != nil {
		return RawPoolState{}, err
	}

	// Bins are a second call because the window depends on what the first returned. Pinned to
	// the SAME BLOCK: a book read at a later block than its activeId is internally
	// inconsistent, and the inconsistency is a mispriced quote rather than an error.
	from := new(big.Int).Sub(st.ActiveId, big.NewInt(int64(radius)))
	to := new(big.Int).Add(st.ActiveId, big.NewInt(int64(radius)))
	binReq := c.client.R().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	binReq.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: c.lens,
		Method: "getBins",
		Params: []any{common.HexToAddress(poolAddr), from, to},
	}, []any{&bins})
	if _, err := binReq.Aggregate(); err != nil {
		return RawPoolState{}, err
	}

	out := RawPoolState{
		TokenX:       strings.ToLower(st.TokenX.Hex()),
		TokenY:       strings.ToLower(st.TokenY.Hex()),
		DecimalsX:    st.DecimalsX,
		DecimalsY:    st.DecimalsY,
		BinStepBps:   uint32(st.BinStepBps.Uint64()),
		ActiveID:     int32(st.ActiveId.Int64()),
		TotalFeeRate: st.TotalFeeRate.Uint64(),
		MarketGuard:  strings.ToLower(st.MarketGuard.Hex()),
	}
	if resp.BlockNumber != nil {
		out.BlockNumber = resp.BlockNumber.Uint64()
		ts, err := c.headerTimestamp(ctx, resp.BlockNumber)
		if err != nil {
			return RawPoolState{}, err
		}
		out.BlockTimestamp = ts
	}
	for _, b := range bins {
		out.Bins = append(out.Bins, RawBin{
			ID:       int32(b.Id.Int64()),
			ReserveX: new(big.Int).Set(b.ReserveX),
			ReserveY: new(big.Int).Set(b.ReserveY),
		})
	}
	return out, nil
}

func (c *EthrpcChain) GuardState(ctx context.Context, guard string, blockNumber uint64) (RawGuardState, error) {
	var paused bool
	var freezeEnd uint64

	req := c.client.R().SetContext(ctx)
	if blockNumber > 0 {
		req.SetBlockNumber(new(big.Int).SetUint64(blockNumber))
	}
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
