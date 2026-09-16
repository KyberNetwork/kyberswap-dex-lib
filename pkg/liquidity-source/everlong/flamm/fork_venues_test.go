package everlongflamm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Router venue admission, LP flows and same-block fills on an anvil fork of Base through the EverlongFlammAdapter
// artifact (EVERLONG_FLAMM_FORK_RPC, EVERLONG_ADAPTER_OUT; EVERLONG_FLAMM_FORK_BLOCK overrides the default block
// 51336630, a block right after a cbBTC round; EVERLONG_FLAMM_FORK_SCENARIOS, comma separated, runs a subset and
// EVERLONG_FLAMM_FORK_PHASE2=1 compares each scenario again after its sequence). Beyond fork_edges_test.go and
// fork_scenarios_test.go:
//
//   - a second Router venue (a fresh cbBTC/USDC Morpho market on the same oracle and IRM, admitted through the
//     curator's scheduled addVenue): the lister must relist it, the old listing must drift, and the two-venue pool
//     must then quote exactly through the adapter under several priority orders, a thin second market, a tight
//     second debt cap, a supply-only second venue, a refinance and a supply migration between the venues;
//   - LP flows (depositProRata, redeemProRata), the emergency lane and maintain between refreshes;
//   - several fills through the pool mined in ONE block (the executor's multi-hop / split route case: every fill at
//     the same timestamp, the simulator advanced by UpdateBalance only), then a refresh equal to the state.

const forkVenuesBlock = "51336630"

var venuesABI = func() abi.ABI {
	const mp = `{"name":"marketParams","type":"tuple","components":[{"name":"loanToken","type":"address"},{"name":"collateralToken","type":"address"},{"name":"oracle","type":"address"},{"name":"irm","type":"address"},{"name":"lltv","type":"uint256"}]}`
	a, err := abi.JSON(strings.NewReader(`[
 {"type":"function","name":"createMarket","stateMutability":"nonpayable","inputs":[` + mp + `],"outputs":[]},
 {"type":"function","name":"idToMarketParams","stateMutability":"view","inputs":[{"name":"id","type":"bytes32"}],"outputs":[{"name":"loanToken","type":"address"},{"name":"collateralToken","type":"address"},{"name":"oracle","type":"address"},{"name":"irm","type":"address"},{"name":"lltv","type":"uint256"}]},
 {"type":"function","name":"addVenue","stateMutability":"nonpayable","inputs":[{"name":"v","type":"tuple","components":[{"name":"kind","type":"uint8"},{"name":"loanIndex","type":"uint8"},{"name":"venueParams","type":"bytes"},{"name":"borrowEnabled","type":"bool"},{"name":"supplyEnabled","type":"bool"},{"name":"debtCap","type":"uint128"},{"name":"supplyCap","type":"uint128"},{"name":"maxBorrowRateWad","type":"uint64"}]}],"outputs":[{"name":"","type":"uint16"}]},
 {"type":"function","name":"setPriorities","stateMutability":"nonpayable","inputs":[{"name":"b","type":"uint16[]"},{"name":"s","type":"uint16[]"},{"name":"w","type":"uint16[]"},{"name":"r","type":"uint16[]"}],"outputs":[]},
 {"type":"function","name":"maintain","stateMutability":"nonpayable","inputs":[{"name":"kind","type":"uint8"},{"name":"a","type":"uint16"},{"name":"b","type":"uint16"},{"name":"amount","type":"uint256"}],"outputs":[]},
 {"type":"function","name":"emergency","stateMutability":"nonpayable","inputs":[{"name":"kind","type":"uint8"},{"name":"id","type":"uint16"},{"name":"assets","type":"uint256"}],"outputs":[]},
 {"type":"function","name":"depositProRata","stateMutability":"nonpayable","inputs":[{"name":"poolAssets","type":"uint256"},{"name":"receiver","type":"address"},{"name":"maxIn","type":"uint256[]"},{"name":"minOut","type":"uint256[]"},{"name":"minShares","type":"uint256"},{"name":"deadline","type":"uint256"}],"outputs":[{"name":"","type":"uint256"},{"name":"","type":"uint256"},{"name":"","type":"uint256"}]},
 {"type":"function","name":"redeemProRata","stateMutability":"nonpayable","inputs":[{"name":"shares","type":"uint256"},{"name":"receiver","type":"address"},{"name":"owner","type":"address"},{"name":"maxIn","type":"uint256[]"},{"name":"minOut","type":"uint256[]"},{"name":"minPool","type":"uint256"},{"name":"deadline","type":"uint256"}],"outputs":[{"name":"","type":"uint256"},{"name":"","type":"uint256"},{"name":"","type":"uint256"}]},
 {"type":"function","name":"balanceOf","stateMutability":"view","inputs":[{"name":"who","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
 {"type":"function","name":"depositAllowlist","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
 {"type":"function","name":"enabled","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"bool"}]},
 {"type":"function","name":"setDepositorAllowed","stateMutability":"nonpayable","inputs":[{"name":"d","type":"address"},{"name":"a","type":"bool"}],"outputs":[]}
]`))
	if err != nil {
		panic(err)
	}
	return a
}()

