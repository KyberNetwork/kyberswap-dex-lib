package everlongflamm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// c104Deployment is the live Base deployment the recorded fixtures were read from, spelled out from its deployment
// record (c104-deploy @ 80abd43, script/flamm/c104/deployments/c104.8453.json) independently of the registries, so
// the registry tests compare the two.
type c104Deployment struct {
	Pool, Factory, Implementation, Hook, LeverageHook, SpreadHook, Router, PriceFeed, Account, Morpho, PoolAsset,
	LoanAsset common.Address
	PoolCodeHash, FactoryCodeHash, ImplementationCodeHash, HookCodeHash, LeverageHookCodeHash, SpreadHookCodeHash,
	RouterCodeHash, PriceFeedCodeHash, AccountCodeHash common.Hash
	HookSetHash, GenesisStrategyHash common.Hash
}

var c104 = c104Deployment{
	Pool:           common.HexToAddress("0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572"),
	Factory:        common.HexToAddress("0x1BfcE014774D0DD7e04bC595D46Fa09F7dCCF45f"),
	Implementation: common.HexToAddress("0xaAD580BeAa2cbd8Ab5F3956a5c56EDa1D5ee7184"),
	Hook:           common.HexToAddress("0x65CBD227cBC61248ae77a5fC813A29C54C092134"),
	LeverageHook:   common.HexToAddress("0xE0A98d8e60035832B8BaD7f7af7B9B0b3A7308F3"),
	SpreadHook:     common.HexToAddress("0x04988aF54ec88D2de77b191025EAef2fe488f93b"),
	Router:         common.HexToAddress("0x19A9b39E6710AAD109C829294b0841F0851c6bB4"),
	PriceFeed:      common.HexToAddress("0xbED275459578C87a63F2f50A0b077C720e838816"),
	Account:        common.HexToAddress("0x6760E3b032eE2d670Cb684d9076b8f48cb066c48"),
	Morpho:         common.HexToAddress("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb"),
	PoolAsset:      common.HexToAddress("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"),
	LoanAsset:      common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),

	PoolCodeHash:           common.HexToHash("0x5045564af89c87eed146243967ad4d743f37677a9b7c10f91c6e51ba5fadbb63"),
	FactoryCodeHash:        common.HexToHash("0x96428fbb30309ec3afad127b436321031c5b8b13b428f4b4c56fc093d343d4c1"),
	ImplementationCodeHash: common.HexToHash("0x2eb0fb32b59b1cca33e7bd6cad0a214bc6a293dc219d6ff9a5db1c58d0c10cb7"),
	HookCodeHash:           common.HexToHash("0x63ca81587b713df89dc9a657dae2cb5a70910cbafbe23b9ebbbdedc1bfdc3e1d"),
	LeverageHookCodeHash:   common.HexToHash("0xc6b46f4287cafa36c360234956b22dc635eb725dd17134a668939e731b819756"),
	SpreadHookCodeHash:     common.HexToHash("0x79d826c02ae2d2de5f8fdebfc1207731a04a96c0733e7af7c1f030b69d65a425"),
	RouterCodeHash:         common.HexToHash("0x6ca3c38096320c757612b113e543c37862f48bce5a2f19de2375357bc31ddcc8"),
	PriceFeedCodeHash:      common.HexToHash("0xe60545060691e05c25a9b76a2ebb012fbeb8f147e77544cc02390ea3880d547f"),
	AccountCodeHash:        common.HexToHash("0x7d5828b262882bb77ce568da078e381979429cc8a40d624e90e15a1895eee847"),

	HookSetHash:         common.HexToHash("0x6b4146f7cd2ae3767a00555254c562476c851b90babbc132dde2fdf7e0e80e0b"),
	GenesisStrategyHash: common.HexToHash("0x533d23efc2573bf73577c58cdb4f547eea911233a5c0dc1433baa73282208fd0"),
}

// hookSet is the pool's hooks() in field order.
func (d *c104Deployment) hookSet() [7]common.Address {
	return [7]common.Address{d.Hook, d.Hook, d.Hook, d.Hook, d.LeverageHook, d.SpreadHook, {}}
}

// hookSetHash is keccak256(abi.encode(hooks)): seven static address words, the deployment record's hookSetHash.
func hookSetHash(h [7]common.Address) common.Hash {
	var buf [7 * 32]byte
	for i := range h {
		copy(buf[i*32+12:(i+1)*32], h[i][:])
	}
	return crypto.Keccak256Hash(buf[:])
}

// staticExtra is the c104 listing's pinned wiring (the feed configuration left out).
func (d *c104Deployment) staticExtra() StaticExtra {
	return StaticExtra{ProfileVersion: profileVersion, ChainID: valueobject.ChainIDBase, Factory: d.Factory,
		Implementation: d.Implementation, Router: d.Router, PriceFeed: d.PriceFeed, Morpho: d.Morpho,
		PoolAsset: d.PoolAsset, LoanAsset: d.LoanAsset, Hooks: d.hookSet()}
}

func (d *c104Deployment) tokens() []string {
	return []string{lowerHex(d.PoolAsset), lowerHex(d.LoanAsset)}
}

// wiring is the c104 listing resolved against the registries.
func (d *c104Deployment) wiring(t testing.TB) *poolWiring {
	t.Helper()
	se := d.staticExtra()
	w, err := validStatic(&se, lowerHex(d.Pool), DexType, d.tokens())
	require.NoError(t, err)
	return w
}

func (d *c104Deployment) venueScope() venueScope {
	se := d.staticExtra()
	return se.venueScope(d.Pool)
}

// baseState builds reads' state for the hook set the reads name, resolved against the Base registry on the c104
// pool, which every recorded fixture was read from.
func (r *flammReads) baseState() (*flammState, error) {
	hooks, err := resolveHooks(valueobject.ChainIDBase, c104.Factory, c104.Pool, r.Pool.Hooks)
	if err != nil {
		return nil, err
	}
	return r.state(&hooks)
}

