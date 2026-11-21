package kalos

import (
	"math/big"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Live tests read Kinetiq Elysium (99901); they are skipped in CI and without the endpoint. Run with:
//
//	ELYSIUM_RPC_URL=https://rpc.elysium.kinetiq.xyz go test ./pkg/liquidity-source/uniswap/v4/hooks/kalos/ -run Live -v
//
// KALOS_WRITE_FIXTURE=1 also rewrites testdata/pools.json from TestLiveQuotesMatchV4Quoter.
const elysiumRPCEnv = "ELYSIUM_RPC_URL"

var (
	livePoolManager = common.HexToAddress("0xE2e105d3F7209A9DA11f83cCf3E7398a753823F1")
	liveStateView   = "0x27eD73aE0652fdCf71d6793412A7525f32bE9dfE"
	liveV4Quoter    = common.HexToAddress("0x379F556c5a748BD7Cc9baF0F7A344094Ac1814f5")
	liveUSDC        = common.HexToAddress("0xb47c4C8F73F009CC0C0d3f92B388d13E293F0B65")
	liveKNTQ        = common.HexToAddress("0xe45a355d9e9cAd2aD85F2d03E3eE3102A036acF0")

	// hookRuntimeHash is keccak256 of KalosDynamic's runtime code, as its repository's reproducible build gives it.
	hookRuntimeHash = common.HexToHash("0x295107603ee96a130e57cdbf0516c875972c45503a8528fae3a1ebe65d6f3cc1")

	// The hook's storage slots (KalosV4FeeHookBase's layout): _poolFeeParams and _poolFeeState.
	feeParamsSlot = big.NewInt(15)
	feeStateSlot  = big.NewInt(16)
)

// livePools are every pool registered with KalosDynamic (its PoolRegistered logs), with their Initialize blocks.
var livePools = []struct {
	name      string
	id        common.Hash
	block     uint64
	currency1 common.Address
	spacing   int64
	decimals  [2]uint8
}{
	{"HYPE/USDC", common.HexToHash("0x1ae8edbaa10136b36eb039dd179de678b11893a16265fece7a56f41cfe0ecd42"), 6898602,
		liveUSDC, 10, [2]uint8{18, 6}},
	{"HYPE/KNTQ", common.HexToHash("0x5218e64384e287e85f1af6fb1893f80d8e1d9fabb03c118bcb265f082c041e98"), 6898643,
		liveKNTQ, 200, [2]uint8{18, 18}},
}

const liveABIJSON = `[
{"type":"function","name":"quoteExactInputSingle","stateMutability":"nonpayable","inputs":[{"name":"params","type":"tuple","components":[{"name":"poolKey","type":"tuple","components":[{"name":"currency0","type":"address"},{"name":"currency1","type":"address"},{"name":"fee","type":"uint24"},{"name":"tickSpacing","type":"int24"},{"name":"hooks","type":"address"}]},{"name":"zeroForOne","type":"bool"},{"name":"exactAmount","type":"uint128"},{"name":"hookData","type":"bytes"}]}],"outputs":[{"name":"amountOut","type":"uint256"},{"name":"gasEstimate","type":"uint256"}]},
{"type":"function","name":"quoteExactInput","stateMutability":"nonpayable","inputs":[{"name":"params","type":"tuple","components":[{"name":"exactCurrency","type":"address"},{"name":"path","type":"tuple[]","components":[{"name":"intermediateCurrency","type":"address"},{"name":"fee","type":"uint24"},{"name":"tickSpacing","type":"int24"},{"name":"hooks","type":"address"},{"name":"hookData","type":"bytes"}]},{"name":"exactAmount","type":"uint128"}]}],"outputs":[{"name":"amountOut","type":"uint256"},{"name":"gasEstimate","type":"uint256"}]},
{"type":"function","name":"previewFeeAt","stateMutability":"view","inputs":[{"name":"key","type":"tuple","components":[{"name":"currency0","type":"address"},{"name":"currency1","type":"address"},{"name":"fee","type":"uint24"},{"name":"tickSpacing","type":"int24"},{"name":"hooks","type":"address"}]},{"name":"zeroForOne","type":"bool"},{"name":"timestamp","type":"uint256"}],"outputs":[{"name":"q","type":"tuple","components":[{"name":"registered","type":"bool"},{"name":"kind","type":"uint8"},{"name":"feeSide","type":"uint8"},{"name":"preset","type":"uint8"},{"name":"feeCurrency","type":"address"},{"name":"totalFee","type":"uint24"},{"name":"lpFee","type":"uint24"},{"name":"hookFee","type":"uint24"},{"name":"kalosShareBps","type":"uint16"},{"name":"creatorShareBps","type":"uint16"},{"name":"volatilityAccumulator","type":"uint24"}]}]}
]`

var liveABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(liveABIJSON))
	if err != nil {
		panic(err)
	}
	return a
}()