type venueInit struct {
	Kind             uint8
	LoanIndex        uint8
	VenueParams      []byte
	BorrowEnabled    bool
	SupplyEnabled    bool
	DebtCap          *big.Int
	SupplyCap        *big.Int
	MaxBorrowRateWad uint64
}

// venuesEnv is the shared cast of a fork_venues_test.go run.
type venuesEnv struct {
	*forkEnv
	se                             StaticExtra
	curator, keeper, anyone, whale common.Address
	params0                        morphoMarketParams
	morpho                         common.Address
}

func newVenuesEnv(t *testing.T) *venuesEnv {
	e := newForkEnvAt(t, forkVenuesBlock)
	v := &venuesEnv{forkEnv: e, morpho: c104.Morpho}
	require.NoError(t, json.Unmarshal([]byte(e.listed.StaticExtra), &v.se))
	core := e.f.view(e.pool, "core")[0].(common.Address)
	v.curator = e.f.view(core, "owner")[0].(common.Address)
	v.keeper = e.f.view(core, "keeper")[0].(common.Address)
	v.anyone = common.HexToAddress("0x00000000000000000000000000000000000beef2")
	v.whale = common.HexToAddress("0x00000000000000000000000000000000000beef3")
	for _, a := range []common.Address{v.curator, v.keeper, v.anyone, v.whale} {
		e.f.impersonate(a)
	}
	v0 := entityVenues(t, e.listed)[0]
	v.params0 = morphoMarketParams{v.se.LoanAsset, v.se.PoolAsset, v0.Oracle, v0.Irm, v0.Lltv.ToBig()}
	return v
}

// fund adds amount of token to who and approves spender.
func (v *venuesEnv) fund(t *testing.T, who, token, spender common.Address, amount *big.Int) bool {
	cur := v.f.balance(token, who)
	v.f.setBalance(token, who, new(big.Int).Add(cur, amount))
	return v.sendTx(t, who, token, &scenariosABI, "approve", spender, amount)
}

func marketID(p morphoMarketParams) common.Hash {
	var buf [160]byte
	copy(buf[12:32], p.LoanToken[:])
	copy(buf[44:64], p.CollateralToken[:])
	copy(buf[76:96], p.Oracle[:])
	copy(buf[108:128], p.Irm[:])
	p.Lltv.FillBytes(buf[128:160])
	return crypto.Keccak256Hash(buf[:])
}

func encodeMarketParams(p morphoMarketParams) []byte {
	var buf [160]byte
	copy(buf[12:32], p.LoanToken[:])
	copy(buf[44:64], p.CollateralToken[:])
	copy(buf[76:96], p.Oracle[:])
	copy(buf[108:128], p.Irm[:])
	p.Lltv.FillBytes(buf[128:160])
	return buf[:]
}

// zeroGovernanceDelay clears FLAMMStore.governanceDelaySec (base+18, bits 176..208) so a scheduled venue can be
// executed in the next block; the field prices nothing on the swap path.
func (v *venuesEnv) zeroGovernanceDelay(t *testing.T) {
	slot := flammSlot(18)
	var w hexutil.Bytes
	v.f.rpc(&w, "eth_getStorageAt", v.pool, slot, "latest")
	word := new(big.Int).SetBytes(w)
	mask := new(big.Int).Lsh(new(big.Int).SetUint64(0xffffffff), 176)
	word.AndNot(word, mask)
	v.f.rpc(nil, "anvil_setStorageAt", v.pool, slot, common.BigToHash(word))
	sw := v.viewCall(t, v.pool, &flammABI, "switches")
	require.EqualValues(t, 0, sw[1], "governance delay cleared")
}