// TestHookRegistry: the registries hold exactly the c104 deployment record's hooks, shared contracts and financing
// account, and a lookup is by chain and address.
func TestHookRegistry(t *testing.T) {
	t.Parallel()
	require.Equal(t, c104.HookSetHash, hookSetHash(c104.hookSet()), "the record's hookSetHash is the hook set")
	for _, c := range []struct {
		addr      common.Address
		kind      hookKind
		code      common.Hash
		role      hookRole
		scale     uint64
		dependent bool
	}{
		{c104.Hook, hookKindEverlongSwapV1, c104.HookCodeHash, roleSwap, 1_000_000_000_000, true},
		{c104.LeverageHook, hookKindEverlongLeverageV1, c104.LeverageHookCodeHash, roleLeverage, 1_000_000_000_000, false},
		{c104.SpreadHook, hookKindEverlongSpreadV1, c104.SpreadHookCodeHash, roleSpread, 0, true},
	} {
		e := hookIn(hookRegistry, valueobject.ChainIDBase, c.addr)
		require.NotNil(t, e, c.addr.Hex())
		require.Equal(t, c.kind, e.Kind, c.addr.Hex())
		require.Equal(t, c.code, e.CodeHash, c.addr.Hex())
		require.Equal(t, c104.Pool, e.Pool, c.addr.Hex())
		require.Equal(t, c.scale, e.LoanScale, c.addr.Hex())
		spec := hookKindSpecOf(e.Kind)
		require.NotNil(t, spec, c.addr.Hex())
		require.Equal(t, c.role, spec.role(), c.addr.Hex())
		require.Equal(t, c.dependent, spec.dependency(), c.addr.Hex())
		require.Nil(t, hookIn(hookRegistry, valueobject.ChainIDEthereum, c.addr), "a lookup is per chain")
	}
	swap := hookIn(hookRegistry, valueobject.ChainIDBase, c104.Hook)
	require.Equal(t, [2]common.Address{c104.PoolAsset, c104.LoanAsset}, [2]common.Address{swap.PoolAsset, swap.LoanAsset})
	require.Equal(t, c104.GenesisStrategyHash, swap.GenesisStrategyHash)
	require.Equal(t, c104.Factory, swap.Factory, "the swap hook's entry names the factory that created the pool")
	require.Nil(t, hookIn(hookRegistry, valueobject.ChainIDBase, c104.Router), "not a hook")
	require.Nil(t, hookIn(hookRegistry, valueobject.ChainIDBase, common.Address{}))
	require.Equal(t, "everlong-swap-v1", hookKindEverlongSwapV1.String())
	require.Equal(t, "everlong-leverage-v1", hookKindEverlongLeverageV1.String())
	require.Equal(t, "everlong-spread-v1", hookKindEverlongSpreadV1.String())
	require.Equal(t, "hookKind(9)", hookKind(9).String())
	require.Nil(t, hookKindSpecOf(hookKindNone))
	require.Nil(t, hookKindSpecOf(hookKind(9)))

	d := deploymentFor(valueobject.ChainIDBase, c104.Factory)
	require.NotNil(t, d)
	require.Nil(t, deploymentFor(valueobject.ChainIDEthereum, c104.Factory))
	require.Nil(t, deploymentFor(valueobject.ChainIDBase, c104.Router))
	require.Equal(t, sharedContract{c104.Factory, c104.FactoryCodeHash}, d.Factory)
	require.Equal(t, sharedContract{c104.Router, c104.RouterCodeHash}, d.Router)
	require.Equal(t, []sharedContract{{c104.Implementation, c104.ImplementationCodeHash}}, d.Implementations)
	require.Equal(t, []sharedContract{{c104.PriceFeed, c104.PriceFeedCodeHash}}, d.PriceFeeds)
	require.Equal(t, c104.PoolCodeHash, d.PoolCodeHash)
	require.Equal(t, c104.Morpho, d.Morpho)
	a := financingAccount(valueobject.ChainIDBase, c104.Account)
	require.NotNil(t, a)
	require.Equal(t, boundContract{valueobject.ChainIDBase, c104.Account, c104.AccountCodeHash, c104.Pool}, *a)
	require.Nil(t, financingAccount(valueobject.ChainIDEthereum, c104.Account))
	require.Nil(t, financingAccount(valueobject.ChainIDBase, c104.Hook))
}

