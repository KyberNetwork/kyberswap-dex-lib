package navjit

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Robinhood chain (4663) addresses used by the live tests.
const (
	defaultRPC   = "https://rpc.mainnet.chain.robinhood.com"
	multicall3   = "0xcA11bde05977b3631167028862be2a173976CA11"
	stateView    = "0xF3334192D15450CdD385c8B70e03f9A6bD9E673b"
	usdgAddr     = "0x5fc5360d0400a0fd4f2af552add042d716f1d168"
	usdgDecimals = 6
)

const stateViewABIJson = `[
 {"type":"function","name":"getTickBitmap","stateMutability":"view","inputs":[{"name":"poolId","type":"bytes32"},{"name":"tick","type":"int16"}],"outputs":[{"name":"tickBitmap","type":"uint256"}]},
 {"type":"function","name":"getTickInfo","stateMutability":"view","inputs":[{"name":"poolId","type":"bytes32"},{"name":"tick","type":"int24"}],"outputs":[{"name":"liquidityGross","type":"uint128"},{"name":"liquidityNet","type":"int128"},{"name":"feeGrowthOutside0X128","type":"uint256"},{"name":"feeGrowthOutside1X128","type":"uint256"}]},
 {"type":"function","name":"getSlot0","stateMutability":"view","inputs":[{"name":"poolId","type":"bytes32"}],"outputs":[{"name":"sqrtPriceX96","type":"uint160"},{"name":"tick","type":"int24"},{"name":"protocolFee","type":"uint24"},{"name":"lpFee","type":"uint24"}]}
]`

var stateViewABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(stateViewABIJson))
	if err != nil {
		panic(err)
	}
	return a
}()

func skipInCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
}

func liveRPC() *ethrpc.Client {
	url := os.Getenv("NAVJIT_RPC")
	if url == "" {
		url = defaultRPC
	}
	return ethrpc.New(url).SetMulticallContract(common.HexToAddress(multicall3))
}

// poolSpec is a v4 PoolKey plus its id; token decimals are looked up per currency.
type poolSpec struct {
	Name        string
	ID          string
	Currency0   string
	Currency1   string
	Dec0, Dec1  uint8
	Fee         uint32
	TickSpacing int32
	Hooks       string
}

// scanTicks reads every initialised tick within ±words bitmap words of the current tick from
// StateView: what Kyber's indexer would have accumulated from ModifyLiquidity logs.
func scanTicks(ctx context.Context, t *testing.T, rpc *ethrpc.Client, spec poolSpec, words int) []uniswapv3.Tick {
	id := common.HexToHash(spec.ID)
	var slot0 struct {
		SqrtPriceX96 *big.Int
		Tick         *big.Int
		ProtocolFee  *big.Int
		LpFee        *big.Int
	}
	// Unpinned: on 4663 the block Multicall3 reports is the L1 block (see L2BlockByChain).
	_, err := rpc.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{ABI: stateViewABI, Target: stateView,
		Method: "getSlot0", Params: []any{id}}, []any{&slot0}).Aggregate()
	require.NoError(t, err)

	ts := int64(spec.TickSpacing)
	comp := slot0.Tick.Int64() / ts
	if slot0.Tick.Int64() < 0 && slot0.Tick.Int64()%ts != 0 {
		comp--
	}
	center := comp >> 8
	// Words around the current tick, plus every word up to the min/max usable ticks when the full
	// range is small (wide tick spacing), else just the two edge words where full-range positions sit.
	const maxTick = 887272
	minWord, maxWord := (-(maxTick/ts)-1)>>8, (maxTick/ts)>>8
	wordSet := map[int64]bool{minWord: true, maxWord: true}
	for w := center - int64(words); w <= center+int64(words); w++ {
		wordSet[w] = true
	}
	if maxWord-minWord <= 512 {
		for w := minWord; w <= maxWord; w++ {
			wordSet[w] = true
		}
	}
	wordList := make([]int64, 0, len(wordSet))
	for w := range wordSet {
		wordList = append(wordList, w)
	}
	bitmaps := make([]*big.Int, len(wordList))
	req := rpc.NewRequest().SetContext(ctx)
	for i, w := range wordList {
		req.AddCall(&ethrpc.Call{ABI: stateViewABI, Target: stateView, Method: "getTickBitmap",
			Params: []any{id, int16(w)}}, []any{&bitmaps[i]})
	}
	_, err = req.Aggregate()
	require.NoError(t, err)

	var idx []int64
	for i, bm := range bitmaps {
		w := wordList[i]
		for b := 0; b < 256; b++ {
			if bm != nil && bm.Bit(b) == 1 {
				idx = append(idx, (w*256+int64(b))*ts)
			}
		}
	}
	if len(idx) == 0 {
		return nil
	}
	sort.Slice(idx, func(i, j int) bool { return idx[i] < idx[j] })
	type tickInfo struct {
		LiquidityGross        *big.Int
		LiquidityNet          *big.Int
		FeeGrowthOutside0X128 *big.Int
		FeeGrowthOutside1X128 *big.Int
	}
	infos := make([]tickInfo, len(idx))
	req = rpc.NewRequest().SetContext(ctx)
	for i, ti := range idx {
		req.AddCall(&ethrpc.Call{ABI: stateViewABI, Target: stateView, Method: "getTickInfo",
			Params: []any{id, big.NewInt(ti)}}, []any{&infos[i]})
	}
	_, err = req.Aggregate()
	require.NoError(t, err)
	ticks := make([]uniswapv3.Tick, 0, len(idx))
	for i, ti := range idx {
		if infos[i].LiquidityGross == nil || infos[i].LiquidityGross.Sign() == 0 {
			continue
		}
		ticks = append(ticks, uniswapv3.Tick{Index: int(ti), LiquidityGross: infos[i].LiquidityGross,
			LiquidityNet: infos[i].LiquidityNet})
	}
	return ticks
}

