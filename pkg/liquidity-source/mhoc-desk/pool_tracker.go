package mhocdesk

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// PoolTracker reads the whole pool state in one multicall plus one timestamp read pinned to the same block.
//
// Price updates are plain storage writes with no event (the maker's updates are 32 bytes of calldata to the
// pool's fallback, kept as cheap as possible; only an update that halts the pool logs UpdateHalted), so a
// log-triggered refresh only sees swaps and owner actions. The tracker must run on every block (Base: every 200 ms)
// or on a short interval. A price card is valid for at most MAX_TTL seconds (validUntil in the state word), so a
// missed refresh fails safe: the simulator rejects an expired card with ErrStalePrice instead of quoting it.
type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: cfg, ethrpcClient: ethrpcClient}, nil
}

// countersOut is the ABI output of counters().
type countersOut struct {
	CardSeq      uint32   `abi:"cardSeq"`
	CardFilled   *big.Int `abi:"cardFilled"`
	FillBlock    uint32   `abi:"fillBlock"`
	BlockFilled  *big.Int `abi:"blockFilled"`
	WindowID     uint32   `abi:"windowId"`
	WindowFilled *big.Int `abi:"windowFilled"`
}

// rawState is what the views return, in ABI types.
type rawState struct {
	StateWord, FloorBase, FloorQuote *big.Int
	MaxNotional, MaxPerUpdate        uint64
	MaxPerWindow                     uint32
	Counters                         countersOut
	BalanceBase, BalanceQuote        *big.Int
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, fmt.Errorf("mhoc-desk: pool %s must have 2 tokens", p.Address)
	}
	var r rawState
	poolAddr := common.HexToAddress(p.Address)
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	add := func(method string, out any) {
		req.AddCall(&ethrpc.Call{ABI: propAMMABI, Target: p.Address, Method: method}, []any{out})
	}
	add("stateWord", &r.StateWord)
	add("floorBase", &r.FloorBase)
	add("floorQuote", &r.FloorQuote)
	add("maxNotional", &r.MaxNotional)
	add("maxNotionalPerUpdate", &r.MaxPerUpdate)
	add("maxNotionalPerWindow", &r.MaxPerWindow)
	add("counters", &r.Counters)
	req.AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: p.Tokens[baseIdx].Address, Method: abi.Erc20BalanceOfMethod,
		Params: []any{poolAddr}}, []any{&r.BalanceBase})
	req.AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: p.Tokens[quoteIdx].Address, Method: abi.Erc20BalanceOfMethod,
		Params: []any{poolAddr}}, []any{&r.BalanceQuote})
	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}
	if resp.BlockNumber == nil {
		return p, fmt.Errorf("mhoc-desk: multicall returned no block number")
	}
	// The expiry and the per-minute counter depend on block.timestamp: read it at the same block.
	ts, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber).GetCurrentBlockTimestamp()
	if err != nil {
		return p, err
	}

	extra, err := json.Marshal(r.toExtra(ts))
	if err != nil {
		return p, err
	}
	p.Extra = string(extra)
	p.Reserves = entity.PoolReserves{r.BalanceBase.String(), r.BalanceQuote.String()}
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = int64(ts)
	return p, nil
}

func (r *rawState) toExtra(ts uint64) Extra {
	return Extra{
		StateWord:    big256.FromBig(r.StateWord),
		FloorBase:    big256.FromBig(r.FloorBase),
		FloorQuote:   big256.FromBig(r.FloorQuote),
		MaxNotional:  r.MaxNotional,
		MaxPerUpdate: r.MaxPerUpdate,
		MaxPerWindow: r.MaxPerWindow,
		Counters: Counters{
			CardSeq:      r.Counters.CardSeq,
			CardFilled:   orZero(r.Counters.CardFilled),
			FillBlock:    r.Counters.FillBlock,
			BlockFilled:  orZero(r.Counters.BlockFilled),
			WindowID:     r.Counters.WindowID,
			WindowFilled: orZero(r.Counters.WindowFilled),
		},
		ReadTimestamp: ts,
	}
}

func orZero(x *big.Int) *uint256.Int {
	if x == nil {
		return new(uint256.Int)
	}
	return big256.FromBig(x)
}
