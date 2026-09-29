package everlongflamm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The code this port is validated against, as hand-maintained registries (c104-deploy @ 80abd43). FLAMM hooks are
// write-once and bound to one pool (EverlongHook.sol:93, EverlongLeverageHook.sol:36, LeverageSpreadHook.sol:29), so
// a hook is registered by its address: the kind of code it runs, which names the port that quotes it
// (hook_kinds.go), that code's runtime codehash, immutables included, and the pool it is bound to. A pool is listed
// and quoted only when every hook its hooks() names is registered, of the kind its slot requires and bound to that
// pool, its swap hook's entry names the factory that created it, and every contract it shares with other pools is
// registered as well: the factory, the factory's Router, the implementation its beacon serves, the PriceFeed, and
// the pool's financing account.
//
// A pool whose hooks run code a kind already ports is added by registering its hooks and its financing account
// here. A hook of new code needs a kind and its port (hook_kinds.go) first.

// profileVersion versions StaticExtra's shape.
const profileVersion = 1

// hookKind names the code a registered hook runs, and with it the port that quotes it.
type hookKind uint8

const (
	hookKindNone hookKind = iota
	// hookKindEverlongSwapV1 is EverlongHook (src/hooks/everlong/EverlongHook.sol) in the invariant, fee, recenter
	// and controller slots, ported by hook.go, fee.go and almcurve.go.
	hookKindEverlongSwapV1
	// hookKindEverlongLeverageV1 is EverlongLeverageHook (src/hooks/everlong/lev/EverlongLeverageHook.sol), ported by
	// levhook.go and levcurve.go.
	hookKindEverlongLeverageV1
	// hookKindEverlongSpreadV1 is LeverageSpreadHook (src/hooks/everlong/lev/LeverageSpreadHook.sol), ported by
	// spreadHookState (state.go).
	hookKindEverlongSpreadV1
	hookKindCount
)

var hookKindNames = [hookKindCount]string{"none", "everlong-swap-v1", "everlong-leverage-v1", "everlong-spread-v1"}

func (k hookKind) String() string {
	if k < hookKindCount {
		return hookKindNames[k]
	}
	return fmt.Sprintf("hookKind(%d)", uint8(k))
}

// hookRole is the part of IFLAMM.HookSet a kind fills. The swap role is the invariant, fee, recenter and controller
// slots, the first two of which the swap path calls (FLAMMSwapLib.sol:84-87, :149, :187, :197); the leverage role is
// the leverageHook slot (FLAMMLeverLib.sol:100, :128, :183) and the spread role the spreadHook slot
// (FLAMMLeverLib.sol:163). No kind fills the loanSwapHook slot.
type hookRole uint8

const (
	roleSwap hookRole = iota
	roleLeverage
	roleSpread
	hookRoles
)

// roleSlot is the hooks() slot each role's hook is read from (IFLAMM.HookSet field order, FLAMM.sol:224), roleName
// how a refusal names it and roleCodeName how the listing's runtime-code check does. hookSlotLoanSwap is the slot no
// kind fills.
var (
	roleSlot     = [hookRoles]int{0, 4, 5}
	roleName     = [hookRoles]string{"swap hook", "leverage hook", "spread hook"}
	roleCodeName = [hookRoles]string{"hook", "leverageHook", "spreadHook"}
)

const hookSlotLoanSwap = 6