// fixturePool is one pool of testdata/pools.json (fixture_test.go reads it): as tracked, and what V4Quoter quoted.
type fixturePool struct {
	Name   string         `json:"name"`
	Pool   entity.Pool    `json:"pool"`
	Quotes []fixtureQuote `json:"quotes"`
}

type fixtureQuote struct {
	ZeroForOne bool   `json:"zeroForOne"`
	AmountIn   string `json:"amountIn"`
	AmountOut  string `json:"amountOut"`
}

type poolKey struct {
	Currency0   common.Address
	Currency1   common.Address
	Fee         *big.Int
	TickSpacing *big.Int
	Hooks       common.Address
}

type pathKey struct {
	IntermediateCurrency common.Address
	Fee                  *big.Int
	TickSpacing          *big.Int
	Hooks                common.Address
	HookData             []byte
}

type singleParams struct {
	PoolKey     poolKey
	ZeroForOne  bool
	ExactAmount *big.Int
	HookData    []byte
}

type pathParams struct {
	ExactCurrency common.Address
	Path          []pathKey
	ExactAmount   *big.Int
}

// feeQuote is KalosV4FeeHookBase.FeeQuote as previewFeeAt returns it.
type feeQuote = struct {
	Registered            bool           `json:"registered"`
	Kind                  uint8          `json:"kind"`
	FeeSide               uint8          `json:"feeSide"`
	Preset                uint8          `json:"preset"`
	FeeCurrency           common.Address `json:"feeCurrency"`
	TotalFee              *big.Int       `json:"totalFee"`
	LpFee                 *big.Int       `json:"lpFee"`
	HookFee               *big.Int       `json:"hookFee"`
	KalosShareBps         uint16         `json:"kalosShareBps"`
	CreatorShareBps       uint16         `json:"creatorShareBps"`
	VolatilityAccumulator *big.Int       `json:"volatilityAccumulator"`
}

type liveEnv struct {
	t    *testing.T
	live *testutil.LiveClient
	geth *gethclient.Client
}

func newLiveEnv(t *testing.T) *liveEnv {
	url := testutil.LiveURL(t, elysiumRPCEnv)
	live := testutil.DialLive(t, url)
	return &liveEnv{t: t, live: live, geth: gethclient.New(live.RPC)}
}

// call runs a read on V4Quoter (or previewFeeAt on the hook) at block, under state overrides when given.
func (e *liveEnv) call(method string, block *big.Int, overrides map[common.Address]ethereum.OverrideAccount,
	args ...any) ([]any, error) {
	data, err := liveABI.Pack(method, args...)
	require.NoError(e.t, err)
	to := liveV4Quoter
	if method == "previewFeeAt" {
		to = HookAddresses[0]
	}
	var ov *map[common.Address]ethereum.OverrideAccount
	if overrides != nil {
		ov = &overrides
	}
	res, err := e.geth.CallContract(e.t.Context(), ethereum.CallMsg{To: &to, Data: data, Gas: 30_000_000}, block, ov)
	if err != nil {
		return nil, err
	}
	return liveABI.Unpack(method, res)
}

func (e *liveEnv) quote(key poolKey, zeroForOne bool, amountIn, block *big.Int,
	overrides map[common.Address]ethereum.OverrideAccount) (*big.Int, uint64) {
	res, err := e.call("quoteExactInputSingle", block, overrides, singleParams{key, zeroForOne, amountIn, []byte{}})
	require.NoError(e.t, err, "quote zfo=%v in=%s", zeroForOne, amountIn)
	return res[0].(*big.Int), res[1].(*big.Int).Uint64()
}