// TestRegistryWellFormed walks the three tables as authored. A lookup returns the first entry at an address
// (hookIn, financingAccount, deploymentFor), so a duplicate would resolve first-wins and a misauthored entry would
// unlist its pool on chain without failing anything here: every (chain, address) is registered once across the
// tables, every kind has a spec and a name, every codehash and every immutable of an entry's kind is set and the
// swap-only fields are set on swap entries alone, every pool is one a registered swap hook of the chain is bound
// to, and every swap entry names a registered deployment of its chain.
func TestRegistryWellFormed(t *testing.T) {
	t.Parallel()
	var zeroA common.Address
	var zeroH common.Hash
	type key struct {
		chain valueobject.ChainID
		addr  common.Address
	}
	seen := map[key]string{}
	once := func(chain valueobject.ChainID, a common.Address, what string) {
		require.NotEqual(t, zeroA, a, what)
		k := key{chain, a}
		prior, dup := seen[k]
		require.False(t, dup, "%s: %s is already registered as %s", what, a.Hex(), prior)
		seen[k] = what
	}
	for i := range flammDeployments {
		d := &flammDeployments[i]
		where := fmt.Sprintf("deployment %d (chain %d)", i, d.ChainID)
		once(d.ChainID, d.Factory.Address, where+" factory")
		once(d.ChainID, d.Router.Address, where+" router")
		require.NotEqual(t, zeroH, d.Factory.CodeHash, where)
		require.NotEqual(t, zeroH, d.Router.CodeHash, where)
		require.NotEqual(t, zeroH, d.PoolCodeHash, where)
		require.NotEqual(t, zeroA, d.Morpho, where)
		require.NotEmpty(t, d.Implementations, where)
		require.NotEmpty(t, d.PriceFeeds, where)
		for _, c := range d.Implementations {
			once(d.ChainID, c.Address, where+" implementation")
			require.NotEqual(t, zeroH, c.CodeHash, where)
		}
		for _, c := range d.PriceFeeds {
			once(d.ChainID, c.Address, where+" price feed")
			require.NotEqual(t, zeroH, c.CodeHash, where)
		}
		require.Same(t, d, deploymentFor(d.ChainID, d.Factory.Address), where)
	}
	for k := hookKind(1); k < hookKindCount; k++ {
		spec := hookKindSpecOf(k)
		require.NotNil(t, spec, k.String())
		require.NotEmpty(t, hookKindNames[k], "kind %d has no name", k)
		require.NotContains(t, k.String(), "hookKind(", k.String())
		if spec.role() == roleLeverage {
			// resolveHooksIn refuses a leverage kind that names no swap kind it can read (quotesOnSwapKind), so a
			// kind added without one would stop listing its own pools rather than quote on any swap kind.
			require.Implements(t, (*swapBoundKind)(nil), spec, "%s: a leverage kind is a swapBoundKind", k)
		}
	}
	pools := map[key]bool{} // (chain, pool) a registered swap hook is bound to
	for i := range hookRegistry {
		e := &hookRegistry[i]
		where := fmt.Sprintf("hook %s (chain %d)", e.Address.Hex(), e.ChainID)
		once(e.ChainID, e.Address, where)
		spec := hookKindSpecOf(e.Kind)
		require.NotNil(t, spec, "%s: kind %s has no spec", where, e.Kind)
		require.NotEqual(t, zeroH, e.CodeHash, where)
		require.NotEqual(t, zeroA, e.Pool, where)
		require.Same(t, e, hookIn(hookRegistry, e.ChainID, e.Address), where)
		switch spec.role() {
		case roleSwap:
			require.NotNil(t, deploymentFor(e.ChainID, e.Factory), "%s: factory %s is not a registered deployment",
				where, e.Factory.Hex())
			require.NotEqual(t, zeroA, e.PoolAsset, where)
			require.NotEqual(t, zeroA, e.LoanAsset, where)
			require.NotEqual(t, e.PoolAsset, e.LoanAsset, where)
			require.NotZero(t, e.LoanScale, where)
			require.NotEqual(t, zeroH, e.GenesisStrategyHash, where)
			require.False(t, pools[key{e.ChainID, e.Pool}], "%s: pool %s has two swap hooks", where, e.Pool.Hex())
			pools[key{e.ChainID, e.Pool}] = true
		case roleLeverage:
			require.NotZero(t, e.LoanScale, where)
			fallthrough
		default:
			require.Equal(t, [3]common.Address{}, [3]common.Address{e.Factory, e.PoolAsset, e.LoanAsset},
				"%s: swap-kind fields on a %s entry", where, e.Kind)
			require.Equal(t, zeroH, e.GenesisStrategyHash, "%s: swap-kind field on a %s entry", where, e.Kind)
			if spec.role() == roleSpread {
				require.Zero(t, e.LoanScale, "%s: LOAN_SCALE on a spread entry", where)
			}
		}
	}
	for i := range hookRegistry {
		e := &hookRegistry[i]
		require.True(t, pools[key{e.ChainID, e.Pool}], "hook %s: pool %s has no registered swap hook on chain %d",
			e.Address.Hex(), e.Pool.Hex(), e.ChainID)
	}
	for i := range financingAccounts {
		a := &financingAccounts[i]
		where := fmt.Sprintf("account %s (chain %d)", a.Address.Hex(), a.ChainID)
		once(a.ChainID, a.Address, where)
		require.NotEqual(t, zeroH, a.CodeHash, where)
		require.True(t, pools[key{a.ChainID, a.Pool}], "%s: pool %s has no registered swap hook", where, a.Pool.Hex())
		require.Same(t, a, financingAccount(a.ChainID, a.Address), where)
	}
	// Every registered deployment lists at least one pool, and the pool the offline tapes answer for is one of
	// them. The count is not pinned: registering a second pool is the table's own growth path, and the tests that
	// replay a recorded tape name the pool they are about instead (tapeConfig; README, "Adding a pool whose hooks
	// are of a registered kind").
	for i := range flammDeployments {
		d := &flammDeployments[i]
		require.NotEmpty(t, listingCandidates(d.ChainID, d, nil),
			"the deployment at %s lists no pool", d.Factory.Address.Hex())
	}
	require.Contains(t, listingCandidates(valueobject.ChainIDBase,
		deploymentFor(valueobject.ChainIDBase, c104.Factory), nil), c104.Pool)
	require.NotEmpty(t, pools)
}