// hookEntry is one registered hook: its address, the kind of code it runs, that code's runtime codehash and the
// pool it is bound to (POOL(), an immutable of every kind), plus the immutables of its kind the lister and every
// refresh read back and compare (hook_kinds.go hookKindSpec.bound). The swap kind's entry also names the pool's
// deployment: the factory that created it, which the pool binds as its factory at initialization
// (FLAMMOpsLib.sol:124) and reports as factory() (FLAMM.sol:218-220).
type hookEntry struct {
	ChainID  valueobject.ChainID
	Address  common.Address
	Kind     hookKind
	CodeHash common.Hash
	Pool     common.Address
	// Factory is the registered deployment (flammDeployments) whose factory created Pool: the pool is a candidate of
	// that deployment's listing alone and its listing is refused under any other. Swap kind only.
	Factory common.Address
	// PoolAsset and LoanAsset are the pair a swap hook is deployed for: its anchor price is quoted in it and its
	// LOAN_SCALE is the loan asset's (EverlongHook.sol:151-159). Swap kind only.
	PoolAsset, LoanAsset common.Address
	// LoanScale is LOAN_SCALE (EverlongHook.sol:94, EverlongLeverageHook.sol:38). Swap and leverage kinds.
	LoanScale uint64
	// GenesisStrategyHash is EverlongHook.genesisStrategyHash (EverlongHook.sol:96), fixed at construction while the
	// live strategyHash and paramsHash move with keeper tuning. Swap kind only.
	GenesisStrategyHash common.Hash
}

// loanScale is the entry's LOAN_SCALE as a word.
func (e *hookEntry) loanScale() uint256.Int {
	return *uint256.NewInt(e.LoanScale)
}

// boundContract is a registered pool-bound contract that is not a hook: a financing account, which the Router
// deploys once per pool, loan asset and venue kind and every venue of that kind uses (MMRouter.sol:131-133,
// MMRouterLib.sol:140-157).
type boundContract struct {
	ChainID  valueobject.ChainID
	Address  common.Address
	CodeHash common.Hash
	Pool     common.Address
}

// sharedContract is a registered contract every pool of a deployment may be wired to.
type sharedContract struct {
	Address  common.Address
	CodeHash common.Hash
}

// flammDeployment is what the pools of one FLAMMFactory share, with the runtime code the port mirrors. A contract
// that is one address is registered by it: the factory, its Router (FLAMMFactory.ROUTER, the one Router of every
// pool the factory creates, FLAMMFactory.sol:17-19), each implementation its beacon may serve and each PriceFeed its
// pools may be wired to. The pool proxy is registered by codehash: every pool is its own FLAMMProxy pointed at the
// factory (FLAMMDeployLib.sol:21-27), whose code is the same for every pool of the factory.
type flammDeployment struct {
	ChainID         valueobject.ChainID
	Factory         sharedContract
	Router          sharedContract
	Implementations []sharedContract
	PriceFeeds      []sharedContract
	PoolCodeHash    common.Hash
	Morpho          common.Address
}

var (
	baseCbBTC       = common.HexToAddress("0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf") // 8 decimals
	baseUSDC        = common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913") // 6 decimals
	baseC104Pool    = common.HexToAddress("0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572")
	baseC104Factory = common.HexToAddress("0x1BfcE014774D0DD7e04bC595D46Fa09F7dCCF45f")
)

// flammDeployments: c104 on Base (c104-deploy @ 80abd43, script/flamm/c104/deployments/c104.8453.json, codehashes
// verified on chain).
var flammDeployments = []flammDeployment{{
	ChainID: valueobject.ChainIDBase,
	Factory: sharedContract{baseC104Factory,
		common.HexToHash("0x96428fbb30309ec3afad127b436321031c5b8b13b428f4b4c56fc093d343d4c1")},
	Router: sharedContract{common.HexToAddress("0x19A9b39E6710AAD109C829294b0841F0851c6bB4"),
		common.HexToHash("0x6ca3c38096320c757612b113e543c37862f48bce5a2f19de2375357bc31ddcc8")},
	Implementations: []sharedContract{{common.HexToAddress("0xaAD580BeAa2cbd8Ab5F3956a5c56EDa1D5ee7184"),
		common.HexToHash("0x2eb0fb32b59b1cca33e7bd6cad0a214bc6a293dc219d6ff9a5db1c58d0c10cb7")}},
	PriceFeeds: []sharedContract{{common.HexToAddress("0xbED275459578C87a63F2f50A0b077C720e838816"),
		common.HexToHash("0xe60545060691e05c25a9b76a2ebb012fbeb8f147e77544cc02390ea3880d547f")}},
	PoolCodeHash: common.HexToHash("0x5045564af89c87eed146243967ad4d743f37677a9b7c10f91c6e51ba5fadbb63"),
	Morpho:       common.HexToAddress("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb"),
}}