// addVenue2 creates (or reuses) the cbBTC/USDC market at lltv on venue 0's oracle and IRM, seeds liquidity USDC and
// admits it as Router venue 1.
func (v *venuesEnv) addVenue2(t *testing.T, lltv uint64, liquidity int64, borrow, supply bool, debtCap, rateCap uint64) bool {
	p := v.params0
	p.Lltv = new(big.Int).SetUint64(lltv)
	id := marketID(p)
	got := v.viewCall(t, v.morpho, &venuesABI, "idToMarketParams", id)
	if got[4].(*big.Int).Sign() == 0 && got[0].(common.Address) == (common.Address{}) {
		if !v.sendTx(t, v.anyone, v.morpho, &venuesABI, "createMarket", p) {
			return false
		}
	} else {
		t.Logf("market %s already exists", id)
	}
	if liquidity > 0 {
		a := big.NewInt(liquidity)
		if !v.fund(t, v.whale, v.se.LoanAsset, v.morpho, a) ||
			!v.sendTx(t, v.whale, v.morpho, &scenariosABI, "supply", p, a, big.NewInt(0), v.whale, []byte{}) {
			return false
		}
	}
	v.zeroGovernanceDelay(t)
	init := venueInit{Kind: 0, LoanIndex: 0, VenueParams: encodeMarketParams(p), BorrowEnabled: borrow,
		SupplyEnabled: supply, DebtCap: new(big.Int).SetUint64(debtCap), SupplyCap: big.NewInt(0),
		MaxBorrowRateWad: rateCap}
	// the first call schedules, the second (the delay cleared) executes
	for range 2 {
		if !v.sendTx(t, v.curator, v.pool, &venuesABI, "addVenue", init) {
			return false
		}
	}
	n := v.viewCall(t, v.se.Router, &routerABI, "venueCount", v.pool)[0].(*big.Int)
	t.Logf("venue count now %s (market %s lltv %d liquidity %d)", n, id, lltv, liquidity)
	return n.Cmp(big.NewInt(2)) == 0
}

func (v *venuesEnv) priorities(t *testing.T, b, s, w, r []uint16) bool {
	return v.sendTx(t, v.curator, v.pool, &venuesABI, "setPriorities", b, s, w, r)
}

type venuesScenario struct {
	name   string
	apply  func(t *testing.T, v *venuesEnv) bool
	relist bool
	drift  []uint64
}