// track lists a pool from its Initialize log through the v4 factory, as pool-service does, and bootstraps it with the
// v4 tracker through the lens: price, liquidity, ticks and the hook's state at one block. Timestamp is that block's,
// so the simulator prices at it (QuoteAt).
func (e *liveEnv) track(i int) (entity.Pool, poolKey) {
	lp := livePools[i]
	ctx := e.t.Context()
	initialize := crypto.Keccak256Hash([]byte("Initialize(bytes32,address,address,uint24,int24,address,uint160,int24)"))
	logs, err := e.live.Eth.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(lp.block), ToBlock: new(big.Int).SetUint64(lp.block),
		Addresses: []common.Address{livePoolManager}, Topics: [][]common.Hash{{initialize}, {lp.id}},
	})
	require.NoError(e.t, err)
	require.Len(e.t, logs, 1)
	cfg := &uniswapv4.Config{
		ChainID: valueobject.ChainIDElysium, DexID: uniswapv4.DexType, StateViewAddress: liveStateView,
		PoolManagerAddress: livePoolManager.Hex(), Multicall3Address: testutil.Multicall3.Hex(),
	}
	cfg.UseLens = true
	p, err := uniswapv4.NewPoolFactory(cfg).DecodePoolCreated(logs[0])
	require.NoError(e.t, err)
	require.Equal(e.t, DexType, p.Exchange)
	for j, d := range lp.decimals {
		p.Tokens[j].Decimals = d
	}
	// The public endpoint is load balanced: a read pinned to the newest block can reach a node one block behind
	// ("header not found"). Such a bootstrap is tried again.
	var tracked entity.Pool
	for attempt := 0; ; attempt++ {
		tracked, err = uniswapv4.NewPoolTracker(cfg, e.live.Ethrpc, nil).BootstrapPoolState(ctx, *p,
			pool.GetNewPoolStateParams{})
		if err == nil || attempt == 5 || !strings.Contains(err.Error(), "header not found") {
			break
		}
		time.Sleep(2 * time.Second)
	}
	require.NoError(e.t, err)
	require.Equal(e.t, DexType, tracked.Exchange)
	header, err := e.live.Eth.HeaderByNumber(ctx, new(big.Int).SetUint64(tracked.BlockNumber))
	require.NoError(e.t, err)
	tracked.Timestamp = int64(header.Time)
	return tracked, poolKey{common.Address{}, lp.currency1, big.NewInt(0x800000), big.NewInt(lp.spacing), HookAddresses[0]}
}

func hookExtraOf(t *testing.T, p entity.Pool) (Extra, int32) {
	var extra struct {
		HookExtra json.RawMessage `json:"hX"`
		Tick      int32           `json:"tick"`
	}
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	var hx Extra
	require.NoError(t, uniswapv4.HookExtra(extra.HookExtra).Unmarshal(&hx))
	return hx, extra.Tick
}

func withHookExtra(t *testing.T, p entity.Pool, hx Extra) entity.Pool {
	var extra map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	raw, err := json.Marshal(hx)
	require.NoError(t, err)
	extra["hX"] = raw
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	p.Extra = string(b)
	return p
}

// amountsIn are the sizes quoted: 1e-8, 1e-6, 1e-4, 1e-2 and 1e-1 of the pool's input-side reserve.
func amountsIn(p entity.Pool, zeroForOne bool) []*big.Int {
	idx := 1
	if zeroForOne {
		idx = 0
	}
	reserve, _ := new(big.Int).SetString(p.Reserves[idx], 10)
	var out []*big.Int
	for _, div := range []int64{100_000_000, 1_000_000, 10_000, 100, 10} {
		if a := new(big.Int).Quo(reserve, big.NewInt(div)); a.Sign() > 0 {
			out = append(out, a)
		}
	}
	return out
}

func tokensOf(p entity.Pool, zeroForOne bool) (tokenIn, tokenOut string) {
	if zeroForOne {
		return p.Tokens[0].Address, p.Tokens[1].Address
	}
	return p.Tokens[1].Address, p.Tokens[0].Address
}

// TestLiveHookCode: the hook at HookAddresses is the reviewed build this package mirrors.
func TestLiveHookCode(t *testing.T) {
	e := newLiveEnv(t)
	code, err := e.live.Eth.CodeAt(t.Context(), HookAddresses[0], nil)
	require.NoError(t, err)
	require.Equal(t, hookRuntimeHash, crypto.Keccak256Hash(code))
}

