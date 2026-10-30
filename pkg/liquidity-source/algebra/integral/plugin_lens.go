package integral

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

//go:generate go run github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral/contracts AlgebraIntegralPluginLens abis/plugin.json
//go:embed abis/plugin.json
var pluginLensJSON []byte

// pluginLens is AlgebraIntegralPluginLens, built from pkg/util/ephemeral/contracts/algebra/integral.
var pluginLens = ephemeral.MustLoad("AlgebraIntegralPluginLens", pluginLensJSON)

// pluginWalk is how far the lens walks the oracle's timepoints: from the newest, or from Start, down
// to the first that is uninitialized or older than Cutoff, to KnownIndex while it still holds the
// timepoint of KnownTimestamp, or MaxCount of them.
type pluginWalk struct {
	FromHead       bool
	Start          uint16
	Cutoff         uint32
	MaxCount       *big.Int
	HasKnown       bool
	KnownIndex     uint16
	KnownTimestamp uint32
}

// callResult is one call's outcome, as a multicall's tryAggregate reports it.
type callResult struct {
	Success    bool
	ReturnData []byte
}

// pluginSnapshot is what getSnapshot returns. Timepoints holds Count replies of timepoints(), from
// slot First down; Extras those of slots 0, Head + 1, Head + 2 and Head - 1 when walking from Head.
type pluginSnapshot struct {
	BlockNumber *big.Int
	Reads       []callResult
	Walked      bool
	Head        uint16
	First       uint16
	Next        uint16
	Done        bool
	Count       *big.Int
	Timepoints  []byte
	Extras      []byte
}

// timepointSize is the bytes of one timepoint as timepoints() answers it: seven words.
const timepointSize = 7 * 32

// errPluginLensMismatch means the lens's reads do not fit the tracker's, which then reads the
// plugin by multicall.
var errPluginLensMismatch = errors.New("algebra-integral: plugin lens reads do not fit the tracker's")

// pluginRead makes the reads of a plugin readPluginData decodes: the multicall of its settings,
// then the oracle's timepoints. rpcPluginRead makes them by multicall, pluginLensRead through the
// lens.
type pluginRead interface {
	// settings makes req's calls as TryAggregate does. cache is what timepoints will be given.
	settings(req *ethrpc.Request, cache map[uint16]Timepoint) (*ethrpc.Response, error)
	// timepoints reads the oracle's timepoints as GetTimepoints does, from cache.
	timepoints(cache map[uint16]Timepoint, currentIndex uint16, currentTimestamp uint32) (map[uint16]Timepoint,
		error)
}

// rpcPluginRead reads a plugin by multicall: one for its settings, then a page of timepoints each.
type rpcPluginRead struct {
	d         *PoolTracker
	ctx       context.Context
	plugin    string
	block     *big.Int
	overrides map[common.Address]gethclient.OverrideAccount
}

func (r rpcPluginRead) settings(req *ethrpc.Request, _ map[uint16]Timepoint) (*ethrpc.Response, error) {
	return req.SetContext(r.ctx).SetBlockNumber(r.block).SetOverrides(r.overrides).TryAggregate()
}

func (r rpcPluginRead) timepoints(cache map[uint16]Timepoint, currentIndex uint16,
	currentTimestamp uint32) (map[uint16]Timepoint, error) {
	return r.d.getTimepoints(r.ctx, r.plugin, r.block, currentIndex, currentTimestamp, cache, r.overrides)
}

// pluginLensRead reads a plugin through the lens, in one call: the settings calls the tracker
// builds, replayed, and the oracle's timepoints, walked back from the newest. ReadTimepoints then
// pages through what the lens read, so the timepoints are GetTimepoints' at the same block.
type pluginLensRead struct {
	d         *PoolTracker
	ctx       context.Context
	plugin    common.Address
	block     *big.Int
	yesterday uint32

	walked bool
	head   uint16                  // the oracle's newest timepoint, as the lens read it
	slots  map[uint16]TimepointRPC // every timepoint the lens read
	// tooOld is how far below head the nearest slot read lies that is uninitialized or older than
	// yesterday; past the ring when none does. Every slot beyond it is too.
	tooOld int
	// faked are the slots past tooOld served without a read.
	faked []uint16
}

func newPluginLensRead(ctx context.Context, d *PoolTracker, plugin common.Address,
	block *big.Int) *pluginLensRead {
	return &pluginLensRead{d: d, ctx: ctx, plugin: plugin, block: block, yesterday: blockTimestamp() - WINDOW,
		slots: map[uint16]TimepointRPC{}, tooOld: math.MaxUint16 + 1}
}