func TestForkVenuesAndFlows(t *testing.T) {
	v := newVenuesEnv(t)
	e := v.forkEnv
	rep := newForkReport(t)
	_, md0, err := NewPoolsListUpdater(baseConfig(), e.f.client).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	listed0 := e.listed

	e.armLeverage(t, 13_000, 0)
	base, _, err := e.tryTrack(t, Policy{LeverRouting: true})
	require.NoError(t, err)
	ts := base.state.Timestamp
	for i, st := range []struct {
		venue uint8
		sell  bool
		a     uint64
	}{{1, true, 9_000}, {0, true, 40_000}, {1, true, 4_000}} {
		in, out := e.tokens(st.sell)
		fl := e.mineFill(t, st.venue, in, out, new(big.Int).SetUint64(st.a), ts+uint64(4*(i+1)))
		t.Logf("setup fill %d: %s gas %d", i, describeFill(fl), fl.gas)
	}
	base, _, err = e.tryTrack(t, Policy{LeverRouting: true})
	require.NoError(t, err)
	vp, err := base.state.Router.venuePosition(0, base.state.Timestamp)
	require.NoError(t, err)
	t.Logf("base venue 0: collateral %s supplied %s debt %s liquid %s", vp[0].Dec(), vp[2].Dec(), vp[4].Dec(),
		base.state.Pool.Loans[0].Liquid.Dec())
	all := []uint16{0, 1}
	first := []uint16{1, 0}
	const (
		lltv915 = 915_000_000_000_000_000
		lltv945 = 945_000_000_000_000_000
		lltv77  = 770_000_000_000_000_000
	)

	scenarios := []venuesScenario{
		{name: "venue2-915-default-order", relist: true, drift: []uint64{700}, apply: func(t *testing.T, v *venuesEnv) bool {
			return v.addVenue2(t, lltv915, 20_000_000_000, true, true, 0, 0)
		}},
		{name: "venue2-945-first", relist: true, drift: []uint64{900}, apply: func(t *testing.T, v *venuesEnv) bool {
			return v.addVenue2(t, lltv945, 20_000_000_000, true, true, 0, 0) && v.priorities(t, first, first, first, first)
		}},
		{name: "venue2-thin-first", relist: true, apply: func(t *testing.T, v *venuesEnv) bool {
			return v.addVenue2(t, lltv915, 5_000_000, true, true, 0, 0) && v.priorities(t, first, first, all, first)
		}},
		{name: "venue2-debtcap-first", relist: true, apply: func(t *testing.T, v *venuesEnv) bool {
			return v.addVenue2(t, lltv945, 20_000_000_000, true, true, 3_000_000, 0) &&
				v.priorities(t, first, all, first, all)
		}},
		{name: "venue2-supply-only-first", relist: true, apply: func(t *testing.T, v *venuesEnv) bool {
			return v.addVenue2(t, lltv77, 0, false, true, 0, 0) && v.priorities(t, all, first, first, first)
		}},
		{name: "venue2-refinance-migrate", relist: true, drift: []uint64{1200}, apply: func(t *testing.T, v *venuesEnv) bool {
			if !v.addVenue2(t, lltv915, 20_000_000_000, true, true, 0, 0) || !v.priorities(t, first, first, first, first) {
				return false
			}
			full := new(big.Int).Lsh(vp[4].ToBig(), 1)
			okRef := v.sendTx(t, v.keeper, v.pool, &venuesABI, "maintain", uint8(2), uint16(0), uint16(1), full)
			okMig := false
			if vp[2].Sign() > 0 {
				okMig = v.sendTx(t, v.keeper, v.pool, &venuesABI, "maintain", uint8(3), uint16(0), uint16(1),
					new(big.Int).Rsh(vp[2].ToBig(), 1))
			}
			t.Logf("refinance %v migrate %v", okRef, okMig)
			return okRef || okMig
		}},
		{name: "lp-deposit-prorata", apply: func(t *testing.T, v *venuesEnv) bool {
			return v.depositProRata(t, 50_000_000)
		}},
		{name: "lp-deposit-then-redeem-prorata", drift: []uint64{600}, apply: func(t *testing.T, v *venuesEnv) bool {
			if !v.depositProRata(t, 30_000_000) {
				return false
			}
			shares := v.viewCall(t, v.pool, &venuesABI, "balanceOf", v.whale)[0].(*big.Int)
			part := new(big.Int).Div(new(big.Int).Mul(shares, big.NewInt(7)), big.NewInt(10))
			huge := new(big.Int).Lsh(big.NewInt(1), 200)
			if !v.fund(t, v.whale, v.se.LoanAsset, v.pool, big.NewInt(1_000_000_000_000)) {
				return false
			}
			return v.sendTx(t, v.whale, v.pool, &venuesABI, "redeemProRata", part, v.whale, v.whale, []*big.Int{huge},
				[]*big.Int{big.NewInt(0)}, big.NewInt(0), big.NewInt(1<<40))
		}},
		{name: "emergency-repay", apply: func(t *testing.T, v *venuesEnv) bool {
			a := big.NewInt(13_000_000)
			return v.fund(t, v.curator, v.se.LoanAsset, v.pool, a) &&
				v.sendTx(t, v.curator, v.pool, &venuesABI, "emergency", uint8(0), uint16(0), a)
		}},
		{name: "emergency-full-repay-withdraw-collateral", apply: func(t *testing.T, v *venuesEnv) bool {
			a := new(big.Int).Add(vp[4].ToBig(), big.NewInt(1_000_000))
			return v.fund(t, v.curator, v.se.LoanAsset, v.pool, a) &&
				v.sendTx(t, v.curator, v.pool, &venuesABI, "emergency", uint8(0), uint16(0), a) &&
				v.sendTx(t, v.curator, v.pool, &venuesABI, "emergency", uint8(3), uint16(0), big.NewInt(20_000))
		}},
		{name: "emergency-withdraw-supplied", apply: func(t *testing.T, v *venuesEnv) bool {
			// a large buy repays the debt and supplies the rest, then the curator pulls half the supply back
			in, out := v.tokens(false)
			fl := v.mineFill(t, VenueSwap, in, out, big.NewInt(150_000_000), v.f.head().Time+2)
			pos := v.viewCall(t, v.se.Router, &routerABI, "venuePosition", v.pool, uint16(0))
			t.Logf("buy %s; venue 0 position %v", describeFill(fl), pos)
			sup := pos[2].(*big.Int)
			if fl.reverted || sup.Sign() == 0 {
				return false
			}
			return v.sendTx(t, v.curator, v.pool, &venuesABI, "emergency", uint8(2), uint16(0), new(big.Int).Rsh(sup, 1))
		}},
		{name: "emergency-topup", apply: func(t *testing.T, v *venuesEnv) bool {
			return v.sendTx(t, v.curator, v.pool, &venuesABI, "emergency", uint8(1), uint16(0), big.NewInt(5_000))
		}},
		// The emergency collateral withdrawal runs only with no debt outstanding (MorphoBlueAccount.sol:272), which the
		// setup fills leave: emergency-full-repay-withdraw-collateral repays first.
		{name: "maintain-topup", apply: func(t *testing.T, v *venuesEnv) bool {
			return v.sendTx(t, v.keeper, v.pool, &venuesABI, "maintain", uint8(0), uint16(0), uint16(0), big.NewInt(7_000))
		}},
		{name: "maintain-release", apply: func(t *testing.T, v *venuesEnv) bool {
			return v.sendTx(t, v.keeper, v.pool, &venuesABI, "maintain", uint8(0), uint16(0), uint16(0), big.NewInt(60_000)) &&
				v.sendTx(t, v.keeper, v.pool, &venuesABI, "maintain", uint8(1), uint16(0), uint16(0), big.NewInt(30_000))
		}},
	}

	var snap hexutil.Big
	e.f.rpc(&snap, "evm_snapshot")
	only := os.Getenv("EVERLONG_FLAMM_FORK_SCENARIOS")
	for si, sc := range scenarios {
		if only != "" && !strings.Contains(","+only+",", ","+sc.name+",") {
			continue
		}
		var ok bool
		e.f.rpc(&ok, "evm_revert", snap)
		require.True(t, ok)
		e.f.rpc(&snap, "evm_snapshot")
		e.listed = listed0
		if !sc.apply(t, v) {
			rep.mismatch(sc.name+"/not-applied", "scenario setup", "could not be applied", "every scenario applies")
			continue
		}
		if sc.relist {
			// The pool gained a venue. The entity pool-service holds must heal on its next refresh, without a
			// relisting: the refresh re-reads the venue set, publishes it in Extra and attests the pool again.
			// The lister must not emit the pool -- its identity (StaticExtra) did not change, and pool-service
			// does not overwrite an existing pool's StaticExtra anyway.
			tracked, err := NewPoolTracker(parityConfig(Policy{LeverRouting: true}), e.f.client).
				GetNewPoolState(context.Background(), e.listed, pool.GetNewPoolStateParams{})
			var extra Extra
			if err == nil {
				require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
			}
			if _, simErr := NewPoolSimulator(tracked); err != nil || simErr != nil || extra.ProfileDrift != "" ||
				!extra.Attested || len(extra.Venues) != 2 {
				rep.mismatch(sc.name+"/self-heal", "the listed entity after addVenue",
					fmt.Sprintf("err %v drift %q attested %v venues %d", err, extra.ProfileDrift, extra.Attested,
						len(extra.Venues)), "expected a refresh that adopts the second venue and attests")
				continue
			}
			t.Logf("%s: the refresh adopted %d venues without relisting", sc.name, len(extra.Venues))
			pools, _, err := NewPoolsListUpdater(baseConfig(), e.f.client).GetNewPools(context.Background(), md0)
			if err != nil || len(pools) != 0 {
				rep.mismatch(sc.name+"/relist", "lister with the old cursor", fmt.Sprintf("%d pools err %v", len(pools), err),
					"expected no relisting: the pool's identity did not change")
			}
			e.listed = tracked // pool-service keeps the refreshed entity; the next refresh starts from its venue set
		}
		sim, extra, err := e.trackDonationAware(t, rep, sc.name)
		if err != nil {
			fills := e.chainFills(t, e.f.head().Time)
			t.Logf("scenario %s: simulator refuses the pool: %v; chain fills on a small grid: %v", sc.name, err, fills)
			if extra != nil {
				t.Logf("  attest %q drift %q", extra.AttestFailure, extra.ProfileDrift)
			}
			if len(fills) > 0 {
				rep.mismatch(sc.name+"/pool-refused", "refresh at head", err.Error(), fmt.Sprintf("chain fills %v", fills))
			} else {
				rep.count(sc.name + "/pool-refused-chain-refuses")
			}
			continue
		}
		for i := range sim.state.Router.Venues {
			p, err := sim.state.Router.venuePosition(uint16(i), sim.state.Timestamp)
			require.NoError(t, err)
			t.Logf("%s: venue %d position coll %s supplied %s debt %s", sc.name, i, p[0].Dec(), p[2].Dec(), p[4].Dec())
		}
		e.edgeSuite(t, rep, sc.name, sim, true, 4)
		for _, d := range sc.drift {
			at := sim.state.Timestamp + d
			for _, venue := range []int{int(VenueSwap), int(VenueLever), -1} {
				for _, sell := range []bool{true, false} {
					amounts := liveGrid(1, 3_000_000, 24)
					if !sell {
						amounts = liveGrid(500, 3_000_000_000, 24)
					}
					e.compareFills(t, rep, fmt.Sprintf("%s/drift+%d/venue%d/sell=%v", sc.name, d, venue, sell), sim, venue,
						sell, amounts, at)
				}
			}
		}
		e.venuesSeq(t, rep, sc.name, sim, 40, int64(0x3F1A)+int64(si))
		if os.Getenv("EVERLONG_FLAMM_FORK_PHASE2") == "" {
			continue
		}
		// phase 2: the sequence moved the book and the venues' positions; compare the evolved state again
		sim2, _, err := e.trackDonationAware(t, rep, sc.name+"/phase2")
		if err != nil {
			rep.mismatch(sc.name+"/phase2", "refresh after the sequence", "simulator refuses", err.Error())
			continue
		}
		for i := range sim2.state.Router.Venues {
			p, err := sim2.state.Router.venuePosition(uint16(i), sim2.state.Timestamp)
			require.NoError(t, err)
			t.Logf("%s phase 2: venue %d position coll %s supplied %s debt %s", sc.name, i, p[0].Dec(), p[2].Dec(),
				p[4].Dec())
		}
		e.edgeSuite(t, rep, sc.name+"/phase2", sim2, true, 3)
		e.venuesSeq(t, rep, sc.name+"/phase2", sim2, 30, int64(0x7F1A)+int64(si))
	}
	require.Empty(t, rep.list)
}