// TestResolveHooks: a hook set is admitted only when every hook it names is registered, of its slot's kind and bound
// to the pool, in the shapes the pool and the ports allow.
func TestResolveHooks(t *testing.T) {
	t.Parallel()
	base := valueobject.ChainIDBase
	live, err := resolveHooks(base, c104.Factory, c104.Pool, c104.hookSet())
	require.NoError(t, err)
	require.Equal(t, c104.hookSet(), live.Addrs)
	require.Equal(t, [hookRoles]hookKind{hookKindEverlongSwapV1, hookKindEverlongLeverageV1, hookKindEverlongSpreadV1},
		[hookRoles]hookKind{live.kind(roleSwap), live.kind(roleLeverage), live.kind(roleSpread)})
	require.Equal(t, c104.Hook, live.Entries[roleSwap].Address)

	// A swap-only pool: no leverage and no spread hook.
	swapOnly := c104.hookSet()
	swapOnly[4], swapOnly[5] = common.Address{}, common.Address{}
	h, err := resolveHooks(base, c104.Factory, c104.Pool, swapOnly)
	require.NoError(t, err, "a swap-only hook set")
	require.Equal(t, hookKindNone, h.kind(roleLeverage))
	require.Equal(t, hookKindNone, h.kind(roleSpread))
	require.True(t, h.bound(roleLeverage, &hookBinding{}, c104.Pool), "an empty role has no binding to disagree")

	unknown := common.HexToAddress("0x0000000000000000000000000000000000000bad")
	other := common.HexToAddress("0x00000000000000000000000000000000000000f1")
	for name, c := range map[string]struct {
		edit func(h *[7]common.Address)
		want string
	}{
		"unknown swap hook": {func(h *[7]common.Address) {
			h[0], h[1], h[2], h[3] = unknown, unknown, unknown, unknown
		}, "not registered"},
		"unknown leverage hook": {func(h *[7]common.Address) { h[4] = unknown }, "not registered"},
		"unknown spread hook":   {func(h *[7]common.Address) { h[5] = unknown }, "not registered"},
		"spread hook as leverage": {func(h *[7]common.Address) {
			h[4] = c104.SpreadHook
		}, "is of kind everlong-spread-v1"},
		"leverage hook as spread": {func(h *[7]common.Address) {
			h[5] = c104.LeverageHook
		}, "is of kind everlong-leverage-v1"},
		"leverage hook as swap": {func(h *[7]common.Address) {
			h[0], h[1], h[2], h[3] = c104.LeverageHook, c104.LeverageHook, c104.LeverageHook, c104.LeverageHook
		}, "is of kind everlong-leverage-v1"},
		"swap hook as spread":  {func(h *[7]common.Address) { h[5] = c104.Hook }, "is of kind everlong-swap-v1"},
		"fee hook differs":     {func(h *[7]common.Address) { h[1] = unknown }, "slot 1"},
		"recenter hook zero":   {func(h *[7]common.Address) { h[2] = common.Address{} }, "slot 2"},
		"controller differs":   {func(h *[7]common.Address) { h[3] = c104.SpreadHook }, "slot 3"},
		"no swap hook":         {func(h *[7]common.Address) { *h = [7]common.Address{} }, "no swap hook"},
		"loan-swap hook":       {func(h *[7]common.Address) { h[6] = c104.Hook }, "loan-swap"},
		"leverage alone":       {func(h *[7]common.Address) { h[5] = common.Address{} }, "pair"},
		"spread alone":         {func(h *[7]common.Address) { h[4] = common.Address{} }, "pair"},
		"unregistered address": {func(h *[7]common.Address) { h[4] = other }, "not registered"},
	} {
		hs := c104.hookSet()
		c.edit(&hs)
		_, err := resolveHooks(base, c104.Factory, c104.Pool, hs)
		require.ErrorIs(t, err, ErrInvalidProfile, name)
		require.Contains(t, err.Error(), c.want, name)
	}
	_, err = resolveHooks(valueobject.ChainIDEthereum, c104.Factory, c104.Pool, c104.hookSet())
	require.ErrorIs(t, err, ErrInvalidProfile, "the Base hooks are not registered on another chain")

	// A registered hook bound to a different pool is refused for this one, and admitted for its own.
	registry := append([]hookEntry(nil), hookRegistry...)
	for i := range registry {
		if registry[i].Address == c104.SpreadHook {
			registry[i].Pool = other
		}
	}
	_, err = resolveHooksIn(registry, base, c104.Factory, c104.Pool, c104.hookSet())
	require.ErrorIs(t, err, ErrInvalidProfile)
	require.Contains(t, err.Error(), "is bound to "+other.Hex())
	_, err = resolveHooks(base, c104.Factory, other, c104.hookSet())
	require.ErrorIs(t, err, ErrInvalidProfile, "the c104 hooks on another pool")
	require.Contains(t, err.Error(), "is bound to "+c104.Pool.Hex())

	// The swap hook's entry names the factory that created the pool: the c104 hook set is refused for a pool of any
	// other deployment, and admitted for its own once its entry says so.
	_, err = resolveHooks(base, other, c104.Pool, c104.hookSet())
	require.ErrorIs(t, err, ErrInvalidProfile, "the c104 hooks under another factory")
	require.Contains(t, err.Error(), "is of factory "+c104.Factory.Hex())
	registry = append([]hookEntry(nil), hookRegistry...)
	for i := range registry {
		if registry[i].Address == c104.Hook {
			registry[i].Factory = other
		}
	}
	_, err = resolveHooksIn(registry, base, c104.Factory, c104.Pool, c104.hookSet())
	require.ErrorIs(t, err, ErrInvalidProfile, "an entry naming another factory")
	require.Contains(t, err.Error(), "is of factory "+other.Hex())
	_, err = resolveHooksIn(registry, base, other, c104.Pool, c104.hookSet())
	require.NoError(t, err, "admitted under the factory the entry names")

	// A registered hook of a kind with no port is refused in its slot; the leverage kind quotes only on the swap
	// kind whose book it reads.
	registry = append([]hookEntry(nil), hookRegistry...)
	registry = append(registry, hookEntry{ChainID: base, Address: unknown, Kind: hookKindEverlongSwapV1 + 100,
		Pool: c104.Pool})
	hs := c104.hookSet()
	hs[0], hs[1], hs[2], hs[3] = unknown, unknown, unknown, unknown
	_, err = resolveHooksIn(registry, base, c104.Factory, c104.Pool, hs)
	require.ErrorIs(t, err, ErrInvalidProfile, "an unported kind")
	require.True(t, strings.Contains(err.Error(), "hookKind("), err.Error())
	require.False(t, everlongLeverageV1{}.quotesOn(hookKindEverlongSpreadV1))
	require.True(t, everlongLeverageV1{}.quotesOn(hookKindEverlongSwapV1))

	// The rule resolveHooksIn ends with, at the only place a spec that is no swapBoundKind can be handed to it:
	// a leverage kind that names no swap kind quotes on none, so the hook set is refused rather than admitted on
	// whatever swap kind the pool happens to list. Every shipped leverage spec implements it
	// (TestRegistryWellFormed), so the branch is reachable only from a kind added later.
	require.True(t, quotesOnSwapKind(everlongLeverageV1{}, hookKindEverlongSwapV1))
	require.False(t, quotesOnSwapKind(everlongLeverageV1{}, hookKindEverlongSpreadV1))
	unbound := hookKindSpec(unboundLeverageKind{})
	require.Equal(t, roleLeverage, unbound.role())
	require.NotImplements(t, (*swapBoundKind)(nil), unbound)
	require.False(t, quotesOnSwapKind(unbound, hookKindEverlongSwapV1), "a spec with no quotesOn")
	require.False(t, quotesOnSwapKind(nil, hookKindEverlongSwapV1), "an unported kind")
}

// unboundLeverageKind is a leverage-role spec that is no swapBoundKind: what a kind added without step 5 of the
// README's "Adding a hook kind" looks like to resolveHooksIn. It borrows the spread kind's spec methods, which are
// the ones it does not exercise.
type unboundLeverageKind struct{ everlongSpreadV1 }

func (unboundLeverageKind) role() hookRole { return roleLeverage }

// TestValidStaticRegistry: a listing is admitted only on its own pool, with the pair its swap hook is deployed for,
// and every venue on a registered financing account of that pool.
func TestValidStaticRegistry(t *testing.T) {
	t.Parallel()
	se := c104.staticExtra()
	w, err := validStatic(&se, lowerHex(c104.Pool), DexType, c104.tokens())
	require.NoError(t, err)
	require.Equal(t, c104.Pool, w.pool)
	require.Equal(t, valueobject.ChainIDBase, w.chainID)

	other := common.HexToAddress("0x00000000000000000000000000000000000000f1")
	_, err = validStatic(&se, lowerHex(other), DexType, c104.tokens())
	require.ErrorIs(t, err, ErrInvalidProfile, "the c104 wiring on another pool: its hooks are bound elsewhere")

	// The pair is the swap hook's, even in token order the pair of the listing agrees with.
	swapped := se
	swapped.PoolAsset, swapped.LoanAsset = se.LoanAsset, se.PoolAsset
	_, err = validStatic(&swapped, lowerHex(c104.Pool), DexType, []string{lowerHex(se.LoanAsset), lowerHex(se.PoolAsset)})
	require.ErrorIs(t, err, ErrInvalidProfile)

	venue := StaticVenue{Account: c104.Account, Oracle: other, Irm: adaptiveCurveIrm}
	require.NoError(t, validVenues(w, []StaticVenue{venue}))
	elsewhere := *w
	elsewhere.pool = other
	require.ErrorIs(t, validVenues(&elsewhere, []StaticVenue{venue}), ErrInvalidProfile,
		"the c104 account finances no other pool")
	unregistered := venue
	unregistered.Account = other
	require.ErrorIs(t, validVenues(w, []StaticVenue{venue, unregistered}), ErrInvalidProfile)
	onHook := venue
	onHook.Account = c104.Hook
	require.ErrorIs(t, validVenues(w, []StaticVenue{onHook}), ErrInvalidProfile, "a registered address of another kind")
}