// settings makes req's calls through the lens, which walks the timepoints in the same call down to
// the newest that cache holds, and decodes the results as ethrpc decodes a tryAggregate: a failed
// call leaves its output untouched, and a successful one is unpacked with the first of its ABIs
// that fits.
func (r *pluginLensRead) settings(req *ethrpc.Request, cache map[uint16]Timepoint) (*ethrpc.Response, error) {
	calls := make([][]byte, len(req.Calls))
	for i, c := range req.Calls {
		data, err := c.ABI.Pack(c.Method, c.Params...)
		if err != nil {
			return nil, err
		}
		calls[i] = data
	}

	walk := pluginWalk{FromHead: true, Cutoff: r.yesterday,
		MaxCount: big.NewInt(int64(algebra.MaxTimepointReads))}
	// The newest timepoint cached, while its slot still holds it, is where the cache takes over:
	// ReadTimepoints asks for none it holds, and a refresh reads only what was written since.
	for idx, tp := range cache {
		if tp.Initialized && (!walk.HasKnown || tp.BlockTimestamp > walk.KnownTimestamp) {
			walk.HasKnown, walk.KnownIndex, walk.KnownTimestamp = true, idx, tp.BlockTimestamp
		}
	}
	snap, err := r.d.pluginLensCall(r.ctx, r.block, r.plugin, calls, walk)
	if err != nil {
		return nil, err
	}
	if len(snap.Reads) != len(calls) {
		return nil, fmt.Errorf("%w: %d results for %d calls", errPluginLensMismatch, len(snap.Reads), len(calls))
	}

	resp := &ethrpc.Response{Request: req, BlockNumber: snap.BlockNumber, Result: make([]bool, len(calls))}
	for i, c := range req.Calls {
		if resp.Result[i] = snap.Reads[i].Success; !resp.Result[i] {
			continue
		}
		for j, unpackABI := range c.UnpackABI {
			if unpackABI.UnpackIntoInterface(c.Output[j], c.Method, snap.Reads[i].ReturnData) == nil {
				break
			}
		}
	}

	r.walked, r.head = snap.Walked, snap.Head
	if !snap.Walked {
		return resp, nil
	}
	return resp, r.keep(snap)
}

// timepoints is GetTimepoints over what the lens read.
func (r *pluginLensRead) timepoints(cache map[uint16]Timepoint, currentIndex uint16,
	currentTimestamp uint32) (map[uint16]Timepoint, error) {
	if !r.walked || r.head != currentIndex {
		return nil, fmt.Errorf("%w: the oracle's newest timepoint is %d, the lens walked from %d (walked %v)",
			errPluginLensMismatch, currentIndex, r.head, r.walked)
	}
	timepoints, err := algebra.ReadTimepoints(cache, r.yesterday, currentIndex, currentTimestamp, r.fetch)
	if err != nil {
		return nil, err
	}
	for _, idx := range r.faked {
		if _, ok := timepoints[idx]; ok {
			return nil, fmt.Errorf("%w: slot %d is kept but was not read", errPluginLensMismatch, idx)
		}
	}
	return timepoints, nil
}

// fetch serves ReadTimepoints a page from what the lens read, walking again from a slot it has not
// read. A slot past tooOld is no newer than one already too old to keep, and ReadTimepoints asks
// for it only to search past where it stops: rather than costing another call, it reads as
// uninitialized, as unusable to the search as it is; timepoints checks that none of them is kept.
func (r *pluginLensRead) fetch(indices []uint16, page []TimepointRPC) error {
	for i, idx := range indices {
		tp, ok := r.slots[idx]
		if !ok && int(r.head-idx) > r.tooOld {
			r.faked = append(r.faked, idx)
			page[i] = TimepointRPC{TickCumulative: new(big.Int), VolatilityCumulative: new(big.Int),
				Tick: new(big.Int), AverageTick: new(big.Int)}
			continue
		}
		if !ok {
			snap, err := r.d.pluginLensCall(r.ctx, r.block, r.plugin, nil,
				pluginWalk{Start: idx, Cutoff: r.yesterday, MaxCount: big.NewInt(int64(algebra.MaxTimepointReads))})
			if err != nil {
				return err
			}
			if snap.First != idx || snap.Count.Sign() == 0 {
				return fmt.Errorf("%w: a walk from slot %d read nothing", errPluginLensMismatch, idx)
			}
			if err := r.keep(snap); err != nil {
				return err
			}
			tp = r.slots[idx]
		}
		page[i] = tp
	}
	return nil
}

// keep records the timepoints a lens call read.
func (r *pluginLensRead) keep(snap pluginSnapshot) error {
	if !snap.Count.IsUint64() || uint64(len(snap.Timepoints)) != snap.Count.Uint64()*timepointSize {
		return fmt.Errorf("%w: %d bytes for %s timepoints", errPluginLensMismatch, len(snap.Timepoints), snap.Count)
	}
	for k := 0; k*timepointSize < len(snap.Timepoints); k++ {
		if err := r.keepSlot(snap.First-uint16(k), snap.Timepoints[k*timepointSize:(k+1)*timepointSize]); err != nil {
			return err
		}
	}
	if len(snap.Extras) == 0 {
		return nil
	}
	extras := []uint16{0, snap.Head + 1, snap.Head + 2, snap.Head - 1}
	if len(snap.Extras) != len(extras)*timepointSize {
		return fmt.Errorf("%w: %d bytes of extra timepoints", errPluginLensMismatch, len(snap.Extras))
	}
	for k, idx := range extras {
		if err := r.keepSlot(idx, snap.Extras[k*timepointSize:(k+1)*timepointSize]); err != nil {
			return err
		}
	}
	return nil
}

// keepSlot decodes slot idx's reply as ethrpc decodes it.
func (r *pluginLensRead) keepSlot(idx uint16, data []byte) error {
	var tp TimepointRPC
	if err := basePluginV2ABI.UnpackIntoInterface(&tp, votalityOraclePluginTimepointsMethod, data); err != nil {
		return err
	}
	r.slots[idx] = tp
	if !tp.Initialized || tp.BlockTimestamp < r.yesterday {
		r.tooOld = min(r.tooOld, int(r.head-idx))
	}
	return nil
}

// callPluginLens runs the lens on plugin at block.
func (d *PoolTracker) callPluginLens(ctx context.Context, block *big.Int, plugin common.Address, calls [][]byte,
	walk pluginWalk) (pluginSnapshot, error) {
	var snap pluginSnapshot
	err := ephemeral.Call(ctx, d.EthrpcClient.GetETHClient(), pluginLens, &snap, d.config.LensGas, block, plugin,
		calls, walk)
	return snap, err
}