// depositProRata deposits poolAssets cbBTC (and whatever loan leg the pool asks) as the whale.
func (v *venuesEnv) depositProRata(t *testing.T, poolAssets int64) bool {
	al := v.viewCall(t, v.pool, &venuesABI, "depositAllowlist")[0].(common.Address)
	if al != (common.Address{}) {
		if en := v.viewCall(t, al, &venuesABI, "enabled")[0].(bool); en {
			if !v.sendTx(t, v.curator, al, &venuesABI, "setDepositorAllowed", v.whale, true) {
				return false
			}
		}
	}
	a := big.NewInt(poolAssets)
	if !v.fund(t, v.whale, v.se.PoolAsset, v.pool, a) || !v.fund(t, v.whale, v.se.LoanAsset, v.pool, big.NewInt(1_000_000_000_000)) {
		return false
	}
	huge := new(big.Int).Lsh(big.NewInt(1), 200)
	return v.sendTx(t, v.whale, v.pool, &venuesABI, "depositProRata", a, v.whale, []*big.Int{huge},
		[]*big.Int{big.NewInt(0)}, big.NewInt(0), big.NewInt(1<<40))
}

// venuesSeq is scenarioSeq with more large fills and forced edge amounts (the two-venue cascades need size), then a
// refresh equal to the state.
func (e *forkEnv) venuesSeq(t *testing.T, rep *forkReport, area string, sim *PoolSimulator, steps int, seed int64) {
	t.Helper()
	s := sim.state
	feedDeadline := min(s.Feed.Asset.Round.UpdatedAt.Uint64()+s.Feed.Asset.Heartbeat.Uint64(),
		s.Feed.Loans[0].Round.UpdatedAt.Uint64()+s.Feed.Loans[0].Heartbeat.Uint64())
	rng := rand.New(rand.NewSource(seed))
	ts := s.Timestamp
	adopted, refused := 0, 0
	for i := 0; i < steps; i++ {
		venue := rng.Intn(3) - 1
		sell := rng.Intn(2) == 0
		gap := uint64(2 + rng.Intn(20))
		if ts+gap+uint64(steps-i)*3 >= feedDeadline {
			gap = 2
		}
		ts += gap
		sim.nowFn = func() uint64 { return ts }
		v := max(venue, 0)
		lo, hi := 100.0, 3_000_000.0
		if !sell {
			lo, hi = 50_000.0, 3_000_000_000.0
		}
		a := uint64(lo * math.Pow(hi/lo, rng.Float64()))
		if rng.Intn(3) == 0 {
			amts := edgeAmounts(sim, v, sell, 1)
			a = amts[rng.Intn(len(amts))]
		}
		params := amountIn(sim, sell, a)
		res, err := sim.calcAmountOut(params, venue)
		where := fmt.Sprintf("%s step %d venue=%d sell=%v amountIn=%d ts=%d", area, i, venue, sell, a, ts)
		if err != nil && errors.Is(err, ErrSpreadNotLive) {
			if chainSpreadLive(sim, ts) {
				rep.mismatch(area+"/spread", where, describeQuote(res, err), "the pool's spread is live")
			}
			ts -= gap
			continue
		}
		fv := uint8(v)
		if err == nil {
			fv = res.SwapInfo.(SwapInfo).Venue
		}
		in, out := e.tokens(sell)
		fill := e.mineFill(t, fv, in, out, params.TokenAmountIn.Amount, ts)
		switch {
		case err == nil && fill.reverted, err != nil && !fill.reverted:
			rep.mismatch(area+"/sequence", where, describeQuote(res, err), describeFill(fill))
			return
		case err == nil:
			if res.TokenAmountOut.Amount.Cmp(fill.out) != 0 || res.RemainingTokenAmountIn.Amount.Cmp(fill.unused) != 0 {
				rep.mismatch(area+"/sequence", where, describeQuote(res, err), describeFill(fill))
				return
			}
			if fill.gas > uint64(res.Gas) {
				rep.mismatch(area+"/gas", where, fmt.Sprintf("estimate %d", res.Gas), fmt.Sprintf("receipt gas %d", fill.gas))
			}
			c := sim.CloneState().(*PoolSimulator)
			c.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
			sim = c
			adopted++
			rep.count(area + "/sequence-identical")
		default:
			if want := forkRevertError(fill.revert); want == nil || !errors.Is(err, want) {
				rep.mismatch(area+"/sequence-refusal", where, describeQuote(res, err), describeFill(fill))
			}
			refused++
		}
		if i%9 == 8 {
			decoded := msgpackHop(t, sim)
			sim = decoded
		}
	}
	fresh, _, err := e.tryTrack(t, sim.Policy)
	if err != nil {
		rep.mismatch(area+"/post-refresh", "after the sequence", "simulator state", err.Error())
		return
	}
	got := sim.state.clone()
	got.Block, got.Timestamp = fresh.state.Block, fresh.state.Timestamp
	if d := e2eDiff(got, fresh.state); len(d) != 0 {
		rep.mismatch(area+"/post-state", "after the sequence", "simulator state", strings.Join(d, "; "))
	}
	t.Logf("%s: sequence adopted %d, refusals mined %d", area, adopted, refused)
}