// TestValidStaticOtherDeployment: a listing that names another registered deployment of the chain -- that
// deployment's factory and Router, and shared contracts it registers too -- is refused, and the pool is no
// candidate of that deployment's listing: its swap hook's entry names the factory that created it. Not parallel:
// it extends the deployment table, which every parallel test reads.
func TestValidStaticOtherDeployment(t *testing.T) {
	otherFactory := common.HexToAddress("0x00000000000000000000000000000000000000fa")
	otherRouter := common.HexToAddress("0x00000000000000000000000000000000000000fb")
	d := *deploymentFor(valueobject.ChainIDBase, c104.Factory)
	d.Factory = sharedContract{otherFactory, d.Factory.CodeHash}
	d.Router = sharedContract{otherRouter, d.Router.CodeHash}
	// And the same factory address on another chain: a deployment is a (chain, factory) pair, and the hooks that
	// name a factory name it on their own chain alone.
	elsewhereChain := d
	elsewhereChain.ChainID = valueobject.ChainIDEthereum
	elsewhereChain.Factory = sharedContract{c104.Factory, d.Factory.CodeHash}
	saved := flammDeployments
	flammDeployments = append(append([]flammDeployment(nil), saved...), d, elsewhereChain)
	t.Cleanup(func() { flammDeployments = saved })

	se := c104.staticExtra()
	elsewhere := se
	elsewhere.Factory, elsewhere.Router = otherFactory, otherRouter
	_, err := validStatic(&elsewhere, lowerHex(c104.Pool), DexType, c104.tokens())
	require.ErrorIs(t, err, ErrInvalidProfile)
	require.Contains(t, err.Error(), "is of factory "+c104.Factory.Hex())
	require.Empty(t, listingCandidates(valueobject.ChainIDBase, deploymentFor(valueobject.ChainIDBase, otherFactory),
		nil), "the pool is no candidate of the other deployment's listing")
	require.Empty(t, listingCandidates(valueobject.ChainIDEthereum,
		deploymentFor(valueobject.ChainIDEthereum, c104.Factory), nil),
		"the Base pool is no candidate of a deployment at the same factory address on another chain")
	require.Contains(t, listingCandidates(valueobject.ChainIDBase,
		deploymentFor(valueobject.ChainIDBase, c104.Factory), nil), c104.Pool, "and still a candidate of its own")
	onEthereum := se
	onEthereum.ChainID = valueobject.ChainIDEthereum
	_, err = validStatic(&onEthereum, lowerHex(c104.Pool), DexType, c104.tokens())
	require.ErrorIs(t, err, ErrInvalidProfile, "the Base hooks are not registered on another chain")
	_, err = validStatic(&se, lowerHex(c104.Pool), DexType, c104.tokens())
	require.NoError(t, err, "and still admitted under its own")
}

// TestListerRegistry: the lister admits a pool only on the hook set and PriceFeed its first round reads, and refuses
// anything else there, before a single read is addressed to an unregistered contract.
func TestListerRegistry(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	u := NewPoolsListUpdater(tapeConfig(), tp.client())
	control, listedMd, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, control, 1)
	unknown := common.HexToAddress("0x0000000000000000000000000000000000000bad")
	hooksAt := func(slot int, a common.Address) func(r *tapeResult) { return setResultWord(slot, a.Bytes()) }
	for _, c := range []struct {
		name string
		edit []func(r *tapeResult)
		call func(common.Address, []byte) bool
	}{
		{"unregistered leverage hook", []func(*tapeResult){hooksAt(4, unknown)},
			callTo(t, c104.Pool, &flammABI, "hooks")},
		{"unregistered swap hook", []func(*tapeResult){hooksAt(0, unknown), hooksAt(1, unknown), hooksAt(2, unknown),
			hooksAt(3, unknown)}, callTo(t, c104.Pool, &flammABI, "hooks")},
		{"hooks of the wrong kind", []func(*tapeResult){hooksAt(4, c104.SpreadHook), hooksAt(5, c104.LeverageHook)},
			callTo(t, c104.Pool, &flammABI, "hooks")},
		{"swap hook in the spread slot", []func(*tapeResult){hooksAt(5, c104.Hook)},
			callTo(t, c104.Pool, &flammABI, "hooks")},
		{"a loan-swap hook", []func(*tapeResult){hooksAt(6, c104.Hook)}, callTo(t, c104.Pool, &flammABI, "hooks")},
		{"unregistered price feed", []func(*tapeResult){setResultWord(0, unknown.Bytes())},
			callTo(t, c104.Pool, &flammABI, "priceFeed")},
		{"hooks unreadable", []func(*tapeResult){func(r *tapeResult) { r.Success = false }},
			callTo(t, c104.Pool, &flammABI, "hooks")},
	} {
		var asked map[string]int
		edits := c.edit
		tp.mutate = chainMutations(aggregateCallMutation(t, c.call, func(r *tapeResult) {
			for _, e := range edits {
				e(r)
			}
		}), tapeRecorder(&asked))
		pools, _, err := u.GetNewPools(context.Background(), nil)
		require.NoError(t, err, c.name)
		require.Empty(t, pools, c.name)
		require.Equal(t, map[string]int{"eth_call": 1}, asked, "%s: refused on the first round", c.name)
		// A listed pool the registry no longer admits is not forgotten while the factory owns it: the cursor keeps
		// its digest, and a hook set the registry admits later relists it.
		_, md, err := u.GetNewPools(context.Background(), listedMd)
		require.NoError(t, err, c.name)
		require.JSONEq(t, string(listedMd), string(md), c.name)
	}
	tp.mutate = nil

	// No registered pool to walk: a factory the registries do not know, and an allow-list without the pool.
	for name, cfg := range map[string]*Config{
		"unregistered factory": {DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: c104.Router.Hex()},
		"allow-list": {DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: c104.Factory.Hex(),
			Pools: []string{unknown.Hex()}},
	} {
		var asked map[string]int
		tp.mutate = tapeRecorder(&asked)
		pools, md, err := NewPoolsListUpdater(cfg, tp.client()).GetNewPools(context.Background(), nil)
		require.NoError(t, err, name)
		require.Empty(t, pools, name)
		require.Empty(t, asked, "%s: nothing to read", name)
		require.Nil(t, md, name)
		// A run with nothing to look at returns the cursor it was handed: the prune is what a run that read the
		// factory's answers does to a pool the factory no longer owns, and this run read nothing.
		pools, md, err = NewPoolsListUpdater(cfg, tp.client()).GetNewPools(context.Background(), listedMd)
		require.NoError(t, err, name)
		require.Empty(t, pools, name)
		require.Equal(t, listedMd, md, "%s: the cursor is left as it was", name)
	}
	tp.mutate = nil
	require.Contains(t, listingCandidates(valueobject.ChainIDBase,
		deploymentFor(valueobject.ChainIDBase, c104.Factory), nil), c104.Pool)
}