// hookRegistry: every hook the port quotes, by chain.
var hookRegistry = []hookEntry{
	// c104 on Base: the pool's EverlongHook, EverlongLeverageHook and LeverageSpreadHook (c104.8453.json hook,
	// leverageHook, spreadHook).
	{
		ChainID:             valueobject.ChainIDBase,
		Address:             common.HexToAddress("0x65CBD227cBC61248ae77a5fC813A29C54C092134"),
		Kind:                hookKindEverlongSwapV1,
		CodeHash:            common.HexToHash("0x63ca81587b713df89dc9a657dae2cb5a70910cbafbe23b9ebbbdedc1bfdc3e1d"),
		Pool:                baseC104Pool,
		Factory:             baseC104Factory,
		PoolAsset:           baseCbBTC,
		LoanAsset:           baseUSDC,
		LoanScale:           1_000_000_000_000,
		GenesisStrategyHash: common.HexToHash("0x533d23efc2573bf73577c58cdb4f547eea911233a5c0dc1433baa73282208fd0"),
	},
	{
		ChainID:   valueobject.ChainIDBase,
		Address:   common.HexToAddress("0xE0A98d8e60035832B8BaD7f7af7B9B0b3A7308F3"),
		Kind:      hookKindEverlongLeverageV1,
		CodeHash:  common.HexToHash("0xc6b46f4287cafa36c360234956b22dc635eb725dd17134a668939e731b819756"),
		Pool:      baseC104Pool,
		LoanScale: 1_000_000_000_000,
	},
	{
		ChainID:  valueobject.ChainIDBase,
		Address:  common.HexToAddress("0x04988aF54ec88D2de77b191025EAef2fe488f93b"),
		Kind:     hookKindEverlongSpreadV1,
		CodeHash: common.HexToHash("0x79d826c02ae2d2de5f8fdebfc1207731a04a96c0733e7af7c1f030b69d65a425"),
		Pool:     baseC104Pool,
	},
}

// financingAccounts: every financing account a venue may finance through, by chain.
var financingAccounts = []boundContract{
	// c104 on Base: the pool's MorphoBlueAccount for its loan asset (c104.8453.json account).
	{
		ChainID:  valueobject.ChainIDBase,
		Address:  common.HexToAddress("0x6760E3b032eE2d670Cb684d9076b8f48cb066c48"),
		CodeHash: common.HexToHash("0x7d5828b262882bb77ce568da078e381979429cc8a40d624e90e15a1895eee847"),
		Pool:     baseC104Pool,
	},
}

// deploymentFor is the registered deployment of factory on chainID, or nil.
func deploymentFor(chainID valueobject.ChainID, factory common.Address) *flammDeployment {
	for i := range flammDeployments {
		if d := &flammDeployments[i]; d.ChainID == chainID && d.Factory.Address == factory {
			return d
		}
	}
	return nil
}

// sharedIn is the registered contract at a in list.
func sharedIn(list []sharedContract, a common.Address) (sharedContract, bool) {
	for _, c := range list {
		if c.Address == a {
			return c, true
		}
	}
	return sharedContract{}, false
}

// hookIn is registry's entry for the hook at a on chainID, or nil.
func hookIn(registry []hookEntry, chainID valueobject.ChainID, a common.Address) *hookEntry {
	for i := range registry {
		if e := &registry[i]; e.ChainID == chainID && e.Address == a {
			return e
		}
	}
	return nil
}

// financingAccount is the registered financing account at a on chainID, or nil.
func financingAccount(chainID valueobject.ChainID, a common.Address) *boundContract {
	for i := range financingAccounts {
		if c := &financingAccounts[i]; c.ChainID == chainID && c.Address == a {
			return c
		}
	}
	return nil
}