// TestLiveQuotesMatchV4Quoter: every pool of the hook, tracked by the v4 tracker, quotes exactly what Elysium's
// V4Quoter (bound to the same PoolManager) quotes at the tracked block, both directions, five sizes; and a round
// trip through the same pool in one route (two swaps, the second at the state the first leaves, UpdateBalance)
// equals the quoter's two-hop path.
func TestLiveQuotesMatchV4Quoter(t *testing.T) {
	e := newLiveEnv(t)
	var fixture []fixturePool
	for i, lp := range livePools {
		p, key := e.track(i)
		block := new(big.Int).SetUint64(p.BlockNumber)
		sim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDElysium)
		require.NoError(t, err)
		fp := fixturePool{Name: lp.name, Pool: p}
		for _, zfo := range []bool{true, false} {
			tokenIn, tokenOut := tokensOf(p, zfo)
			for _, amountIn := range amountsIn(p, zfo) {
				want, quoterGas := e.quote(key, zfo, amountIn, block, nil)
				got, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn}, TokenOut: tokenOut})
				if want.Sign() == 0 { // a zero-output swap is no quote
					require.Error(t, err, "%s zfo=%v in=%s", lp.name, zfo, amountIn)
					t.Logf("| %s | %d | %v | %s | 0 | no quote | %d |", lp.name, p.BlockNumber, zfo, amountIn, quoterGas)
					continue
				}
				require.NoError(t, err)
				t.Logf("| %s | %d | %v | %s | %s | %s | %d |", lp.name, p.BlockNumber, zfo, amountIn, want,
					got.TokenAmountOut.Amount, quoterGas)
				require.Equal(t, want.String(), got.TokenAmountOut.Amount.String(), "%s zfo=%v in=%s", lp.name, zfo,
					amountIn)
				fp.Quotes = append(fp.Quotes, fixtureQuote{zfo, amountIn.String(), want.String()})
			}

			// Out and back through the same pool: currency in -> other -> currency in.
			amountIn := amountsIn(p, zfo)[2]
			currencyIn, other := key.Currency1, key.Currency0
			if zfo {
				currencyIn, other = other, currencyIn
			}
			res, err := e.call("quoteExactInput", block, nil, pathParams{currencyIn, []pathKey{
				{other, key.Fee, key.TickSpacing, key.Hooks, []byte{}},
				{currencyIn, key.Fee, key.TickSpacing, key.Hooks, []byte{}}}, amountIn})
			require.NoError(t, err)
			cloned := sim.CloneState()
			leg1, err := cloned.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn}, TokenOut: tokenOut})
			require.NoError(t, err)
			cloned.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: tokenIn,
				Amount: amountIn}, TokenAmountOut: *leg1.TokenAmountOut, Fee: *leg1.Fee, SwapInfo: leg1.SwapInfo})
			leg2, err := cloned.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: *leg1.TokenAmountOut,
				TokenOut: tokenIn})
			require.NoError(t, err)
			t.Logf("| %s round trip | %d | %v | %s | %s | %s | |", lp.name, p.BlockNumber, zfo, amountIn, res[0],
				leg2.TokenAmountOut.Amount)
			require.Equal(t, res[0].(*big.Int).String(), leg2.TokenAmountOut.Amount.String(), "%s round trip", lp.name)
		}
		fixture = append(fixture, fp)
	}
	if os.Getenv("KALOS_WRITE_FIXTURE") != "" {
		b, err := json.Marshal(fixture)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile("testdata/pools.json", append(b, '\n'), 0o644))
	}
}