// tapeAnswersAt is every (block, target, calldata) -> answer the tape recorded inside a tryBlockAndAggregate sent at
// a pinned block, at that block's own clock.
func tapeAnswersAt(t *testing.T, tp *rpcTape) map[string]tapeResult {
	t.Helper()
	out := map[string]tapeResult{}
	tp.mu.Lock()
	defer tp.mu.Unlock()
	for key, e := range tp.entries {
		method, params, ok := strings.Cut(key, "|")
		if !ok || method != "eth_call" || e.Result == nil || strings.Contains(params, `"time"`) {
			continue
		}
		var args []json.RawMessage
		var block string
		if json.Unmarshal([]byte(params), &args) != nil || len(args) < 2 || json.Unmarshal(args[1], &block) != nil ||
			!strings.HasPrefix(block, "0x") {
			continue
		}
		calls, results := aggregateOf(t, json.RawMessage(params), e.Result)
		for i := range results {
			out[block+subcallKey(calls[i].Target, calls[i].CallData)] = results[i]
		}
	}
	return out
}

// aggregateAtBlock answers an aggregate the tape has no entry for, sent at a pinned block and at its own clock, with
// the answers the tape recorded for each of its calls at that block; a call with none leaves the tape miss alone.
func aggregateAtBlock(t *testing.T, answers map[string]tapeResult) func(string, json.RawMessage, *tapeEntry) {
	m := multicallABI.Methods["tryBlockAndAggregate"]
	return func(method string, params json.RawMessage, e *tapeEntry) {
		if method != "eth_call" || e.Result != nil {
			return
		}
		calls, _ := aggregateOf(t, params, nil)
		var args []json.RawMessage
		var block string
		if len(calls) == 0 || json.Unmarshal(params, &args) != nil || len(args) < 2 ||
			json.Unmarshal(args[1], &block) != nil || !strings.HasPrefix(block, "0x") ||
			(len(args) > 3 && strings.Contains(string(args[3]), `"time"`)) {
			return
		}
		results := make([]tapeResult, len(calls))
		for i := range calls {
			r, ok := answers[block+subcallKey(calls[i].Target, calls[i].CallData)]
			if !ok {
				return
			}
			results[i] = tapeResult{Success: r.Success, ReturnData: append([]byte(nil), r.ReturnData...)}
		}
		packed, err := m.Outputs.Pack(new(big.Int).SetBytes(common.FromHex(block)), [32]byte{}, results)
		require.NoError(t, err)
		e.Error = nil
		e.Result, _ = json.Marshal(hexutil.Encode(packed))
	}
}