// poolHookSet is a pool's hooks() resolved against the registry: the listed addresses and the entry behind each
// role's hook, nil for an empty role.
type poolHookSet struct {
	Addrs   [7]common.Address
	Entries [hookRoles]*hookEntry
}

// kind is the kind of role's hook, hookKindNone for an empty role.
func (h *poolHookSet) kind(role hookRole) hookKind {
	if e := h.Entries[role]; e != nil {
		return e.Kind
	}
	return hookKindNone
}

// bound reports whether the bindings read from role's hook are its entry's (an empty role has none to disagree).
func (h *poolHookSet) bound(role hookRole, b *hookBinding, pool common.Address) bool {
	e := h.Entries[role]
	if e == nil {
		return true
	}
	spec := hookKindSpecOf(e.Kind)
	return spec != nil && spec.bound(e, b, pool, h.Addrs[roleSlot[roleSwap]])
}

// resolveHooks resolves hooks, the hook set pool lists, against chainID's registry, for a pool of the deployment
// whose factory is factory, or refuses it.
func resolveHooks(chainID valueobject.ChainID, factory, pool common.Address,
	hooks [7]common.Address) (poolHookSet, error) {
	return resolveHooksIn(hookRegistry, chainID, factory, pool, hooks)
}

// resolveHooksIn is resolveHooks over registry. Every non-zero hook must be registered, of a kind whose role is its
// slot's and bound to pool, and the swap hook's entry must name factory as the pool's. Beyond what the pool itself
// enforces (FLAMMOpsLib.sol:203-225: a swap role, and the leverage and spread hooks set as a pair), the swap role's
// four slots must name one hook, since the swap kinds port the invariant and fee roles as one contract's storage;
// the loanSwapHook slot must be empty, since no kind ports it -- defence in depth over the one-loan envelope, which
// already makes swapLoan unreachable (a pool whose only loan asset is the listed one has no pair to swap:
// FLAMMLoanSwapLib.sol:67, :127) and is re-pinned on every quote (pool_simulator.go envelope); and a leverage kind
// must be able to quote on the swap kind (quotesOnSwapKind).
func resolveHooksIn(registry []hookEntry, chainID valueobject.ChainID, factory, pool common.Address,
	hooks [7]common.Address) (poolHookSet, error) {
	out := poolHookSet{Addrs: hooks}
	var zero common.Address
	for i := roleSlot[roleSwap] + 1; i < roleSlot[roleLeverage]; i++ {
		if hooks[i] != hooks[roleSlot[roleSwap]] {
			return out, fmt.Errorf("%w: hook slot %d is not the invariant hook", ErrInvalidProfile, i)
		}
	}
	switch {
	case hooks[hookSlotLoanSwap] != zero:
		return out, fmt.Errorf("%w: loan-swap hook %s", ErrInvalidProfile, hooks[hookSlotLoanSwap].Hex())
	case (hooks[roleSlot[roleLeverage]] == zero) != (hooks[roleSlot[roleSpread]] == zero):
		return out, fmt.Errorf("%w: leverage and spread hooks are set as a pair", ErrInvalidProfile)
	}
	for role := roleSwap; role < hookRoles; role++ {
		a := hooks[roleSlot[role]]
		if a == zero {
			if role == roleSwap {
				return out, fmt.Errorf("%w: no swap hook", ErrInvalidProfile)
			}
			continue
		}
		e := hookIn(registry, chainID, a)
		switch {
		case e == nil:
			return out, fmt.Errorf("%w: %s %s is not registered", ErrInvalidProfile, roleName[role], a.Hex())
		case hookKindSpecOf(e.Kind) == nil || hookKindSpecOf(e.Kind).role() != role:
			return out, fmt.Errorf("%w: %s %s is of kind %s", ErrInvalidProfile, roleName[role], a.Hex(), e.Kind)
		case e.Pool != pool:
			return out, fmt.Errorf("%w: %s %s is bound to %s", ErrInvalidProfile, roleName[role], a.Hex(), e.Pool.Hex())
		}
		out.Entries[role] = e
	}
	if swap := out.Entries[roleSwap]; swap.Factory != factory {
		return out, fmt.Errorf("%w: swap hook %s is of factory %s", ErrInvalidProfile, swap.Address.Hex(),
			swap.Factory.Hex())
	}
	if lev := out.Entries[roleLeverage]; lev != nil &&
		!quotesOnSwapKind(hookKindSpecOf(lev.Kind), out.kind(roleSwap)) {
		return out, fmt.Errorf("%w: kind %s does not quote on kind %s", ErrInvalidProfile, lev.Kind,
			out.kind(roleSwap))
	}
	return out, nil
}