// TestLiveBurstStates: the live pools are quiet, so their tracked state exercises little of the fee engine. Here the
// hook's own storage is overridden with random volatility states (and, every third, other fee parameters) around
// the pool's tick, and both the hook's previewFeeAt and real swaps through V4Quoter at that state are compared
// with this package: the fee, its split and the accumulator, then every quote to the wei.
func TestLiveBurstStates(t *testing.T) {
	e := newLiveEnv(t)
	rng := rand.New(rand.NewPCG(20261117, 1))
	cases, quotes, dynamicFees := 0, 0, 0
	for i, lp := range livePools {
		p, key := e.track(i)
		block := new(big.Int).SetUint64(p.BlockNumber)
		base, tick := hookExtraOf(t, p)
		now := uint64(p.Timestamp)
		for n := range 16 {
			hx := base
			hx.State = randomState(rng, tick, now, hx.Params.MaxVolatilityAccumulator)
			if n%3 == 2 {
				hx.Params.Directional = rng.IntN(2) == 0
				hx.Params.SameMomentSeconds = uint64(rng.IntN(3))
				hx.Params.MaxFeeZeroForOne = min(50_000, hx.Params.MaxFeeZeroForOne*2)
			}
			overrides := map[common.Address]ethereum.OverrideAccount{HookAddresses[0]: {StateDiff: map[common.Hash]common.Hash{
				mappingSlot(lp.id, feeParamsSlot): packParams(&hx.Params),
				mappingSlot(lp.id, feeStateSlot):  packState(&hx.State),
			}}}
			sim, err := uniswapv4.NewPoolSimulator(withHookExtra(t, p, hx), valueobject.ChainIDElysium)
			require.NoError(t, err)
			for _, zfo := range []bool{true, false} {
				res, err := e.call("previewFeeAt", block, overrides, key, zfo, new(big.Int).SetUint64(now))
				require.NoError(t, err)
				q := res[0].(feeQuote)
				total, va, _, ok := hx.Params.computeFee(hx.State, tick, now, zfo)
				require.True(t, ok)
				h, lpFee := splitFee(hx.Kind, hx.FeeSide, hx.KalosShareBps, total, zfo)
				require.Equal(t, q.TotalFee.Uint64(), uint64(total), "%s case %d zfo=%v total", lp.name, n, zfo)
				require.Equal(t, q.VolatilityAccumulator.Uint64(), va, "%s case %d zfo=%v accumulator", lp.name, n, zfo)
				require.Equal(t, q.HookFee.Uint64(), uint64(h), "%s case %d zfo=%v hook fee", lp.name, n, zfo)
				require.Equal(t, q.LpFee.Uint64(), uint64(lpFee), "%s case %d zfo=%v lp fee", lp.name, n, zfo)
				cases++
				if total > hx.Params.calmFee(zfo) {
					dynamicFees++
				}

				tokenIn, tokenOut := tokensOf(p, zfo)
				for _, amountIn := range amountsIn(p, zfo)[1:4] {
					want, _ := e.quote(key, zfo, amountIn, block, overrides)
					got, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn}, TokenOut: tokenOut})
					require.NoError(t, err)
					require.Equal(t, want.String(), got.TokenAmountOut.Amount.String(),
						"%s case %d zfo=%v in=%s fee=%d", lp.name, n, zfo, amountIn, total)
					quotes++
				}
				t.Logf("%s case %d zfo=%v total=%d hook=%d lp=%d va=%d", lp.name, n, zfo, total, h, lpFee, va)
			}
		}
	}
	t.Logf("%d previewFeeAt cases (%d above the calm fee) and %d quotes equal", cases, dynamicFees, quotes)
	require.Positive(t, dynamicFees)
}

// TestLiveHookGas: the gas the hook adds to a swap, V4Quoter's gasEstimate on each Kalos pool less the same quote on
// the unhooked HYPE/USDC 0.3% pool of the same PoolManager at the same block, stays near gasFeeAssetIn/Out.
func TestLiveHookGas(t *testing.T) {
	e := newLiveEnv(t)
	plain := poolKey{common.Address{}, liveUSDC, big.NewInt(3000), big.NewInt(60), common.Address{}}
	measured := map[bool][]int64{}
	for i, lp := range livePools {
		p, key := e.track(i)
		block := new(big.Int).SetUint64(p.BlockNumber)
		hx, _ := hookExtraOf(t, p)
		for _, zfo := range []bool{true, false} {
			for _, amountIn := range amountsIn(p, zfo)[1:4] {
				plainAmount := amountIn
				if !zfo && lp.currency1 != liveUSDC {
					plainAmount = big.NewInt(100_000) // 0.1 USDC
				}
				_, hooked := e.quote(key, zfo, amountIn, block, nil)
				_, unhooked := e.quote(plain, zfo, plainAmount, block, nil)
				feeIn := hx.feeAssetIn(zfo)
				measured[feeIn] = append(measured[feeIn], int64(hooked)-int64(unhooked))
				t.Logf("%s zfo=%v feeAssetIn=%v in=%s hooked=%d unhooked=%d hook gas=%d", lp.name, zfo, feeIn,
					amountIn, hooked, unhooked, int64(hooked)-int64(unhooked))
			}
		}
	}
	for feeIn, want := range map[bool]int64{true: gasFeeAssetIn, false: gasFeeAssetOut} {
		var total int64
		for _, g := range measured[feeIn] {
			total += g
		}
		mean := total / int64(len(measured[feeIn]))
		t.Logf("feeAssetIn=%v: mean hook gas %d over %d quotes (constant %d)", feeIn, mean, len(measured[feeIn]), want)
		require.InDelta(t, mean, want, float64(mean)/4, "feeAssetIn=%v", feeIn)
	}
}

