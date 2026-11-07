package volatile

import (
	"encoding/hex"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// ChangingFeeHook (Polygon, Ethereum, Base) is a VolatilePairHook fork with the same fee math and the same
// feeConfig/feeState views, whose charged fee is clamped at 20,000 pips (ChangingFeeHook._chargedFee). Two
// fixtures, both read from archive nodes (no transaction was sent):
//
//   - changingfee_replay.json: every Swap event of the six pools that had swaps by Oct 24, 2026, 89 blocks and
//     90 swaps (hook deployed Oct 21, 2026). Per active block b: feeState at b-1 (what a tracker stores), the pool
//     price at b-1, each swap's direction, post-swap price and charged fee (the PoolManager's Swap event, never the
//     stored state), and feeState at b.
//   - changingfee_quotes.json: getFee(key) of the deployed hooks under eth_call state overrides (feeState in the
//     hook's slot 1, slot0 price in the PoolManager), at a real block each. They reach states the pools have not
//     (big moves, a stored temp fee above the ceiling) and carry the cases where the fee is clamped at 20,000.

type replayPool struct {
	Chain  int               `json:"chain"`
	Name   string            `json:"name"`
	Hook   string            `json:"hook"`
	PoolID string            `json:"poolId"`
	Config map[string]uint32 `json:"config"`
	Cases  []fixtureCase     `json:"cases"`
}

type quoteTarget struct {
	ChainID int      `json:"chainId"`
	Hook    string   `json:"hook"`
	PoolID  string   `json:"poolId"`
	Block   uint64   `json:"block"`
	Config  []uint32 `json:"config"` // feeConfig(poolId) return order
}

type quoteCase struct {
	Pool      string `json:"pool"`
	Mode      string `json:"mode"` // same: the call is in the block of the stored state; new: a later block
	AskPerm   uint32 `json:"askPerm"`
	Temp      uint32 `json:"temp"`
	TempOnBid bool   `json:"tempOnBid"`
	Top       string `json:"top"`
	Cur       string `json:"cur"`
	FeeZfo    uint32 `json:"feeZfo"`
	FeeOfz    uint32 `json:"feeOfz"`
}

func (p replayPool) feeConfig() FeeConfig {
	c := p.Config
	return FeeConfig{RestingSpread: c["restingSpread"], BackrunCaptureFraction: c["backrunCaptureFraction"],
		MidFadeRate: c["midFadeRate"], MaxMidFade: c["maxMidFade"], SkewRate: c["skewRate"],
		MaxSkewStep: c["maxSkewStep"], MaxSkew: c["maxSkew"], BackrunThreshold: c["backrunThreshold"],
		RestoreFraction: c["restoreFraction"], RestoreThreshold: c["restoreThreshold"]}
}

func (q quoteTarget) feeConfig() FeeConfig {
	c := q.Config
	return FeeConfig{RestingSpread: c[0], BackrunCaptureFraction: c[1], MidFadeRate: c[2], MaxMidFade: c[3],
		SkewRate: c[4], MaxSkewStep: c[5], MaxSkew: c[6], BackrunThreshold: c[7], RestoreFraction: c[8],
		RestoreThreshold: c[9]}
}

func readJSON(t testing.TB, path string, dst any) {
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, dst))
}

// newHookAt builds the hook the way the pool simulator does: through the registered factory, from the tracker's hX.
func newHookAt(t testing.TB, addr common.Address, cfg FeeConfig, state FeeState) *Hook {
	t.Helper()
	// a pointer, as Track marshals the hook: uint256.Int encodes as a decimal string only when addressable
	extra, err := json.Marshal(&Extra{Config: cfg, State: state})
	require.NoError(t, err)
	hook, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(extra)})
	require.True(t, ok, "hook %s is not registered", addr)
	h, ok := hook.(*Hook)
	require.True(t, ok)
	return h
}