func lowerHex(a common.Address) string {
	return hexutil.Encode(a[:])
}

// poolWiring is a listing resolved against the registries: the pool, its deployment and its hook set.
type poolWiring struct {
	chainID    valueobject.ChainID
	pool       common.Address
	deployment *flammDeployment
	hooks      poolHookSet
}

// validStatic checks a persisted listing's pinned identity against the registries: the pool, its pair in token
// order (pool asset, loan asset 0) and the pair its swap hook is deployed for, the shared contracts and the hook set,
// whose swap hook must name the listing's deployment as the pool's. The venue set is not pinned here (validVenues).
func validStatic(se *StaticExtra, address, poolType string, tokens []string) (*poolWiring, error) {
	if se == nil || se.ProfileVersion != profileVersion || poolType != DexType || !common.IsHexAddress(address) ||
		address != lowerHex(common.HexToAddress(address)) {
		return nil, ErrInvalidProfile
	}
	d := deploymentFor(se.ChainID, se.Factory)
	if d == nil || se.Router != d.Router.Address || se.Morpho != d.Morpho || len(tokens) != 2 ||
		tokens[0] != lowerHex(se.PoolAsset) || tokens[1] != lowerHex(se.LoanAsset) {
		return nil, ErrInvalidProfile
	}
	if _, ok := sharedIn(d.Implementations, se.Implementation); !ok {
		return nil, ErrInvalidProfile
	}
	if _, ok := sharedIn(d.PriceFeeds, se.PriceFeed); !ok {
		return nil, ErrInvalidProfile
	}
	pool := common.HexToAddress(address)
	hooks, err := resolveHooks(se.ChainID, d.Factory.Address, pool, se.Hooks)
	if err != nil {
		return nil, err
	}
	if swap := hooks.Entries[roleSwap]; swap.PoolAsset != se.PoolAsset || swap.LoanAsset != se.LoanAsset {
		return nil, fmt.Errorf("%w: pair", ErrInvalidProfile)
	}
	return &poolWiring{chainID: se.ChainID, pool: pool, deployment: d, hooks: hooks}, nil
}

// validVenues checks a refreshed venue set against the registry: at least one venue and at most maxVenues, each on a
// registered financing account bound to the pool -- whose runtime code the registry pins and the listing verified --
// lending against AdaptiveCurveIrm (or a market with no IRM) through an oracle that exists. A venue the curator adds
// to a live pool reaches the simulator through Extra, so this runs on every quote, not only at listing.
func validVenues(w *poolWiring, venues []StaticVenue) error {
	if w == nil || len(venues) == 0 || len(venues) > maxVenues {
		return ErrInvalidProfile
	}
	for i := range venues {
		v := &venues[i]
		if a := financingAccount(w.chainID, v.Account); a == nil || a.Pool != w.pool ||
			(v.Irm != adaptiveCurveIrm && v.Irm != (common.Address{})) || v.Oracle == (common.Address{}) {
			return ErrInvalidProfile
		}
	}
	return nil
}

// validEntity is validStatic over an entity.
func validEntity(p *entity.Pool, se *StaticExtra) (*poolWiring, error) {
	tokens := make([]string, len(p.Tokens))
	for i, t := range p.Tokens {
		if t == nil {
			return nil, ErrInvalidProfile
		}
		tokens[i] = t.Address
	}
	return validStatic(se, p.Address, p.Type, tokens)
}