// trackPool runs Kyber's own uniswap-v4 PoolTracker over the pool (slot0, liquidity, hook Track,
// reserves), starting from the ticks an indexer would hold.
func trackPool(ctx context.Context, t *testing.T, rpc *ethrpc.Client, spec poolSpec) entity.Pool {
	ticks := scanTicks(ctx, t, rpc, spec, 64)
	staticExtra, err := json.Marshal(uniswapv4.StaticExtra{
		Fee: spec.Fee, TickSpacing: spec.TickSpacing, HooksAddress: common.HexToAddress(spec.Hooks),
		Multicall3Address: common.HexToAddress(multicall3),
	})
	require.NoError(t, err)
	extra, err := json.Marshal(uniswapv4.Extra{Extra: &uniswapv3.Extra{
		Liquidity: big.NewInt(0), SqrtPriceX96: big.NewInt(0), TickSpacing: uint64(spec.TickSpacing),
		Tick: big.NewInt(0), Ticks: ticks}})
	require.NoError(t, err)

	p := entity.Pool{
		Address:  strings.ToLower(spec.ID),
		Exchange: uniswapv4.DexType,
		Type:     uniswapv4.DexType,
		Tokens: []*entity.PoolToken{
			{Address: strings.ToLower(spec.Currency0), Decimals: spec.Dec0, Swappable: true},
			{Address: strings.ToLower(spec.Currency1), Decimals: spec.Dec1, Swappable: true},
		},
		Reserves:    entity.PoolReserves{"0", "0"},
		StaticExtra: string(staticExtra),
		Extra:       string(extra),
	}
	tracker := uniswapv4.NewPoolTracker(&uniswapv4.Config{
		ChainID: valueobject.ChainIDRobinhood, DexID: uniswapv4.DexType,
		StateViewAddress: stateView, Multicall3Address: multicall3,
	}, rpc, nil)
	p, err = tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	return p
}

// v4Quote is the stock V4Quoter's exact-in answer at `block`; ok=false when it reverts.
func v4Quote(ctx context.Context, rpc *ethrpc.Client, block *big.Int, spec poolSpec, zeroForOne bool,
	amt *big.Int) (*big.Int, bool) {
	quoter := QuoterByChain[valueobject.ChainIDRobinhood]
	qp := quoteParams{ZeroForOne: zeroForOne, ExactAmount: amt, HookData: []byte{}}
	qp.PoolKey.Currency0 = common.HexToAddress(spec.Currency0)
	qp.PoolKey.Currency1 = common.HexToAddress(spec.Currency1)
	qp.PoolKey.Fee = big.NewInt(int64(spec.Fee))
	qp.PoolKey.TickSpacing = big.NewInt(int64(spec.TickSpacing))
	qp.PoolKey.Hooks = common.HexToAddress(spec.Hooks)
	var out quoteResult
	req := rpc.NewRequest().SetContext(ctx)
	if block != nil {
		req.SetBlockNumber(block)
	}
	res, err := req.AddCall(&ethrpc.Call{ABI: quoterABI, Target: hexutil.Encode(quoter[:]),
		Method: "quoteExactInputSingle", Params: []any{qp}}, []any{&out}).TryAggregate()
	if err != nil || !res.Result[0] || out.AmountOut == nil {
		return nil, false
	}
	return out.AmountOut, true
}

// errBps is (sim - ref) / ref in basis points, signed: negative means the simulator under-quotes.
func errBps(sim, ref *big.Int) float64 {
	d := new(big.Float).SetInt(new(big.Int).Sub(sim, ref))
	r, _ := d.Quo(d, new(big.Float).SetInt(ref)).Float64()
	return r * 10_000
}

// liveLog mirrors every line to t.Log and, when NAVJIT_LIVE_OUT is set, appends it to that file.
type liveLog struct {
	t *testing.T
	f *os.File
}

func newLiveLog(t *testing.T) *liveLog {
	l := &liveLog{t: t}
	if path := os.Getenv("NAVJIT_LIVE_OUT"); path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		require.NoError(t, err)
		l.f = f
		t.Cleanup(func() { _ = f.Close() })
	}
	return l
}

func (l *liveLog) Printf(format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	l.t.Log(s)
	if l.f != nil {
		_, _ = l.f.WriteString(s + "\n")
	}
}