func TestChangingFee_Registered(t *testing.T) {
	require.Len(t, ChangingFeeHookAddresses, 3)
	for _, addr := range ChangingFeeHookAddresses {
		h := newHookAt(t, addr, liveConfig, FeeState{})
		require.EqualValues(t, ChangingFeeMaxFee, h.MaxFee, addr)
		require.EqualValues(t, 20_000, h.maxFee())
		require.Equal(t, DexType, h.GetExchange())
		// the address flags carry beforeSwap, so an unregistered address would make the pool unsupported
		require.True(t, uniswapv4.HasSwapPermissions(addr), addr)
		require.True(t, h.CanBeforeSwap(addr), addr)
		require.False(t, h.CanAfterSwap(addr), addr)
	}
	// Uniswap's own hook keeps its ceiling, and a Hook built without the factory too
	require.EqualValues(t, 0, newHookAt(t, HookAddress, liveConfig, FeeState{}).MaxFee)
	require.EqualValues(t, maxFeeE6, newHookAt(t, HookAddress, liveConfig, FeeState{}).maxFee())
	require.EqualValues(t, maxFeeE6, (&Hook{}).maxFee())
}

// The pool simulator refuses a pool whose hook has swap permissions and is not registered. Registering the three
// addresses is what makes a ChangingFeeHook pool quotable; the same pool under an unregistered address with the same
// flags (0x2080) is still refused.
func TestChangingFee_PoolSimulatorAcceptsOurHooks(t *testing.T) {
	raw, err := os.ReadFile("testdata/pool.json")
	require.NoError(t, err)
	var ep entity.Pool
	require.NoError(t, json.Unmarshal(raw, &ep))
	var extra map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(ep.Extra), &extra))
	extra["hX"] = json.RawMessage(benchHookExtra)
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	ep.Extra = string(b)

	withHook := func(addr common.Address) (*uniswapv4.PoolSimulator, error) {
		p := ep
		p.StaticExtra = strings.Replace(ep.StaticExtra, strings.ToLower(HookAddress.Hex()), strings.ToLower(addr.Hex()), 1)
		return uniswapv4.NewPoolSimulator(p, valueobject.ChainIDEthereum)
	}
	for _, addr := range ChangingFeeHookAddresses {
		sim, err := withHook(addr)
		require.NoError(t, err, addr)
		require.Equal(t, DexType, sim.GetExchange())
	}
	unregistered := common.HexToAddress("0x1111111111111111111111111111111111112080")
	require.True(t, uniswapv4.HasSwapPermissions(unregistered))
	_, err = withHook(unregistered)
	require.ErrorIs(t, err, pool.ErrUnsupported)
}

// Every swap of our pools recorded since the hook went live: the fee the code computes from the stored state and
// the pool price equals the fee the Swap event says was charged, and the first swap of a block leaves the state the
// hook persisted on chain. Same method as TestBeforeSwap_ReproducesOnchainFees.
func TestChangingFee_ReplaysOnchainFees(t *testing.T) {
	var f struct {
		Pools []replayPool `json:"pools"`
	}
	readJSON(t, "testdata/changingfee_replay.json", &f)
	require.NotEmpty(t, f.Pools)

	var swaps, blocks, multi, tempOn int
	for _, p := range f.Pools {
		addr := common.HexToAddress(p.Hook)
		cfg := p.feeConfig()
		for _, c := range p.Cases {
			blocks++
			h := newHookAt(t, addr, cfg, c.State.feeState(t))
			price := uint256.MustFromDecimal(c.PriceBefore)
			for i, sw := range c.Swaps {
				res := beforeSwap(t, h, price, sw.Zfo)
				require.Equal(t, sw.Fee, uint32(res.SwapFee), "%s block %d swap %d", p.Name, c.Block, i)
				swaps++
				if i == 0 {
					info, ok := res.SwapInfo.(*SwapInfo)
					require.True(t, ok, "%s block %d", p.Name, c.Block)
					require.Equal(t, c.After.feeState(t), info.State, "%s block %d advanced state", p.Name, c.Block)
					if info.State.TempFee > 0 {
						tempOn++
					}
				} else {
					multi++
					require.Nil(t, res.SwapInfo, "%s block %d: no second advance in a block", p.Name, c.Block)
				}
				h.UpdateBalance(res.SwapInfo)
				price = uint256.MustFromDecimal(sw.Post)
			}
		}
	}
	t.Logf("pools %d, blocks %d, swaps %d matched (later legs %d, blocks with a temp fee %d)",
		len(f.Pools), blocks, swaps, multi, tempOn)
}