func mappingSlot(id common.Hash, slot *big.Int) common.Hash {
	return crypto.Keccak256Hash(id.Bytes(), common.BigToHash(slot).Bytes())
}

// packParams and packState are the one-slot storage words of FeeParams and FeeState: members packed from the low
// bytes, in declaration order.
func packParams(p *FeeParams) common.Hash {
	var w packer
	w.put(uint64(p.MinFeeZeroForOne), 24).put(uint64(p.MaxFeeZeroForOne), 24)
	w.put(uint64(p.MinFeeOneForZero), 24).put(uint64(p.MaxFeeOneForZero), 24)
	w.put(b2u(p.Dynamic), 8).put(b2u(p.Directional), 8)
	w.put(p.FilterPeriod, 16).put(p.DecayPeriod, 16).put(p.ReductionFactor, 16)
	w.put(p.VariableFeeControl, 24).put(p.MaxVolatilityAccumulator, 24).put(p.ResetPeriod, 16)
	w.put(p.ResetTickFilter, 24).put(p.SameMomentSeconds, 8)
	return w.hash()
}

func packState(s *FeeState) common.Hash {
	var w packer
	w.put(uint64(uint32(s.ReferenceTick))&0xffffff, 24).put(uint64(uint32(s.LastTick))&0xffffff, 24)
	w.put(uint64(uint32(s.Peak))&0xffffff, 24).put(s.VolatilityReference, 24)
	w.put(s.LastSwap, 40).put(s.ResetTime, 40).put(s.MoveTime, 40).put(s.PeakTime, 40)
	return w.hash()
}

type packer struct {
	v   big.Int
	off uint
}

func (w *packer) put(x uint64, bits uint) *packer {
	w.v.Or(&w.v, new(big.Int).Lsh(new(big.Int).SetUint64(x), w.off))
	w.off += bits
	return w
}

func (w *packer) hash() common.Hash { return common.BigToHash(&w.v) }

func b2u(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

// TestLiveStorageLayout: packParams/packState give the hook's own storage words for a live pool, so the overrides
// TestLiveBurstStates writes are the states it means.
func TestLiveStorageLayout(t *testing.T) {
	e := newLiveEnv(t)
	for i, lp := range livePools {
		p, _ := e.track(i)
		hx, _ := hookExtraOf(t, p)
		block := new(big.Int).SetUint64(p.BlockNumber)
		params, err := e.live.Eth.StorageAt(t.Context(), HookAddresses[0], mappingSlot(lp.id, feeParamsSlot), block)
		require.NoError(t, err)
		state, err := e.live.Eth.StorageAt(t.Context(), HookAddresses[0], mappingSlot(lp.id, feeStateSlot), block)
		require.NoError(t, err)
		require.Equal(t, packParams(&hx.Params), common.BytesToHash(params), lp.name)
		require.Equal(t, packState(&hx.State), common.BytesToHash(state), lp.name)
	}
}

// randomState is a volatility state the engine can reach: a reference near the tick, a peak up to the cap with its
// reference volatility below it, and times within the last ten minutes, none after the last swap.
func randomState(rng *rand.Rand, tick int32, now, maxVA uint64) FeeState {
	lastSwap := now - uint64(rng.IntN(3)*rng.IntN(200))
	vr := uint64(rng.IntN(int(maxVA) + 1))
	if rng.IntN(3) == 0 {
		vr = 0
	}
	peak := int32(vr + uint64(rng.IntN(int(maxVA-vr)+1)))
	if rng.IntN(2) == 0 {
		peak = -peak
	}
	return FeeState{
		ReferenceTick: tick + int32(rng.IntN(1201)) - 600, LastTick: tick + int32(rng.IntN(41)) - 20*int32(rng.IntN(2)),
		Peak: peak, VolatilityReference: vr, LastSwap: lastSwap, ResetTime: lastSwap - uint64(rng.IntN(300)),
		MoveTime: lastSwap - uint64(rng.IntN(2)*rng.IntN(30)), PeakTime: lastSwap - uint64(rng.IntN(150)),
	}
}