// ---------------------------------------------------------------- several fills in one block

type plannedFill struct {
	venue  int
	sell   bool
	amount *big.Int
	res    *pool.CalcAmountOutResult
	err    error
	sent   uint8
	hash   common.Hash
	adapt  common.Address
	recip  common.Address
}

func TestForkSameBlock(t *testing.T) {
	v := newVenuesEnv(t)
	e := v.forkEnv
	rep := newForkReport(t)
	e.armLeverage(t, 13_000, 0)
	rng := rand.New(rand.NewSource(0x5AB1))
	const rounds = 10
	nextAddr := uint64(0x1000)
	for round := 0; round < rounds; round++ {
		sim, _, err := e.tryTrack(t, Policy{LeverRouting: true})
		require.NoError(t, err)
		ts := e.f.head().Time + 3
		sim.nowFn = func() uint64 { return ts }
		n := 3 + rng.Intn(10)
		plan := make([]*plannedFill, n)
		cur := sim
		for i := 0; i < n; i++ {
			p := &plannedFill{venue: rng.Intn(3) - 1, sell: rng.Intn(2) == 0}
			lo, hi := 500.0, 400_000.0
			if !p.sell {
				lo, hi = 300_000.0, 400_000_000.0
			}
			a := uint64(lo * math.Pow(hi/lo, rng.Float64()))
			if rng.Intn(4) == 0 {
				amts := edgeAmounts(cur, max(p.venue, 0), p.sell, 1)
				a = amts[rng.Intn(len(amts))]
			}
			p.amount = new(big.Int).SetUint64(a)
			params := amountIn(cur, p.sell, a)
			p.res, p.err = cur.calcAmountOut(params, p.venue)
			p.sent = uint8(max(p.venue, 0))
			if p.err == nil {
				p.sent = p.res.SwapInfo.(SwapInfo).Venue
				c := cur.CloneState().(*PoolSimulator)
				c.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: p.res.SwapInfo})
				if i%4 == 3 {
					c = msgpackHop(t, c)
				}
				c.nowFn = func() uint64 { return ts }
				cur = c
			}
			nextAddr++
			p.adapt = common.BigToAddress(new(big.Int).SetUint64(0xada9700000 + nextAddr))
			p.recip = common.BigToAddress(new(big.Int).SetUint64(0xa1a1a100000 + nextAddr))
			plan[i] = p
		}
		// place, fund and send every fill with mining off, then mine them in one block at ts
		f := e.f
		f.rpc(nil, "evm_setAutomine", false)
		for _, p := range plan {
			f.rpc(nil, "anvil_setCode", p.adapt, hexutil.Encode(e.code))
			in, _ := e.tokens(p.sell)
			f.rpc(nil, "anvil_setStorageAt", in, balanceSlot(p.adapt), common.BigToHash(p.amount))
		}
		for _, p := range plan {
			in, out := e.tokens(p.sell)
			calldata, err := forkABI.Pack("executeEverlongFlamm", adapterData(e.pool, p.sent), p.amount, in, out, p.recip)
			require.NoError(t, err)
			f.rpc(&p.hash, "eth_sendTransaction", map[string]any{"from": forkDeployer, "to": p.adapt,
				"data": hexutil.Encode(calldata), "gas": hexutil.Uint64(8_000_000)})
		}
		f.rpc(nil, "evm_setNextBlockTimestamp", hexutil.Uint64(ts))
		f.rpc(nil, "evm_mine")
		f.rpc(nil, "evm_setAutomine", true)
		var blockHash common.Hash
		identical, refusedAlike := 0, 0
		diverged := false
		for i, p := range plan {
			var rcpt *types.Receipt
			for k := 0; k < 400; k++ {
				if rcpt, err = f.geth.TransactionReceipt(context.Background(), p.hash); err == nil {
					break
				}
				time.Sleep(25 * time.Millisecond)
			}
			require.NotNil(t, rcpt, "receipt %d", i)
			if i == 0 {
				blockHash = rcpt.BlockHash
				h, err := f.geth.HeaderByHash(context.Background(), blockHash)
				require.NoError(t, err)
				require.Equal(t, ts, h.Time)
			}
			require.Equal(t, blockHash, rcpt.BlockHash, "all fills in one block")
			in, out := e.tokens(p.sell)
			where := fmt.Sprintf("round %d fill %d/%d venue=%d sent=%d sell=%v amountIn=%s ts=%d", round, i, n, p.venue,
				p.sent, p.sell, p.amount, ts)
			chainOk := rcpt.Status == types.ReceiptStatusSuccessful
			switch {
			case p.err == nil && !chainOk:
				rep.mismatch("same-block/fill", where, describeQuote(p.res, p.err), "reverted in block")
				diverged = true
			case p.err != nil && chainOk:
				gotOut := f.balance(out, p.recip)
				rep.mismatch("same-block/fill", where, describeQuote(p.res, p.err), fmt.Sprintf("filled out %s", gotOut))
				diverged = true
			case p.err != nil:
				refusedAlike++
			default:
				gotOut, unused := f.balance(out, p.recip), f.balance(in, p.adapt)
				if gotOut.Cmp(p.res.TokenAmountOut.Amount) != 0 || unused.Cmp(p.res.RemainingTokenAmountIn.Amount) != 0 {
					rep.mismatch("same-block/fill", where, describeQuote(p.res, p.err),
						fmt.Sprintf("out %s unused %s", gotOut, unused))
					diverged = true
				} else {
					identical++
					rep.count("same-block/identical")
				}
				if rcpt.GasUsed > uint64(p.res.Gas) {
					rep.mismatch("same-block/gas", where, fmt.Sprintf("estimate %d", p.res.Gas),
						fmt.Sprintf("receipt %d", rcpt.GasUsed))
				}
			}
		}
		t.Logf("round %d: %d fills in one block at %d: %d identical, %d refused alike", round, n, ts, identical,
			refusedAlike)
		if diverged {
			break
		}
		fresh, _, err := e.tryTrack(t, Policy{LeverRouting: true})
		require.NoError(t, err)
		got := cur.state.clone()
		got.Block, got.Timestamp = fresh.state.Block, fresh.state.Timestamp
		if d := e2eDiff(got, fresh.state); len(d) != 0 {
			rep.mismatch("same-block/post-state", fmt.Sprintf("round %d", round), "simulator state",
				strings.Join(d, "; "))
			break
		}
		// sweep the executors' leftovers so balances do not leak into the next round
		for _, p := range plan {
			in, _ := e.tokens(p.sell)
			f.rpc(nil, "anvil_setStorageAt", in, balanceSlot(p.adapt), common.Hash{})
		}
	}
	require.Empty(t, rep.list)
}