// getFee of the deployed hooks under overridden states. In "same" mode the stored state is of the call's own block, so
// the quote is the in-block displacement fee; in "new" mode a later block, so the quote includes the top-of-block
// advance. 29 of the 180 quoted fees are the 20,000 ceiling; for each of those the code under Uniswap's address
// (999,998) computes a fee above the ceiling: the per-address ceiling is what makes them equal. The recorded swaps
// above never reached 2% (the highest fee charged was 1,425 pips), so these quotes are the proof of the ceiling.
func TestChangingFee_QuotesEqualGetFee(t *testing.T) {
	var f struct {
		Targets map[string]quoteTarget `json:"targets"`
		Cases   []quoteCase            `json:"cases"`
	}
	readJSON(t, "testdata/changingfee_quotes.json", &f)
	require.NotEmpty(t, f.Cases)

	var atCeiling, aboveWithoutIt int
	for _, c := range f.Cases {
		tgt, ok := f.Targets[c.Pool]
		require.True(t, ok, c.Pool)
		cfg := tgt.feeConfig()
		state := FeeState{AskPermFee: c.AskPerm, TempFee: c.Temp, TempOnBid: c.TempOnBid}
		require.NoError(t, state.SqrtPriceTop.SetFromDecimal(c.Top))
		cur := uint256.MustFromDecimal(c.Cur)

		ours := newHookAt(t, common.HexToAddress(tgt.Hook), cfg, state)
		uniswaps := newHookAt(t, HookAddress, cfg, state)
		ours.advanced = c.Mode == "same"
		uniswaps.advanced = ours.advanced

		for _, d := range []struct {
			zfo  bool
			want uint32
		}{{true, c.FeeZfo}, {false, c.FeeOfz}} {
			got := uint32(beforeSwap(t, ours, cur, d.zfo).SwapFee)
			require.Equal(t, d.want, got, "%s %s zfo=%v top=%s cur=%s", c.Pool, c.Mode, d.zfo, c.Top, c.Cur)

			unclamped := uint32(beforeSwap(t, uniswaps, cur, d.zfo).SwapFee)
			if d.want == ChangingFeeMaxFee {
				atCeiling++
				require.GreaterOrEqual(t, unclamped, ChangingFeeMaxFee, "%s %s", c.Pool, c.Mode)
				if unclamped > ChangingFeeMaxFee {
					aboveWithoutIt++
				}
			} else {
				require.Equal(t, d.want, unclamped, "below the ceiling both addresses quote the same")
			}
		}
	}
	require.NotZero(t, atCeiling)
	require.NotZero(t, aboveWithoutIt, "without a per-address ceiling some quotes would be wrong")
	t.Logf("quotes %d (%d fees), %d at the ceiling, %d of those above it under Uniswap's address",
		len(f.Cases), 2*len(f.Cases), atCeiling, aboveWithoutIt)
}

// The same state quoted under both ceilings, from an extreme one-block move on the yen pool: Uniswap's hook charges
// far more than 2%, ChangingFeeHook exactly 2%. Neither side's baseline changes (the library clamps stay at 999,998).
func TestChangingFee_ExtremeMoveIsClampedAtTwoPercent(t *testing.T) {
	cfg := liveConfig
	cfg.RestingSpread = 650
	cfg.MaxSkew = 350
	cfg.MaxSkewStep = 50
	cfg.MaxMidFade = 200
	cfg.BackrunThreshold = 2000
	cfg.RestoreThreshold = 2000
	var state FeeState
	require.NoError(t, state.SqrtPriceTop.SetFromDecimal("994684633017934431386376699516896839"))
	state.AskPermFee = 325
	// the pool price fell 10% in sqrt terms inside the block since the open: the buyer's side is the backrun
	cur := uint256.MustFromDecimal("895216169716140988247739029565207155")

	ours := newHookAt(t, ChangingFeeHookAddresses[0], cfg, state)
	uniswaps := newHookAt(t, HookAddress, cfg, state)
	ours.advanced, uniswaps.advanced = true, true
	buyOurs := beforeSwap(t, ours, cur, false).SwapFee
	buyUniswaps := beforeSwap(t, uniswaps, cur, false).SwapFee
	require.EqualValues(t, ChangingFeeMaxFee, buyOurs)
	require.Greater(t, uint32(buyUniswaps), ChangingFeeMaxFee)
	require.LessOrEqual(t, uint32(buyUniswaps), uint32(maxFeeE6))

	// the continuing side pays baseline + maxMidFade under both
	sellOurs := beforeSwap(t, ours, cur, true).SwapFee
	require.EqualValues(t, beforeSwap(t, uniswaps, cur, true).SwapFee, sellOurs)
	t.Logf("buy side: ChangingFeeHook %d, Uniswap's VolatilePairHook %d; sell side %d under both", buyOurs, buyUniswaps, sellOurs)
}