// TestSwapOnlyPool: a pool that lists no leverage and no spread hook is listed, refreshed and quoted on its swap
// venue alone. The recorded pool is replayed with both roles emptied -- in hooks() and in the FLAMMStore words the
// layout check compares -- so every read the swap-only plan leaves out is not sent, and the rounds that change are
// assembled from the answers the tape recorded at the same block. The swap venue quotes exactly what the full pool
// quotes, and the leverage venue is disabled.
func TestSwapOnlyPool(t *testing.T) {
	t.Parallel()
	cfg := parityConfig(Policy{})
	tpFull := openTape(t, trackerTape)
	fullTracked, err := NewPoolTracker(cfg, tpFull.client()).GetNewPoolStateAtBlock(context.Background(),
		replayListing(t, tpFull), big.NewInt(trackerBlock))
	require.NoError(t, err)

	tp := openTape(t, trackerTape)
	answers := tapeAnswersAt(t, tp)
	levSlot := strings.ToLower(flammSlot(slotLevHook).Hex())
	spreadSlot := strings.ToLower(flammSlot(slotSpreadHook).Hex())
	var asked map[string]int
	var sent []string
	tp.mutate = chainMutations(aggregateAtBlock(t, answers),
		aggregateCallMutation(t, callTo(t, c104.Pool, &flammABI, "hooks"), func(r *tapeResult) {
			setResultWord(4, nil)(r)
			setResultWord(5, nil)(r)
		}),
		func(method string, params json.RawMessage, e *tapeEntry) {
			p := strings.ToLower(string(params))
			if method == "eth_getStorageAt" && strings.Contains(p, strings.ToLower(c104.Pool.Hex()[2:])) &&
				(strings.Contains(p, levSlot) || strings.Contains(p, spreadSlot)) {
				e.Result = json.RawMessage(`"0x0000000000000000000000000000000000000000000000000000000000000000"`)
			}
			if method == "eth_call" || method == "eth_getCode" {
				sent = append(sent, string(params))
			}
			require.Nil(t, e.Error, "%s %s", method, params)
		},
		tapeRecorder(&asked))

	pools, _, err := NewPoolsListUpdater(tapeConfig(), tp.client()).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	listed := pools[0]
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(listed.StaticExtra), &se))
	require.Equal(t, [7]common.Address{c104.Hook, c104.Hook, c104.Hook, c104.Hook}, se.Hooks)
	require.Equal(t, map[string]int{"eth_call": 5, "eth_getCode": 7}, asked, "no code read for the empty roles")

	tracked, err := NewPoolTracker(cfg, tp.client()).GetNewPoolStateAtBlock(context.Background(), listed,
		big.NewInt(trackerBlock))
	require.NoError(t, err)
	for _, params := range sent {
		for _, hook := range []common.Address{c104.LeverageHook, c104.SpreadHook} {
			require.NotContains(t, strings.ToLower(params), strings.ToLower(hook.Hex()[2:]),
				"no read is addressed to a hook the pool does not list")
		}
	}
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.True(t, extra.Attested, "attest %q drift %q", extra.AttestFailure, extra.ProfileDrift)
	require.NotNil(t, extra.Reads.Hook)
	require.Nil(t, extra.Reads.Spread, "no spread hook, no spread reads")
	require.NotContains(t, tracked.Extra, `"spread"`)
	var fullExtra Extra
	require.NoError(t, json.Unmarshal([]byte(fullTracked.Extra), &fullExtra))
	require.Less(t, extra.Probes, fullExtra.Probes, "no previewLever probe")
	deps, _, err := NewPoolTracker(cfg, nil).GetDependencies(context.Background(), tracked)
	require.NoError(t, err)
	require.Contains(t, deps, lowerHex(c104.Hook))
	require.Contains(t, deps, lowerHex(c104.Router))
	require.NotContains(t, deps, lowerHex(c104.SpreadHook))
	require.Equal(t, fullTracked.Reserves, tracked.Reserves)

	sim, err := NewPoolSimulator(tracked)
	require.NoError(t, err)
	fullSim, err := NewPoolSimulator(fullTracked)
	require.NoError(t, err)
	require.False(t, sim.state.Hooks.hasLeverage())
	require.False(t, sim.state.Hooks.hasSpread())
	require.Nil(t, sim.state.Hooks.Spread.EverlongSpread)
	ts := sim.state.Timestamp
	for _, s := range []*PoolSimulator{sim, fullSim} {
		s.nowFn = func() uint64 { return ts }
		s.Policy.LeverRouting = true
	}
	quoted := 0
	for _, sell := range []bool{true, false} {
		for _, a := range []uint64{1_000, 15_000, 100_000, 1_000_000, 3_000_000, 20_000_000, 1_000_000_000} {
			want, wantErr := fullSim.calcAmountOut(amountIn(fullSim, sell, a), int(VenueSwap))
			got, gotErr := sim.calcAmountOut(amountIn(sim, sell, a), -1)
			require.Equal(t, wantErr, gotErr, "sell=%v a=%d", sell, a)
			_, levErr := sim.calcAmountOut(amountIn(sim, sell, a), int(VenueLever))
			require.ErrorIs(t, levErr, ErrLeverageDisabled, "sell=%v a=%d", sell, a)
			if wantErr != nil {
				continue
			}
			quoted++
			require.Equal(t, want.TokenAmountOut.Amount.String(), got.TokenAmountOut.Amount.String())
			require.Equal(t, want.RemainingTokenAmountIn.Amount.String(), got.RemainingTokenAmountIn.Amount.String())
			require.Equal(t, want.Fee.Amount.String(), got.Fee.Amount.String())
			require.Equal(t, want.Gas, got.Gas)
			require.Equal(t, VenueSwap, got.SwapInfo.(SwapInfo).Venue)
			// The post-state is the full pool's but for the roles it does not list.
			gotNext, wantNext := got.SwapInfo.(SwapInfo).next.clone(), want.SwapInfo.(SwapInfo).next.clone()
			gotNext.Hooks.Addrs, gotNext.Hooks.Leverage, gotNext.Hooks.Spread = wantNext.Hooks.Addrs,
				wantNext.Hooks.Leverage, wantNext.Hooks.Spread
			require.Empty(t, e2eDiff(gotNext, wantNext))
			c := sim.CloneState().(*PoolSimulator)
			c.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: got.SwapInfo})
			_, err := c.calcAmountOut(amountIn(c, !sell, a), -1)
			require.NotErrorIs(t, err, ErrSwapInfoMismatch)
		}
	}
	require.Positive(t, quoted)
	require.ErrorIs(t, sim.spreadLive(ts, 0), ErrSpreadNotLive)
}

// swapOnly is reads and their entity with both leverage roles emptied.
func swapOnly(t *testing.T, reads *flammReads, policy Policy) entity.Pool {
	t.Helper()
	r := *reads
	r.Pool.Hooks[4], r.Pool.Hooks[5] = common.Address{}, common.Address{}
	r.Spread = nil
	e := simEntity(t, &r, policy)
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(e.StaticExtra), &se))
	se.Hooks = r.Pool.Hooks
	raw, err := json.Marshal(&se)
	require.NoError(t, err)
	e.StaticExtra = string(raw)
	return e
}