// feeConfig and feeState of the deployed Polygon hook (USDC/JPYC pool, block 96,111,141) decode with the ABI this
// package already uses for Uniswap's hook, into the same tagged structs Track uses: same selectors, names, order.
func TestChangingFee_ABIDecodesOurViews(t *testing.T) {
	require.Equal(t, "b9629a41", hex.EncodeToString(volatilePairHookABI.Methods["feeConfig"].ID))
	require.Equal(t, "0d1fd58e", hex.EncodeToString(volatilePairHookABI.Methods["feeState"].ID))

	feeConfigRet, err := hex.DecodeString("000000000000000000000000000000000000000000000000000000000000028a" +
		"00000000000000000000000000000000000000000000000000000000000186a0" +
		"00000000000000000000000000000000000000000000000000000000000f4240" +
		"00000000000000000000000000000000000000000000000000000000000000c8" +
		"000000000000000000000000000000000000000000000000000000000007a120" +
		"0000000000000000000000000000000000000000000000000000000000000032" +
		"000000000000000000000000000000000000000000000000000000000000015e" +
		"00000000000000000000000000000000000000000000000000000000000007d0" +
		"00000000000000000000000000000000000000000000000000000000000f4240" +
		"00000000000000000000000000000000000000000000000000000000000007d0")
	require.NoError(t, err)
	var cfg struct {
		RestingSpread          *big.Int `abi:"restingSpreadE6"`
		BackrunCaptureFraction *big.Int `abi:"backrunCaptureFractionE6"`
		MidFadeRate            *big.Int `abi:"midFadeRateE6"`
		MaxMidFade             *big.Int `abi:"maxMidFadeE6"`
		SkewRate               *big.Int `abi:"skewRateE6"`
		MaxSkewStep            *big.Int `abi:"maxSkewStepE6"`
		MaxSkew                *big.Int `abi:"maxSkewE6"`
		BackrunThreshold       *big.Int `abi:"backrunThresholdE6"`
		RestoreFraction        *big.Int `abi:"restoreFractionE6"`
		RestoreThreshold       *big.Int `abi:"restoreThresholdE6"`
	}
	require.NoError(t, volatilePairHookABI.UnpackIntoInterface(&cfg, "feeConfig", feeConfigRet))
	require.EqualValues(t, []int64{650, 100000, 1000000, 200, 500000, 50, 350, 2000, 1000000, 2000},
		[]int64{cfg.RestingSpread.Int64(), cfg.BackrunCaptureFraction.Int64(), cfg.MidFadeRate.Int64(),
			cfg.MaxMidFade.Int64(), cfg.SkewRate.Int64(), cfg.MaxSkewStep.Int64(), cfg.MaxSkew.Int64(),
			cfg.BackrunThreshold.Int64(), cfg.RestoreFraction.Int64(), cfg.RestoreThreshold.Int64()})

	feeStateRet, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000005ba6e49" +
		"0000000000000000000000000000000000bf9dc17d0f96b45a92b3d462109aef" +
		"000000000000000000000000000000000000000000000000000000000000015e" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000001")
	require.NoError(t, err)
	var state struct {
		BlockNumber  *big.Int `abi:"blockNumber"`
		SqrtPriceTop *big.Int `abi:"sqrtPriceTopX96"`
		AskPermFee   *big.Int `abi:"askPermFeeE6"`
		TempFee      *big.Int `abi:"tempFeeE6"`
		TempOnBid    bool     `abi:"tempOnBid"`
	}
	require.NoError(t, volatilePairHookABI.UnpackIntoInterface(&state, "feeState", feeStateRet))
	require.EqualValues(t, 96104009, state.BlockNumber.Int64())
	require.Equal(t, "994928368027762439198233348807891695", state.SqrtPriceTop.String())
	require.EqualValues(t, 350, state.AskPermFee.Int64())
	require.EqualValues(t, 0, state.TempFee.Int64())
	require.True(t, state.TempOnBid)
}