// TestSimulatorHookRegistry: the simulator refuses, at construction and on every quote, a listing whose hooks the
// registry does not admit for the pool and a state whose hook kinds are not the listing's.
func TestSimulatorHookRegistry(t *testing.T) {
	t.Parallel()
	reads := gridReads(t, "51313000", "armed")
	unknown := common.HexToAddress("0x0000000000000000000000000000000000000bad")
	e := simEntity(t, reads, Policy{LeverRouting: true})
	for name, c := range map[string]struct {
		mutate func(se *StaticExtra, x *Extra, p *entity.Pool)
		want   error
	}{
		"unregistered hook": {func(se *StaticExtra, x *Extra, _ *entity.Pool) {
			se.Hooks[4] = unknown
			r := *x.Reads
			r.Pool.Hooks = se.Hooks
			x.Reads = &r
		}, ErrInvalidProfile},
		"hooks of the wrong kind": {func(se *StaticExtra, x *Extra, _ *entity.Pool) {
			se.Hooks[4], se.Hooks[5] = se.Hooks[5], se.Hooks[4]
			r := *x.Reads
			r.Pool.Hooks = se.Hooks
			x.Reads = &r
		}, ErrInvalidProfile},
		"another pool's listing": {func(_ *StaticExtra, _ *Extra, p *entity.Pool) {
			p.Address = lowerHex(unknown)
		}, ErrInvalidProfile},
		"reads of another hook set": {func(_ *StaticExtra, x *Extra, _ *entity.Pool) {
			r := *x.Reads
			r.Pool.Hooks[5] = unknown
			x.Reads = &r
		}, errReadsInconsistent},
		"no swap hook reads": {func(_ *StaticExtra, x *Extra, _ *entity.Pool) {
			r := *x.Reads
			r.Hook = nil
			x.Reads = &r
		}, errReadsInconsistent},
		"no spread hook reads": {func(_ *StaticExtra, x *Extra, _ *entity.Pool) {
			r := *x.Reads
			r.Spread = nil
			x.Reads = &r
		}, errReadsInconsistent},
		// The hook's LOAN_SCALE is compiled in at construction (EverlongHook.sol:156, 10 ** (18 - loanDecimals))
		// while the scale the state is built with comes from the decimals the pool reports at refresh time
		// (state_reads.go), and hook_kinds.go build is the one comparison between the two. Decimals moved with the
		// Router's own loan scale in lockstep, so the read side agrees with itself and only the hook disagrees: the
		// state is refused rather than quoted off a ten-times-wrong scale.
		"loan decimals the hook was not compiled for": {func(_ *StaticExtra, x *Extra, _ *entity.Pool) {
			r := *x.Reads
			r.Pool.Loans = append([]loanConfigReads(nil), r.Pool.Loans...)
			r.Pool.Loans[0].Decimals = 7
			r.Router.Loans = append([]routerLoanReads(nil), r.Router.Loans...)
			r.Router.Loans[0].LoanScale.SetUint64(100_000_000_000) // 10 ** (18 - 7)
			x.Reads = &r
		}, errReadsInconsistent},
	} {
		p := e
		p.Tokens = []*entity.PoolToken{{Address: e.Tokens[0].Address, Swappable: true},
			{Address: e.Tokens[1].Address, Swappable: true}}
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(e.StaticExtra), &se))
		var x Extra
		require.NoError(t, json.Unmarshal([]byte(e.Extra), &x))
		c.mutate(&se, &x, &p)
		raw, err := json.Marshal(&se)
		require.NoError(t, err)
		p.StaticExtra = string(raw)
		raw, err = json.Marshal(&x)
		require.NoError(t, err)
		p.Extra = string(raw)
		_, err = NewPoolSimulator(p)
		require.ErrorIs(t, err, c.want, name)
	}

	sim := simFor(t, reads, Policy{LeverRouting: true})
	params := amountIn(sim, true, 5_000)
	for name, mutate := range map[string]func(s *PoolSimulator){
		"listing on another pool": func(s *PoolSimulator) { s.Info.Address = lowerHex(unknown) },
		"unregistered hook":       func(s *PoolSimulator) { s.StaticExtra.Hooks[4] = unknown },
		"state hook set":          func(s *PoolSimulator) { s.state.Hooks.Addrs[5] = unknown },
		"state kind tag":          func(s *PoolSimulator) { s.state.Hooks.Spread.Kind = hookKindEverlongLeverageV1 },
		"state role emptied":      func(s *PoolSimulator) { s.state.Hooks.Leverage = leverageHookSlot{} },
		"state without its kind's state": func(s *PoolSimulator) {
			s.state.Hooks.Swap.EverlongSwap = nil
		},
		"spread state without its kind": func(s *PoolSimulator) {
			s.state.Hooks.Spread.EverlongSpread = nil
		},
		"an unported kind": func(s *PoolSimulator) { s.state.Hooks.Swap.Kind = hookKind(9) },
	} {
		c := sim.CloneState().(*PoolSimulator)
		mutate(c)
		_, err := c.calcAmountOut(params, -1)
		require.ErrorIs(t, err, ErrInvalidProfile, name)
	}
	for _, venue := range []int{-1, int(VenueSwap), int(VenueLever)} {
		_, err := sim.calcAmountOut(params, venue)
		require.NoError(t, err, "venue %d", venue)
	}

	// The ports refuse a slot whose tag names no state.
	var slot swapHookSlot
	_, err := slot.port()
	require.ErrorIs(t, err, ErrInvalidProfile)
	_, err = (&leverageHookSlot{}).port()
	require.ErrorIs(t, err, ErrInvalidProfile)
	ok, ppm, err := (&spreadHookSlot{}).spreadPpm(sim.now())
	require.NoError(t, err, "an empty spread role answers nothing")
	require.False(t, ok)
	require.True(t, ppm.IsZero())
	_, _, err = (&spreadHookSlot{Kind: hookKindEverlongSpreadV1}).spreadPpm(sim.now())
	require.ErrorIs(t, err, ErrInvalidProfile)
	_, err = everlongLeverageV1{}.previewLever(&leverContext{}, nil)
	require.ErrorIs(t, err, ErrInvalidProfile, "a swap hook the leverage kind cannot read")
}

// TestSimulatorSwapOnly: over every recorded scenario, a pool without leverage and spread hooks quotes its swap venue
// exactly as the pool that lists them, refuses the leverage venue with the pool's own LeverageDisabled, and survives
// the msgpack hop with its empty roles.
func TestSimulatorSwapOnly(t *testing.T) {
	t.Parallel()
	for _, tag := range []string{"live", "armed", "capped", "stale_spread"} {
		reads := gridReads(t, "51313000", tag)
		policy := Policy{LeverRouting: true, PriceBandMarginBps: 1, DebtDriftSec: 30}
		full := simFor(t, reads, policy)
		only, err := NewPoolSimulator(swapOnly(t, reads, policy))
		require.NoError(t, err, tag)
		ts := reads.Timestamp
		only.nowFn = func() uint64 { return ts }
		for _, sell := range []bool{true, false} {
			for _, a := range liveGrid(100, 1<<40, 24) {
				want, wantErr := full.calcAmountOut(amountIn(full, sell, a), int(VenueSwap))
				got, gotErr := only.calcAmountOut(amountIn(only, sell, a), -1)
				require.Equal(t, wantErr, gotErr, "%s sell=%v a=%d", tag, sell, a)
				if wantErr == nil {
					require.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
					require.Equal(t, want.RemainingTokenAmountIn.Amount, got.RemainingTokenAmountIn.Amount)
					require.Equal(t, want.Gas, got.Gas)
				}
				_, err := only.calcAmountOut(amountIn(only, sell, a), int(VenueLever))
				require.ErrorIs(t, err, ErrLeverageDisabled)
			}
		}
		raw, err := json.Marshal(only.state)
		require.NoError(t, err)
		require.NotContains(t, string(raw), `"everlongSpread"`)

		// The msgpack hop keeps the empty roles empty and the swap role's state.
		var buf bytes.Buffer
		enc := msgpack.NewEncoder(&buf)
		enc.IncludeUnexported(true)
		enc.SetForceAsArray(true)
		require.NoError(t, enc.Encode(only))
		dec := msgpack.NewDecoder(&buf)
		dec.IncludeUnexported(true)
		dec.SetForceAsArray(true)
		var decoded PoolSimulator
		require.NoError(t, dec.Decode(&decoded))
		decoded.nowFn = only.nowFn
		require.Equal(t, only.state.Hooks, decoded.state.Hooks, tag)
		require.Nil(t, decoded.state.Hooks.Spread.EverlongSpread)
		require.NotSame(t, only.state.Hooks.Swap.EverlongSwap, decoded.state.Hooks.Swap.EverlongSwap)
		for _, sell := range []bool{true, false} {
			want, wantErr := only.calcAmountOut(amountIn(only, sell, 15_000), -1)
			got, gotErr := decoded.calcAmountOut(amountIn(&decoded, sell, 15_000), -1)
			require.Equal(t, wantErr, gotErr, tag)
			if wantErr == nil {
				require.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount, tag)
			}
		}
	}
}
