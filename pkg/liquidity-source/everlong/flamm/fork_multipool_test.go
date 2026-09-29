package everlongflamm

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Several FLAMM pools on one anvil fork of Base (same gating as fork_parity_test.go, plus EVERLONG_FLAMM_C104_OUT, the
// c104-deploy forge output directory the hooks are deployed from; default block 51313000):
//
//   - two more pools are created through the live FLAMMFactory.createPool, each with hooks of its own: fresh
//     EverlongHook / EverlongLeverageHook / LeverageSpreadHook instances built from the c104 artifacts, linked to the
//     deployed libraries and wired as C104Deployer.deployHook / createPool wire them (C104Deployer.sol:179-225),
//     bound to the predicted pool, on a Morpho market of their own (LLTV 0.915 on the c104 oracle and IRM). Pool B
//     runs a different strategy (amplification, span, fee row) and spread; pool C is B's twin with the c104
//     strategy. The creation code and constructor arguments are first shown to be exactly what the c104 deployment
//     sent, and each fresh hook's runtime code to be the live c104 hook's but for its immutables;
//   - the new hooks and accounts are registered in a test-only extension of the registries (mpRegistry), on the
//     fork's chain, and the real lister lists exactly the pools every one of whose hooks is registered, of its slot's
//     kind and bound to it -- a pool whose hooks are not registered, or a registry line of the wrong kind, pool,
//     factory, codehash or genesis strategy, is not listed, and unregistering a listed pool's hooks makes it fail
//     closed;
//   - the tracker attests every listed pool under the parity and the production policies, each with its own hooks
//     and kinds, and every pool's quotes equal its own previewSwap / previewLever and the adapter's fills on both
//     venues, including a chained sequence of mined fills interleaved across the c104 pool and pool B;
//   - per-pool dispatch: B's and C's quotes differ, and each equals the other's once only the hook states are
//     exchanged; a keeper retuning B's fee row moves B and leaves the c104 pool's state untouched;
//   - static data naming another pool's registered hook, and a hook slot rewritten on chain to another pool's hook,
//     are refused.

const multipoolArtifactsEnv = "EVERLONG_FLAMM_C104_OUT"

// mpLinkedLibraries are the libraries the c104 hooks and factory link, at their Base addresses (the deploy
// broadcast's `libraries`; README "linked libraries").
var mpLinkedLibraries = map[string]common.Address{
	"AlmCurve":           common.HexToAddress("0xf82DdF0A8a50bc2C3F163997766bA1839E527A17"),
	"CollRebalancerMath": common.HexToAddress("0xc002d0731e6A2E6e80bE754779BCef6B01aFF0BB"),
	"PoolDeployLib":      common.HexToAddress("0x4F26504C30999CdA1A16d0B09E8c85D4cCf27013"),
	"RouterDeployLib":    common.HexToAddress("0xDe756432Cbe1d813E97e2B30148cba8dfe9Fd816"),
}

// mpC104Txs are the c104 deployment transactions whose inputs the artifacts and parameters here must reproduce
// (broadcast/DeployC104.s.sol/8453/run-latest.json).
var mpC104Txs = struct{ hook, leverageHook, spreadHook, createPool common.Hash }{
	hook:         common.HexToHash("0x535571239108699b51103de915ecb7005dceb8f7923bbe920b3343682298be6d"),
	leverageHook: common.HexToHash("0xeb9575c7bd2bcc7de0bbc11a1bb936bd4dfb04d70e3e69729817cb17eff322a4"),
	spreadHook:   common.HexToHash("0xcb95906da659f310b4d56065c778d962d4857df9f019174aa36cafcc48374785"),
	createPool:   common.HexToHash("0x783b464e93692538bde6dc1b53b959087f1e0ba2af3b6bc55ff13a449101d1e0"),
}

// c104 deployment record values the wiring below reproduces (c104.8453.json).
var (
	mpC104Core      = common.HexToAddress("0xf39b775926b876768F215489DE4f42703F153fD1")
	mpC104Allowlist = common.HexToAddress("0x1da990C8b0bF15D1F441782f01351aCFEE49BBAA")
	mpC104Receiver  = common.HexToAddress("0xeb765F3184f705F5292679f42beEDBbe5D272c49")
	mpC104Anchor    = big.NewInt(768302232848967)
	mpC104RiskHash  = common.HexToHash("0x010982230d4bd1462816dcbc277a406e735fa3240bef632ee2ff156c8c26af0a")
	mpC104Market    = common.HexToHash("0x9103c3b4e834476c9a62ea009ba2c884ee42e94e6e314a26f04d312434191836")
	mpC104Oracle    = common.HexToAddress("0x663BECd10daE6C4A3Dcd89F1d76c1174199639B9")
)

const (
	mpLltv915   = 915_000_000_000_000_000
	mpLltv86    = 860_000_000_000_000_000
	mpFeatures  = 63 // the operating bitmap; genesis withholds SWAP_BUY (DeployC104.s.sol:232-233)
	mpLoanScale = 1_000_000_000_000
)

// ---------------------------------------------------------------- c104 artifacts

type mpCodeRange struct {
	Start  int `json:"start"`
	Length int `json:"length"`
}

type mpBytecode struct {
	Object              string                              `json:"object"`
	LinkReferences      map[string]map[string][]mpCodeRange `json:"linkReferences"`
	ImmutableReferences map[string][]mpCodeRange            `json:"immutableReferences"`
}

// mpArtifact is one forge artifact of the c104 build: its ABI, its creation code with the libraries linked, and
// where its runtime code holds immutables.
type mpArtifact struct {
	ABI        abi.ABI
	Creation   []byte
	Immutables []mpCodeRange
}

func loadMPArtifact(t *testing.T, dir, name string) *mpArtifact {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name+".sol", name+".json"))
	require.NoError(t, err)
	var art struct {
		ABI              json.RawMessage `json:"abi"`
		Bytecode         mpBytecode      `json:"bytecode"`
		DeployedBytecode mpBytecode      `json:"deployedBytecode"`
	}
	require.NoError(t, json.Unmarshal(raw, &art))
	parsed, err := abi.JSON(bytes.NewReader(art.ABI))
	require.NoError(t, err)
	code := []byte(strings.TrimPrefix(art.Bytecode.Object, "0x"))
	for _, libs := range art.Bytecode.LinkReferences {
		for lib, refs := range libs {
			addr, ok := mpLinkedLibraries[lib]
			require.True(t, ok, "%s links %s", name, lib)
			for _, r := range refs {
				require.Equal(t, 20, r.Length)
				copy(code[2*r.Start:2*(r.Start+r.Length)], hex.EncodeToString(addr[:]))
			}
		}
	}
	creation, err := hex.DecodeString(string(code))
	require.NoError(t, err, "%s: every link reference resolved", name)
	var immutables []mpCodeRange
	for _, refs := range art.DeployedBytecode.ImmutableReferences {
		immutables = append(immutables, refs...)
	}
	return &mpArtifact{ABI: parsed, Creation: creation, Immutables: immutables}
}

// deployment is the creation code followed by the abi-encoded constructor arguments.
func (a *mpArtifact) deployment(t *testing.T, args ...any) []byte {
	t.Helper()
	packed, err := a.ABI.Constructor.Inputs.Pack(args...)
	require.NoError(t, err)
	return append(append([]byte(nil), a.Creation...), packed...)
}

// masked is runtime code with every immutable zeroed.
func (a *mpArtifact) masked(code []byte) []byte {
	out := append([]byte(nil), code...)
	for _, r := range a.Immutables {
		if r.Start+r.Length <= len(out) {
			clear(out[r.Start : r.Start+r.Length])
		}
	}
	return out
}

// EverlongHook.Params (EverlongHook.sol:52-83), EverlongStrategy.FeeParams (EverlongStrategy.sol:28-37).
type mpFeeParams struct {
	MidFeeWad, OutFeeWad, GammaWad, SigmaRefWad, VolBetaWad, VolMinWad, VolMaxWad, DirSkewWad uint64
}

type mpTuning struct {
	Fee                                                      mpFeeParams
	InvSkewKappaWad, InvSkewBandWad, EmaHalfLife, RvHalfLife uint64
	StepDivisorWad                                           *big.Int
	InertiaWad, InertiaMaxWad                                uint64
}

type mpBounds struct {
	MinFeeWad, MaxFeeWad, MinEmaHalfLife, MaxEmaHalfLife, MaxRvHalfLife, MaxStepWad uint64
	MinStepDivisorWad                                                               *big.Int
}

type mpHookParams struct {
	AWad, SpanUpWad, SpanDnWad, AnchorPriceWad *big.Int
	LoanDecimals                               uint8
	Tuning                                     mpTuning
	Bounds                                     mpBounds
}

// FLAMMStore.RiskInit (FLAMMStore.sol:221-243) and FLAMMFactory.PoolParams (FLAMMFactory.sol:31-44).
type mpRiskInit struct {
	PhiWad, PhiMinWad, PhiMaxWad, LtvWad, LtvMinWad, LtvMaxWad, LtvMaxStepWad  uint64
	LtvCooldownSec                                                             uint32
	MinStructDistWad, RoomEpsilonWad, SwapPriceBandWad, FeeFloorWad, FeeCapWad uint64
	DepositCapPoolAsset, MaxSwapNotionalLoan, ReserveTargetLoan                *big.Int
	MaxPerformanceFeeBp                                                        uint16
	GovernanceDelaySec, MaxPriceAgeSec                                         uint32
	Features                                                                   *big.Int
	StartPaused                                                                bool
}

type mpPoolParams struct {
	Core, PoolAsset, LoanAsset, PriceFeed, Allowlist common.Address
	Name, Symbol                                     string
	Risk                                             mpRiskInit
	SafetyGapWad, VenueOracleBandWad                 uint64
	SeedPoolAssets                                   *big.Int
	SeedReceiver                                     common.Address
}

func mpWad(n int64) *big.Int { return new(big.Int).Mul(big.NewInt(n), big.NewInt(1e18)) }

// mpC104Strategy is the c104 genesis strategy (c104.strategy.json `solidity`), deployment fields zero.
func mpC104Strategy() mpHookParams {
	return mpHookParams{
		AWad: mpWad(34), SpanUpWad: mpWad(6), SpanDnWad: mpWad(6), AnchorPriceWad: new(big.Int),
		Tuning: mpTuning{
			Fee: mpFeeParams{MidFeeWad: 3e16, OutFeeWad: 5e15, GammaWad: 5e16, SigmaRefWad: 4e14, VolBetaWad: 4e18,
				VolMinWad: 5e17, VolMaxWad: 2e18, DirSkewWad: 15e16},
			InvSkewKappaWad: 4e18, InvSkewBandWad: 6e16, EmaHalfLife: 4500, RvHalfLife: 1800,
			StepDivisorWad: mpWad(2), InertiaWad: 1e14, InertiaMaxWad: 5e15,
		},
		Bounds: mpBounds{MinFeeWad: 1e15, MaxFeeWad: 5e16, MinEmaHalfLife: 600, MaxEmaHalfLife: 86400,
			MaxRvHalfLife: 86400, MaxStepWad: 2e16, MinStepDivisorWad: mpWad(1)},
	}
}

// mpStrategyB is a second strategy inside the c104 bounds (EverlongHook.sol:177-217): a flatter curve on a narrower
// band and a cheaper, more convex fee row.
func mpStrategyB() mpHookParams {
	p := mpC104Strategy()
	p.AWad, p.SpanUpWad, p.SpanDnWad = mpWad(20), mpWad(4), mpWad(4)
	p.Tuning.Fee = mpFeeParams{MidFeeWad: 2e16, OutFeeWad: 3e15, GammaWad: 8e16, SigmaRefWad: 4e14,
		VolBetaWad: 3e18, VolMinWad: 5e17, VolMaxWad: 2e18, DirSkewWad: 1e17}
	p.Tuning.InvSkewKappaWad, p.Tuning.InvSkewBandWad = 2e18, 5e16
	return p
}

// mpC104Risk is the c104 genesis RiskInit (DeployC104.loadPinned / loadApproved over c104.deployment.json): the
// operating features with SWAP_BUY withheld, born paused.
func mpC104Risk() mpRiskInit {
	return mpRiskInit{PhiWad: 1e18, PhiMinWad: 5e17, PhiMaxWad: 1e18, LtvWad: 55e16, LtvMinWad: 1e17,
		LtvMaxWad: 7e17, LtvMaxStepWad: 1e17, LtvCooldownSec: 3600, MinStructDistWad: 15e16, RoomEpsilonWad: 1e14,
		SwapPriceBandWad: 8e16, FeeFloorWad: 0, FeeCapWad: 1e17, DepositCapPoolAsset: big.NewInt(3_000_000_000),
		MaxSwapNotionalLoan: big.NewInt(1_000_000_000_000), ReserveTargetLoan: new(big.Int),
		MaxPerformanceFeeBp: 5000, GovernanceDelaySec: 172800, MaxPriceAgeSec: 900,
		Features: big.NewInt(mpFeatures &^ 4), StartPaused: true}
}

// mpPoolParamsOf is C104Deployer.poolParams (C104Deployer.sol:156-171) for a pool of the c104 family.
func mpPoolParamsOf(name, symbol string, seed *big.Int, receiver common.Address) mpPoolParams {
	return mpPoolParams{Core: mpC104Core, PoolAsset: c104.PoolAsset, LoanAsset: c104.LoanAsset,
		PriceFeed: c104.PriceFeed, Allowlist: mpC104Allowlist, Name: name, Symbol: symbol, Risk: mpC104Risk(),
		SafetyGapWad: 2e16, VenueOracleBandWad: 2e16, SeedPoolAssets: seed, SeedReceiver: receiver}
}

// mpVenue is C104Deployer.createPool's one Morpho venue (C104Deployer.sol:192-202) on market.
func mpVenue(market morphoMarketParams) venueInit {
	return venueInit{Kind: 0, LoanIndex: 0, VenueParams: encodeMarketParams(market), BorrowEnabled: true,
		SupplyEnabled: true, DebtCap: big.NewInt(1_500_000_000_000), SupplyCap: big.NewInt(2_000_000_000_000),
		MaxBorrowRateWad: 7386586395}
}

func mpMarket(lltv uint64) morphoMarketParams {
	return morphoMarketParams{LoanToken: c104.LoanAsset, CollateralToken: c104.PoolAsset, Oracle: mpC104Oracle,
		Irm: adaptiveCurveIrm, Lltv: new(big.Int).SetUint64(lltv)}
}

// ---------------------------------------------------------------- the test-only registry extension

// mpRegistry puts the fork's own hooks and financing accounts into the package registries, on the fork's chain
// (Base), beside the production tables, and restores the production tables when the test ends. The test is not
// parallel, so Go runs it on its own and resumes no parallel test before it has returned: nothing else reads the
// registries while they are extended.
type mpRegistry struct {
	hooks    []hookEntry
	accounts []boundContract
}

func newMPRegistry(t *testing.T) *mpRegistry {
	r := &mpRegistry{hooks: hookRegistry, accounts: financingAccounts}
	t.Cleanup(func() { hookRegistry, financingAccounts = r.hooks, r.accounts })
	return r
}

// use sets the registries to the production tables plus hooks and accounts.
func (r *mpRegistry) use(hooks []hookEntry, accounts []boundContract) {
	hookRegistry = append(append([]hookEntry(nil), r.hooks...), hooks...)
	financingAccounts = append(append([]boundContract(nil), r.accounts...), accounts...)
}

// usePools registers every hook and account of pools.
func (r *mpRegistry) usePools(pools ...*mpPool) {
	var hooks []hookEntry
	var accounts []boundContract
	for _, p := range pools {
		hooks = append(hooks, p.hookEntries()...)
		accounts = append(accounts, p.accountEntry())
	}
	r.use(hooks, accounts)
}

// ---------------------------------------------------------------- pools on the fork

// mpPool is one pool of the fork with the hooks it was created with.
type mpPool struct {
	*forkEnv
	name                           string
	hook, leverageHook, spreadHook common.Address
	account                        common.Address
	codes                          [3]common.Hash // hook, leverage hook, spread hook
	accountCode, genesis           common.Hash
	// strategy is what a pool this test created was deployed with (the c104 pool's live row is read from chain).
	strategy  mpHookParams
	spreadPpm uint64
}

func (p *mpPool) hookSet() [7]common.Address {
	return [7]common.Address{p.hook, p.hook, p.hook, p.hook, p.leverageHook, p.spreadHook, {}}
}

func (p *mpPool) hookEntries() []hookEntry {
	base := valueobject.ChainIDBase
	return []hookEntry{
		{ChainID: base, Address: p.hook, Kind: hookKindEverlongSwapV1, CodeHash: p.codes[0], Pool: p.pool,
			Factory: c104.Factory, PoolAsset: c104.PoolAsset, LoanAsset: c104.LoanAsset, LoanScale: mpLoanScale,
			GenesisStrategyHash: p.genesis},
		{ChainID: base, Address: p.leverageHook, Kind: hookKindEverlongLeverageV1, CodeHash: p.codes[1], Pool: p.pool,
			LoanScale: mpLoanScale},
		{ChainID: base, Address: p.spreadHook, Kind: hookKindEverlongSpreadV1, CodeHash: p.codes[2], Pool: p.pool},
	}
}

func (p *mpPool) accountEntry() boundContract {
	return boundContract{ChainID: valueobject.ChainIDBase, Address: p.account, CodeHash: p.accountCode, Pool: p.pool}
}

type multipoolEnv struct {
	f                                *anvilFork
	a                                *mpPool // the c104 pool
	reg                              *mpRegistry
	factory, hookArt, levArt, sprArt *mpArtifact
	curator, keeper, whale           common.Address
}

func newMultipoolEnv(t *testing.T, outDir string) *multipoolEnv {
	e := newForkEnv(t)
	m := &multipoolEnv{f: e.f, reg: newMPRegistry(t),
		factory: loadMPArtifact(t, outDir, "FLAMMFactory"),
		hookArt: loadMPArtifact(t, outDir, "EverlongHook"),
		levArt:  loadMPArtifact(t, outDir, "EverlongLeverageHook"),
		sprArt:  loadMPArtifact(t, outDir, "LeverageSpreadHook"),
		whale:   common.HexToAddress("0x00000000000000000000000000000000000beef4"),
	}
	m.a = &mpPool{forkEnv: e, name: "c104", hook: c104.Hook, leverageHook: c104.LeverageHook,
		spreadHook: c104.SpreadHook, account: c104.Account, accountCode: c104.AccountCodeHash,
		codes:   [3]common.Hash{c104.HookCodeHash, c104.LeverageHookCodeHash, c104.SpreadHookCodeHash},
		genesis: c104.GenesisStrategyHash, spreadPpm: 17_500}
	core := e.f.view(c104.Pool, "core")[0].(common.Address)
	require.Equal(t, mpC104Core, core)
	m.curator = e.f.view(core, "owner")[0].(common.Address)
	m.keeper = e.f.view(core, "keeper")[0].(common.Address)
	for _, a := range []common.Address{m.curator, m.keeper, m.whale, forkDeployer} {
		e.f.impersonate(a)
	}
	return m
}

// receipt waits for a mined transaction (fork_harness_test.go send: an upstream fetch can be throttled).
func (m *multipoolEnv) receipt(t *testing.T, hash common.Hash) *types.Receipt {
	t.Helper()
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		if rcpt, err := m.f.geth.TransactionReceipt(context.Background(), hash); err == nil {
			return rcpt
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("tx %s never mined", hash)
	return nil
}

// tx mines one call two seconds after the head and requires it to succeed.
func (m *multipoolEnv) tx(t *testing.T, from, to common.Address, a *abi.ABI, method string, args ...any) {
	t.Helper()
	data, err := a.Pack(method, args...)
	require.NoError(t, err)
	m.f.rpc(nil, "evm_setNextBlockTimestamp", hexutil.Uint64(m.f.head().Time+2))
	var hash common.Hash
	m.f.rpc(&hash, "eth_sendTransaction", map[string]any{"from": from, "to": to, "data": hexutil.Encode(data),
		"gas": hexutil.Uint64(16_000_000)})
	if rcpt := m.receipt(t, hash); rcpt.Status != types.ReceiptStatusSuccessful {
		var out hexutil.Bytes
		callErr := m.f.rc.CallContext(context.Background(), &out, "eth_call", map[string]any{"from": from, "to": to,
			"data": hexutil.Encode(data), "gas": hexutil.Uint64(16_000_000)}, "latest")
		rd, _ := revertData(callErr)
		t.Fatalf("%s from %s reverted: %v %x (%v)", method, from, callErr, rd, revertError(rd))
	}
}

// deploy mines a contract creation from the fork deployer two seconds after the head.
func (m *multipoolEnv) deploy(t *testing.T, code []byte) common.Address {
	t.Helper()
	m.f.rpc(nil, "evm_setNextBlockTimestamp", hexutil.Uint64(m.f.head().Time+2))
	var hash common.Hash
	m.f.rpc(&hash, "eth_sendTransaction", map[string]any{"from": forkDeployer, "data": hexutil.Encode(code),
		"gas": hexutil.Uint64(25_000_000)})
	rcpt := m.receipt(t, hash)
	require.Equal(t, types.ReceiptStatusSuccessful, rcpt.Status, "creation reverted")
	require.NotEqual(t, common.Address{}, rcpt.ContractAddress)
	return rcpt.ContractAddress
}

func (m *multipoolEnv) code(t *testing.T, a common.Address) []byte {
	t.Helper()
	var out hexutil.Bytes
	m.f.rpc(&out, "eth_getCode", a, "latest")
	require.NotEmpty(t, out, "code at %s", a)
	return out
}

func (m *multipoolEnv) call(t *testing.T, to common.Address, a *abi.ABI, method string, args ...any) []any {
	t.Helper()
	data, err := a.Pack(method, args...)
	require.NoError(t, err)
	var out hexutil.Bytes
	m.f.rpc(&out, "eth_call", map[string]any{"to": to, "data": hexutil.Encode(data)}, "latest")
	vals, err := a.Methods[method].Outputs.Unpack(out)
	require.NoError(t, err)
	return vals
}

// strategyHash is EverlongHook._strategyHash (EverlongHook.sol:382-386): keccak256(abi.encode(p)) with the
// deployment fields zeroed.
func (m *multipoolEnv) strategyHash(t *testing.T, p mpHookParams) common.Hash {
	t.Helper()
	p.AnchorPriceWad, p.LoanDecimals = new(big.Int), 0
	packed, err := abi.Arguments{m.hookArt.ABI.Constructor.Inputs[1]}.Pack(p)
	require.NoError(t, err)
	return crypto.Keccak256Hash(packed)
}

// checkC104Wiring shows the bytecode and parameters this test deploys with are the c104 deployment's: each hook's
// linked creation code followed by the c104 constructor arguments is that hook's deployment transaction input, the
// createPool calldata built here is the c104 createPool transaction's, and the strategy and risk hashes are the
// deployment record's.
func (m *multipoolEnv) checkC104Wiring(t *testing.T) {
	t.Helper()
	input := func(h common.Hash) []byte {
		tx, _, err := m.f.geth.TransactionByHash(context.Background(), h)
		require.NoError(t, err, "c104 deployment tx %s", h)
		return tx.Data()
	}
	strategy := mpC104Strategy()
	require.Equal(t, c104.GenesisStrategyHash, m.strategyHash(t, strategy))
	strategy.AnchorPriceWad, strategy.LoanDecimals = mpC104Anchor, 6
	require.Equal(t, input(mpC104Txs.hook), m.hookArt.deployment(t, c104.Pool, strategy), "EverlongHook creation")
	require.Equal(t, input(mpC104Txs.leverageHook), m.levArt.deployment(t, c104.Pool, c104.Hook),
		"EverlongLeverageHook creation")
	require.Equal(t, input(mpC104Txs.spreadHook), m.sprArt.deployment(t, c104.Pool, big.NewInt(17_500),
		big.NewInt(2_500), big.NewInt(100_000), uint32(3600)), "LeverageSpreadHook creation")

	risk, err := abi.Arguments{{Type: *m.factory.ABI.Methods["createPool"].Inputs[0].Type.TupleElems[7]}}.Pack(
		mpC104Risk())
	require.NoError(t, err)
	require.Equal(t, mpC104RiskHash, crypto.Keccak256Hash(risk), "c104 riskHash")
	market := mpMarket(mpLltv86)
	require.Equal(t, mpC104Market, marketID(market))
	calldata, err := m.factory.ABI.Pack("createPool",
		mpPoolParamsOf("Everlong cbBTC", "ev.cbBTC", big.NewInt(26053), mpC104Receiver),
		abiHookSet{c104.Hook, c104.Hook, c104.Hook, c104.Hook, c104.LeverageHook, c104.SpreadHook, common.Address{}},
		crypto.Keccak256Hash([]byte("everlong-c104")), []venueInit{mpVenue(market)})
	require.NoError(t, err)
	require.Equal(t, input(mpC104Txs.createPool), calldata, "c104 createPool calldata")
	t.Logf("c104 wiring reproduced: three hook creations and createPool byte for byte")
}

// supplyMarket makes sure market exists and lends it liquidity from the whale.
func (m *multipoolEnv) supplyMarket(t *testing.T, market morphoMarketParams, liquidity *big.Int) {
	t.Helper()
	id := marketID(market)
	got := m.call(t, c104.Morpho, &venuesABI, "idToMarketParams", id)
	if got[0].(common.Address) == (common.Address{}) {
		m.tx(t, m.whale, c104.Morpho, &venuesABI, "createMarket", market)
	}
	m.f.setBalance(c104.LoanAsset, m.whale, liquidity)
	m.tx(t, m.whale, c104.LoanAsset, &scenariosABI, "approve", c104.Morpho, liquidity)
	m.tx(t, m.whale, c104.Morpho, &scenariosABI, "supply", market, liquidity, big.NewInt(0), m.whale, []byte{})
	t.Logf("market %s (lltv %s) supplied %s", id, market.Lltv, liquidity)
}

// createPool is C104Deployer.deployHook + createPool + the curator's activation (C104Deployer.sol:179-225,
// DeployC104.s.sol:474-481) for a pool of the c104 family with its own strategy, spread and Morpho market: the
// hooks are deployed against the predicted pool, the pool is created through the factory with a seed, then
// unpaused with every operating feature, the keeper posts the spread and the curator arms the leverage venue.
func (m *multipoolEnv) createPool(t *testing.T, name string, strategy mpHookParams, spreadPpm uint64,
	market morphoMarketParams, seed *big.Int) *mpPool {
	t.Helper()
	f := m.f
	factory := c104.Factory
	params := mpPoolParamsOf("Everlong cbBTC "+name, "ev.cbBTC."+name, seed, forkDeployer)
	salt := crypto.Keccak256Hash([]byte("everlong-multipool-" + name))
	predicted := m.call(t, factory, &m.factory.ABI, "predictPool", forkDeployer, salt, params)[0].(common.Address)
	cross := m.call(t, c104.PriceFeed, &mpABI, "cross", c104.PoolAsset, c104.LoanAsset)[0].(*big.Int)

	p := &mpPool{name: name, strategy: strategy, spreadPpm: spreadPpm}
	hookParams := strategy
	hookParams.AnchorPriceWad, hookParams.LoanDecimals = cross, 6
	p.hook = m.deploy(t, m.hookArt.deployment(t, predicted, hookParams))
	p.genesis = m.call(t, p.hook, &hookABI, "genesisStrategyHash")[0].([32]byte)
	require.Equal(t, m.strategyHash(t, strategy), p.genesis, "%s genesisStrategyHash", name)
	p.leverageHook = m.deploy(t, m.levArt.deployment(t, predicted, p.hook))
	p.spreadHook = m.deploy(t, m.sprArt.deployment(t, predicted, new(big.Int).SetUint64(spreadPpm),
		big.NewInt(2_500), big.NewInt(100_000), uint32(3600)))

	f.setBalance(c104.PoolAsset, forkDeployer, seed)
	m.tx(t, forkDeployer, c104.PoolAsset, &scenariosABI, "approve", factory, seed)
	m.tx(t, forkDeployer, factory, &m.factory.ABI, "createPool", params,
		abiHookSet{p.hook, p.hook, p.hook, p.hook, p.leverageHook, p.spreadHook, common.Address{}}, salt,
		[]venueInit{mpVenue(market)})
	require.True(t, m.call(t, factory, &factoryABI, "isPool", predicted)[0].(bool))
	hs, err := tupleOf[abiHookSet](m.call(t, predicted, &flammABI, "hooks")[0])
	require.NoError(t, err)
	require.Equal(t, p.hookSet(), hs.array())
	p.account = m.call(t, c104.Router, &mpABI, "accountOf", predicted, uint8(0), uint8(0))[0].(common.Address)
	venue, err := tupleOf[abiVenueView](m.call(t, c104.Router, &routerABI, "venue", predicted, uint16(0))[0])
	require.NoError(t, err)
	require.Equal(t, p.account, venue.Account)
	require.Equal(t, marketID(market), common.Hash(venue.Id))

	m.tx(t, m.curator, predicted, &scenariosABI, "setPaused", false)
	m.tx(t, m.curator, predicted, &scenariosABI, "setFeatures", big.NewInt(mpFeatures))
	m.tx(t, m.keeper, p.spreadHook, &forkABI, "setSpread", new(big.Int).SetUint64(spreadPpm))
	m.tx(t, m.curator, predicted, &forkABI, "setLevPaused", false)

	p.codes = [3]common.Hash{crypto.Keccak256Hash(m.code(t, p.hook)), crypto.Keccak256Hash(m.code(t, p.leverageHook)),
		crypto.Keccak256Hash(m.code(t, p.spreadHook))}
	p.accountCode = crypto.Keccak256Hash(m.code(t, p.account))
	p.forkEnv = &forkEnv{f: f, code: m.a.code, pool: predicted, cbBTC: c104.PoolAsset, usdc: c104.LoanAsset,
		gas: map[string]uint64{}}
	t.Logf("pool %s %s: hook %s leverage %s spread %s account %s", name, predicted, p.hook, p.leverageHook,
		p.spreadHook, p.account)
	return p
}

// sameCode requires p's hooks to run the live c104 hooks' runtime code but for their immutables, and p's financing
// account to run the c104 account's code exactly.
func (m *multipoolEnv) sameCode(t *testing.T, p *mpPool) {
	t.Helper()
	for _, c := range []struct {
		art        *mpArtifact
		live, mine common.Address
	}{{m.hookArt, c104.Hook, p.hook}, {m.levArt, c104.LeverageHook, p.leverageHook},
		{m.sprArt, c104.SpreadHook, p.spreadHook}} {
		live, mine := m.code(t, c.live), m.code(t, c.mine)
		require.NotEqual(t, crypto.Keccak256Hash(live), crypto.Keccak256Hash(mine), "immutables bind the pool")
		require.Equal(t, c.art.masked(live), c.art.masked(mine), "%s: %s runs %s's code", p.name, c.mine, c.live)
	}
	require.Equal(t, c104.AccountCodeHash, p.accountCode, "%s: every MorphoBlueAccount runs the same code", p.name)
}

var mpABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(`[
 {"type":"function","name":"cross","stateMutability":"view","inputs":[{"name":"base","type":"address"},{"name":"quote","type":"address"}],"outputs":[{"name":"priceWad","type":"uint256"},{"name":"observedAt","type":"uint48"}]},
 {"type":"function","name":"accountOf","stateMutability":"view","inputs":[{"name":"pool","type":"address"},{"name":"idx","type":"uint8"},{"name":"kind","type":"uint8"}],"outputs":[{"name":"","type":"address"}]}
]`))
	if err != nil {
		panic(err)
	}
	return a
}()

// ---------------------------------------------------------------- listing, tracking, quoting

// list runs the lister on the fork (the allow-list pools, when given) from cursor md and returns what it emitted by
// pool, the cursor and what it logged.
func (m *multipoolEnv) list(t *testing.T, md []byte, allow ...*mpPool) (map[common.Address]entity.Pool, []byte,
	string) {
	t.Helper()
	var logs bytes.Buffer
	ctx := zerolog.New(&logs).WithContext(context.Background())
	cfg := baseConfig()
	for _, p := range allow {
		cfg.Pools = append(cfg.Pools, p.pool.Hex())
	}
	out, cursor, err := NewPoolsListUpdater(cfg, m.f.client).GetNewPools(ctx, md)
	require.NoError(t, err)
	got := map[common.Address]entity.Pool{}
	for _, p := range out {
		a := common.HexToAddress(p.Address)
		_, dup := got[a]
		require.False(t, dup, "listed twice: %s", a)
		got[a] = p
	}
	return got, cursor, logs.String()
}

// requireListed requires the listing to be exactly pools, each with its own hook set, pair and registered wiring.
func requireListed(t *testing.T, got map[common.Address]entity.Pool, pools ...*mpPool) {
	t.Helper()
	want := map[common.Address]bool{}
	for _, p := range pools {
		want[p.pool] = true
		e, ok := got[p.pool]
		require.True(t, ok, "%s not listed", p.name)
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(e.StaticExtra), &se))
		require.Equal(t, p.hookSet(), se.Hooks, p.name)
		require.Equal(t, []string{lowerHex(c104.PoolAsset), lowerHex(c104.LoanAsset)},
			[]string{e.Tokens[0].Address, e.Tokens[1].Address}, p.name)
		require.Equal(t, entityVenues(t, e)[0].Account, p.account, p.name)
	}
	for a := range got {
		require.True(t, want[a], "unexpected listing %s", a)
	}
}

// requireRefused requires the lister to have logged pool's refusal with reason.
func requireRefused(t *testing.T, logs string, p *mpPool, reason string) {
	t.Helper()
	for _, line := range strings.Split(logs, "\n") {
		if strings.Contains(line, lowerHex(p.pool)) && strings.Contains(line, "does not match the registry") {
			require.Contains(t, line, reason, p.name)
			return
		}
	}
	t.Fatalf("%s: no refusal logged in %s", p.name, logs)
}

// requireAttested refreshes p at the head under cfg and requires the refresh to attest.
func requireAttested(t *testing.T, p *mpPool, cfg *Config) (entity.Pool, *Extra) {
	t.Helper()
	tracked, err := NewPoolTracker(cfg, p.f.client).GetNewPoolState(context.Background(), p.listed,
		pool.GetNewPoolStateParams{})
	require.NoError(t, err, p.name)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.True(t, extra.Attested, "%s: attest %q drift %q", p.name, extra.AttestFailure, extra.ProfileDrift)
	return tracked, &extra
}

// params is EverlongHook.params() of hook at the fork head.
func (m *multipoolEnv) params(t *testing.T, hook common.Address) mpHookParams {
	t.Helper()
	vals := m.call(t, hook, &m.hookArt.ABI, "params")
	p, err := tupleOf[mpHookParams](vals[0])
	require.NoError(t, err)
	return p
}

// requireOwnHooks requires sim to carry p's hooks: the addresses, the registered kinds, and the kinds' states as
// p's own hooks report them -- which, for a pool this test created, is the strategy it was deployed with.
func (m *multipoolEnv) requireOwnHooks(t *testing.T, sim *PoolSimulator, p *mpPool) {
	t.Helper()
	h := &sim.state.Hooks
	require.Equal(t, p.hookSet(), h.Addrs, p.name)
	require.Equal(t, [3]hookKind{hookKindEverlongSwapV1, hookKindEverlongLeverageV1, hookKindEverlongSpreadV1},
		[3]hookKind{h.Swap.Kind, h.Leverage.Kind, h.Spread.Kind}, p.name)
	chain := m.params(t, p.hook)
	if p != m.a {
		require.Equal(t, m.strategyHash(t, p.strategy), m.strategyHash(t, chain), "%s: the strategy it runs", p.name)
	}
	swap := h.Swap.EverlongSwap
	require.NotNil(t, swap, p.name)
	fee := chain.Tuning.Fee
	require.Equal(t,
		[]uint64{fee.MidFeeWad, fee.OutFeeWad, fee.GammaWad, fee.SigmaRefWad, fee.VolBetaWad, fee.VolMinWad,
			fee.VolMaxWad, fee.DirSkewWad, chain.Tuning.InvSkewKappaWad, chain.Tuning.InvSkewBandWad},
		[]uint64{swap.Fee.MidFeeWad.Uint64(), swap.Fee.OutFeeWad.Uint64(), swap.Fee.GammaWad.Uint64(),
			swap.Fee.SigmaRefWad.Uint64(), swap.Fee.VolBetaWad.Uint64(), swap.Fee.VolMinWad.Uint64(),
			swap.Fee.VolMaxWad.Uint64(), swap.Fee.DirSkewWad.Uint64(), swap.InvSkewKappaWad.Uint64(),
			swap.InvSkewBandWad.Uint64()}, "%s fee row", p.name)
	require.Equal(t, chain.AWad.String(), swap.AWad.Dec(), p.name)
	require.Equal(t, chain.AWad.String(), swap.Support.AWad.Dec(), p.name)
	require.Equal(t, p.spreadPpm, h.Spread.EverlongSpread.Spread.Uint64(), p.name)
}

// preview compares one simulator venue quote with the pool's own preview at block (live_test.go compareLive, on
// any pool of the fork). A size the preview accepts and the port refuses with a settlement-only error is confirmed
// against the settlement through the adapter, which has to revert too.
func (p *mpPool) preview(t *testing.T, sim *PoolSimulator, block uint64, lever, dir bool, amount uint64,
	tally *liveTally) {
	t.Helper()
	method, venue := "previewSwap", VenueSwap
	var f *fill
	var simErr error
	if lever {
		method, venue = "previewLever", VenueLever
		f, simErr = sim.quoteLever(dir, uint256.NewInt(amount), sim.now())
	} else {
		f, simErr = sim.quoteSwap(dir, uint256.NewInt(amount), sim.now())
	}
	where := fmt.Sprintf("%s %s(%v,%d)", p.name, method, dir, amount)
	spreadLive := chainSpreadLive(sim, sim.now())
	data, err := flammABI.Pack(method, dir, new(big.Int).SetUint64(amount))
	require.NoError(t, err)
	var out hexutil.Bytes
	callErr := p.f.rc.CallContext(context.Background(), &out, "eth_call",
		map[string]any{"to": p.pool, "data": hexutil.Encode(data)}, hexutil.EncodeUint64(block))
	if callErr != nil {
		rd, isRevert := revertData(callErr)
		require.True(t, isRevert, "%s: %v", where, callErr)
		want := revertError(rd)
		require.NotNil(t, want, "%s: unmapped revert %x", where, rd)
		if policyRefusal(simErr, spreadLive) {
			tally.policyRefusedRevert++
			return
		}
		require.ErrorIs(t, simErr, want, "%s: the pool reverted %v", where, want)
		tally.chainReverted++
		return
	}
	vals, err := flammABI.Methods[method].Outputs.Unpack(out)
	require.NoError(t, err)
	if simErr != nil {
		if isOneOf(simErr, policyOnly) {
			require.True(t, policyRefusal(simErr, spreadLive), "%s: spread refusal %v while the pool's post answers",
				where, simErr)
			tally.policyRefusedFill++
			return
		}
		require.True(t, isSettlementOnly(simErr), "%s: the pool previews %v, the port refused %v", where, vals, simErr)
		require.Equal(t, block, p.f.head().Number.Uint64(), "%s: settle at the previewed block", where)
		in, outTok := p.tokens(dir)
		settled := p.f.callFill(p.pool, venue, in, outTok, new(big.Int).SetUint64(amount), p.code)
		require.True(t, settled.reverted, "%s: the adapter filled a refused size (%s out)", where, settled.out)
		tally.settlementOnly++
		return
	}
	used, err := wordOf(vals[0])
	require.NoError(t, err)
	got, err := wordOf(vals[1])
	require.NoError(t, err)
	require.Equal(t, used.Dec(), f.used.Dec(), "%s used", where)
	require.Equal(t, got.Dec(), f.out.Dec(), "%s out", where)
	tally.identical++
}

// previewGrid compares sim with p's previews on both venues and directions at sim's block.
func (p *mpPool) previewGrid(t *testing.T, sim *PoolSimulator, n int) {
	t.Helper()
	var tally liveTally
	block := sim.state.Block
	for _, lever := range []bool{false, true} {
		for _, dir := range []struct {
			sell   bool
			lo, hi uint64
		}{{true, 1, 20_000_000}, {false, 100, 20_000_000_000}} {
			amounts := liveGrid(dir.lo, dir.hi, n)
			venue := VenueSwap
			if lever {
				venue = VenueLever
			}
			for _, a := range p.grid(sim, venue, dir.sell) {
				amounts = append(amounts, a.Uint64())
			}
			for _, a := range amounts {
				p.preview(t, sim, block, lever, dir.sell, a, &tally)
			}
		}
	}
	t.Logf("%s previews at %d: %s", p.name, block, tally.String())
	require.Positive(t, tally.identical, p.name)
}

// adapterGrid compares sim with the adapter's eth_call fills on both venues and directions.
func (p *mpPool) adapterGrid(t *testing.T, sim *PoolSimulator) {
	t.Helper()
	before := p.matchedN
	for _, venue := range []uint8{VenueSwap, VenueLever} {
		for _, sell := range []bool{true, false} {
			n := p.matchedN
			for _, a := range p.grid(sim, venue, sell) {
				p.singleFill(t, sim, venue, sell, a)
			}
			require.Greater(t, p.matchedN, n, "%s venue %d sell=%v: no fill matched", p.name, venue, sell)
		}
	}
	t.Logf("%s adapter fills identical: %d", p.name, p.matchedN-before)
}

var (
	mpSellGrid = liveGrid(1_000, 1_000_000, 14)
	mpBuyGrid  = liveGrid(1_000_000, 2_000_000_000, 14)
)

// quoteLines renders sim's forced-venue quotes on a fixed grid at its clock, one line per venue, direction and amount.
func quoteLines(sim *PoolSimulator) []string {
	var out []string
	for _, venue := range []int{int(VenueSwap), int(VenueLever)} {
		for _, sell := range []bool{true, false} {
			grid := mpSellGrid
			if !sell {
				grid = mpBuyGrid
			}
			for _, a := range grid {
				res, err := sim.calcAmountOut(amountIn(sim, sell, a), venue)
				line := fmt.Sprintf("venue %d sell=%v %d: ", venue, sell, a)
				if err != nil {
					line += "refused " + err.Error()
				} else {
					line += fmt.Sprintf("out %s unused %s fee %s gas %d", res.TokenAmountOut.Amount,
						res.RemainingTokenAmountIn.Amount, res.Fee.Amount, res.Gas)
				}
				out = append(out, line)
			}
		}
	}
	return out
}

// quotedOut is a rendered fill's amountOut, or "" for a refusal.
func quotedOut(line string) string {
	_, rest, ok := strings.Cut(line, ": out ")
	if !ok {
		return ""
	}
	out, _, _ := strings.Cut(rest, " ")
	return out
}

// requireQuotesDiffer requires every fill both quote sets make on the swap venue to pay a different amount, and at
// least one such fill.
func requireQuotesDiffer(t *testing.T, what string, a, b []string) {
	t.Helper()
	require.Len(t, b, len(a))
	n, both := 0, 0
	for i := range a {
		if a[i] != b[i] {
			n++
		}
		if strings.HasPrefix(a[i], "venue 0 ") && quotedOut(a[i]) != "" && quotedOut(b[i]) != "" {
			both++
			require.NotEqual(t, quotedOut(a[i]), quotedOut(b[i]), "%s: a swap fill pays alike: %s", what, a[i])
		}
	}
	t.Logf("%s: %d of %d quotes differ (%d swap fills on both, every one paying differently)", what, n, len(a), both)
	require.Positive(t, both, what)
}

// withHooksOf is sim with other's hook states in place of its own: the same pool, the same listed hooks, the other
// pool's hook storage.
func withHooksOf(sim, other *PoolSimulator) *PoolSimulator {
	c := sim.CloneState().(*PoolSimulator)
	h := other.state.Hooks.clone()
	h.Addrs = c.state.Hooks.Addrs
	c.state.Hooks = h
	return c
}

// mpStep is one mined fill of the interleaved sequence.
type mpStep struct {
	pool    int
	venue   int
	sell    bool
	amounts []uint64
}

func TestForkMultiPool(t *testing.T) {
	outDir := os.Getenv(multipoolArtifactsEnv)
	if os.Getenv("EVERLONG_FLAMM_FORK_RPC") == "" || os.Getenv("EVERLONG_ADAPTER_OUT") == "" || outDir == "" {
		t.Skip("EVERLONG_FLAMM_FORK_RPC / EVERLONG_ADAPTER_OUT / " + multipoolArtifactsEnv + " not set")
	}
	m := newMultipoolEnv(t, outDir)
	a := m.a
	ctx := context.Background()
	m.checkC104Wiring(t)

	// ---- two pools of the c104 family on a market of their own, B with its own strategy and C with c104's
	market := mpMarket(mpLltv915)
	m.supplyMarket(t, market, big.NewInt(20_000_000_000))
	seed := big.NewInt(500_000)
	b := m.createPool(t, "b", mpStrategyB(), 12_000, market, seed)
	c := m.createPool(t, "c", mpC104Strategy(), 17_500, market, seed)
	for _, p := range []*mpPool{b, c} {
		m.sameCode(t, p)
	}
	require.NotEqual(t, b.genesis, c.genesis)
	require.Equal(t, c104.GenesisStrategyHash, c.genesis, "C runs the c104 strategy")
	// the c104 pool's leverage venue, armed as fork_parity_test.go arms it
	m.tx(t, m.curator, c104.Pool, &forkABI, "setLevPaused", false)
	m.tx(t, m.keeper, c104.SpreadHook, &forkABI, "setSpread", big.NewInt(17_500))

	// ---- the lister lists a pool only when the registry names every hook it lists, bound to it
	listed, _, logs := m.list(t, nil)
	requireListed(t, listed, a)
	require.NotContains(t, logs, lowerHex(b.pool), "an unregistered pool is not even a candidate")
	require.NotContains(t, logs, lowerHex(c.pool), "an unregistered pool is not even a candidate")

	m.reg.usePools(b)
	listed, md, logs := m.list(t, nil)
	requireListed(t, listed, a, b)
	require.NotContains(t, logs, "does not match")
	listed, _, _ = m.list(t, nil, a, b, c)
	requireListed(t, listed, a, b)
	listed, _, _ = m.list(t, nil, c)
	requireListed(t, listed)
	again, _, _ := m.list(t, md)
	require.Empty(t, again, "the cursor holds both pools")

	for name, edit := range map[string]struct {
		hooks    func(h []hookEntry) []hookEntry
		accounts func(a []boundContract) []boundContract
		pool     *mpPool
		reason   string
	}{
		"C's swap hook alone": {func(h []hookEntry) []hookEntry { return append(h, c.hookEntries()[0]) }, nil, c,
			"leverage hook " + c.leverageHook.Hex() + " is not registered"},
		"C's spread hook bound to B": {func(h []hookEntry) []hookEntry {
			ce := c.hookEntries()
			ce[2].Pool = b.pool
			return append(h, ce...)
		}, nil, c, "spread hook " + c.spreadHook.Hex() + " is bound to " + b.pool.Hex()},
		"B's leverage hook of the spread kind": {func(h []hookEntry) []hookEntry {
			h[1].Kind = hookKindEverlongSpreadV1
			return h
		}, nil, b, "leverage hook " + b.leverageHook.Hex() + " is of kind everlong-spread-v1"},
		"B's swap hook on the c104 codehash": {func(h []hookEntry) []hookEntry {
			h[0].CodeHash = c104.HookCodeHash
			return h
		}, nil, b, "hook codehash"},
		"B's swap hook on the c104 genesis strategy": {func(h []hookEntry) []hookEntry {
			h[0].GenesisStrategyHash = c104.GenesisStrategyHash
			return h
		}, nil, b, "swap hook binding"},
		"B's leverage hook on another loan scale": {func(h []hookEntry) []hookEntry {
			h[1].LoanScale = 1
			return h
		}, nil, b, "leverage hook binding"},
		"B's account not registered": {nil, func([]boundContract) []boundContract { return nil }, b,
			"account(0) codehash"},
		"B's account bound to C": {nil, func(a []boundContract) []boundContract {
			a[0].Pool = c.pool
			return a
		}, b, "venue set"},
	} {
		hooks, accounts := b.hookEntries(), []boundContract{b.accountEntry()}
		if edit.hooks != nil {
			hooks = edit.hooks(hooks)
		}
		if edit.accounts != nil {
			accounts = edit.accounts(accounts)
		}
		m.reg.use(hooks, accounts)
		listed, _, logs = m.list(t, nil)
		want := []*mpPool{a, b}
		if edit.pool == b {
			want = want[:1]
		}
		requireListed(t, listed, want...)
		requireRefused(t, logs, edit.pool, edit.reason)
		t.Logf("registry %s: %s refused (%s)", name, edit.pool.name, edit.reason)
	}

	// B's swap hook registered as of another factory: B is no candidate of this deployment's listing at all, and
	// a listing of B made under the right entry is refused once the entry names another factory.
	{
		bad := b.hookEntries()
		bad[0].Factory = c.pool
		m.reg.use(bad, []boundContract{b.accountEntry()})
		listed, _, logs = m.list(t, nil)
		requireListed(t, listed, a)
		require.NotContains(t, logs, lowerHex(b.pool), "a pool of another factory is not a candidate")
		m.reg.usePools(b)
		listed, _, _ = m.list(t, nil)
		requireListed(t, listed, a, b)
		entityB := listed[b.pool]
		m.reg.use(bad, []boundContract{b.accountEntry()})
		_, err := NewPoolSimulator(entityB)
		require.ErrorIs(t, err, ErrInvalidProfile, "B's listing under a registry naming another factory")
		require.Contains(t, err.Error(), "is of factory "+c.pool.Hex())
		t.Logf("registry B's swap hook of another factory: B is not a candidate, and its listing is refused")
	}

	m.reg.usePools(b, c)
	listed, md2, _ := m.list(t, md)
	requireListed(t, listed, c) // A and B are already listed with this wiring
	listed, _, _ = m.list(t, nil)
	requireListed(t, listed, a, b, c)
	for _, p := range []*mpPool{a, b, c} {
		p.listed = listed[p.pool]
	}

	// ---- every listed pool attests with its own hooks, under the parity and the production policies
	armed := Policy{LeverRouting: true}
	sims := map[*mpPool]*PoolSimulator{}
	head := m.f.head()
	for _, p := range []*mpPool{a, b, c} {
		_, prod := requireAttested(t, p, baseConfig())
		tracked, extra := requireAttested(t, p, parityConfig(armed))
		require.Equal(t, head.Number.Uint64(), tracked.BlockNumber)
		t.Logf("%s attested at %d: %d probes (production policy %d), reserves %v", p.name, tracked.BlockNumber,
			extra.Probes, prod.Probes, tracked.Reserves)
		require.Positive(t, extra.Probes)
		require.Len(t, prod.OracleAhead, 1, "%s: the production policy reads the window", p.name)
		p.tracked = tracked
		sim, err := NewPoolSimulator(tracked)
		require.NoError(t, err, p.name)
		ts := sim.state.Timestamp
		sim.nowFn = func() uint64 { return ts }
		m.requireOwnHooks(t, sim, p)
		require.False(t, sim.state.LevPaused, p.name)
		sims[p] = sim

		deps, _, err := NewPoolTracker(parityConfig(armed), nil).GetDependencies(ctx, tracked)
		require.NoError(t, err)
		for _, q := range []*mpPool{a, b, c} {
			for _, h := range []common.Address{q.hook, q.spreadHook} {
				if q == p {
					require.Contains(t, deps, lowerHex(h), p.name)
				} else {
					require.NotContains(t, deps, lowerHex(h), "%s depends on %s's hooks", p.name, q.name)
				}
			}
		}
		hop := msgpackHop(t, sim)
		hop.nowFn = sim.nowFn
		require.Equal(t, quoteLines(sim), quoteLines(hop), "%s through msgpack", p.name)
	}
	simA, simB, simC := sims[a], sims[b], sims[c]
	require.Equal(t, simB.state.Timestamp, simC.state.Timestamp)

	// ---- per-pool dispatch: B and C differ only in their hooks, and quote each other's once the hooks are swapped
	for _, line := range e2eDiff(withHooksOf(simB, simC).state, simC.state) {
		t.Logf("B with C's hook states vs C: %s", line)
		require.True(t, strings.HasPrefix(line, "hook set:") || strings.HasPrefix(line, "router.venue[0]:"),
			"B and C differ beyond their hook addresses and venue accounts: %s", line)
	}
	qA, qB, qC := quoteLines(simA), quoteLines(simB), quoteLines(simC)
	requireQuotesDiffer(t, "c104 vs B", qA, qB)
	requireQuotesDiffer(t, "B vs C", qB, qC)
	require.Equal(t, qC, quoteLines(withHooksOf(simB, simC)), "B's pool state with C's hook states quotes C")
	require.Equal(t, qB, quoteLines(withHooksOf(simC, simB)), "C's pool state with B's hook states quotes B")
	require.Equal(t, qA, quoteLines(simA), "quoting B and C left the c104 simulator alone")

	// ---- every pool quotes its own previews and the adapter's fills on both venues
	for _, p := range []*mpPool{a, b, c} {
		p.previewGrid(t, sims[p], 24)
		p.adapterGrid(t, sims[p])
	}

	// ---- unregistering C's hooks fails C closed: the simulator, the tracker, and the lister's cursor
	m.reg.usePools(b)
	_, err := simC.calcAmountOut(amountIn(simC, true, 5_000), -1)
	require.ErrorIs(t, err, ErrInvalidProfile)
	require.Contains(t, err.Error(), "is not registered")
	_, err = NewPoolSimulator(c.tracked)
	require.ErrorIs(t, err, ErrInvalidProfile)
	_, err = NewPoolTracker(parityConfig(armed), m.f.client).GetNewPoolState(ctx, c.tracked,
		pool.GetNewPoolStateParams{})
	require.ErrorIs(t, err, ErrInvalidProfile)
	listed, md3, _ := m.list(t, md2)
	require.Empty(t, listed)
	var cursor Metadata
	require.NoError(t, json.Unmarshal(md3, &cursor))
	require.Contains(t, cursor.Listed, lowerHex(a.pool))
	require.Contains(t, cursor.Listed, lowerHex(b.pool))
	require.NotContains(t, cursor.Listed, lowerHex(c.pool), "an unregistered pool leaves the cursor")
	_, err = simB.calcAmountOut(amountIn(simB, true, 5_000), -1)
	require.NotErrorIs(t, err, ErrInvalidProfile, "B is still registered")

	// ---- a chained sequence of mined fills interleaved across the c104 pool and B, each simulator advanced only by
	// its own UpdateBalance; neither pool's fills move the other's quotes, on chain or in memory
	ups := []uint64{9_000, 6_000, 4_000, 2_500, 1_500, 1_000, 600}
	downs := []uint64{20_000_000, 10_000_000, 5_000_000, 2_500_000, 1_200_000, 600_000, 300_000, 150_000}
	sells := []uint64{60_000, 30_000, 15_000, 7_000, 3_000}
	buys := []uint64{40_000_000, 20_000_000, 8_000_000, 3_000_000, 1_000_000}
	pools := []*mpPool{a, b}
	steps := []mpStep{
		{0, int(VenueLever), true, ups}, {1, int(VenueLever), true, ups},
		{0, int(VenueSwap), true, []uint64{15_000}}, {1, int(VenueSwap), true, sells},
		{0, int(VenueLever), false, downs}, {1, int(VenueLever), false, downs},
		{0, int(VenueSwap), false, []uint64{20_000_000}}, {1, int(VenueSwap), false, buys},
		{0, -1, false, []uint64{8_000_000}}, {1, -1, true, sells},
		{0, int(VenueSwap), true, []uint64{60_000}}, {1, int(VenueLever), true, ups},
		{0, int(VenueLever), false, downs}, {1, int(VenueSwap), false, buys},
		{1, int(VenueLever), false, downs}, {0, int(VenueLever), true, ups},
		{1, int(VenueSwap), true, sells}, {0, int(VenueSwap), false, []uint64{45_000_000}},
		{1, -1, false, buys}, {0, int(VenueSwap), true, []uint64{30_000}},
	}
	seqSims := map[*mpPool]*PoolSimulator{a: simA.CloneState().(*PoolSimulator), b: simB.CloneState().(*PoolSimulator)}
	fills := map[string]int{}
	start := m.f.head().Time
	for i, s := range steps {
		p, other := pools[s.pool], pools[1-s.pool]
		ts := start + uint64(4*(i+1))
		otherSim := seqSims[other]
		otherSim.nowFn = func() uint64 { return ts }
		before := quoteLines(otherSim)
		t.Logf("step %d: %s venue %d sell=%v", i, p.name, s.venue, s.sell)
		if p.seqFill(t, seqSims[p], s.venue, s.sell, s.amounts, ts) {
			fills[fmt.Sprintf("%s/venue%d/sell=%v", p.name, s.venue, s.sell)]++
		}
		require.Equal(t, before, quoteLines(otherSim), "step %d: %s's fill moved %s's simulator", i, p.name, other.name)
	}
	t.Logf("interleaved fills: %v", fills)
	for _, p := range pools {
		for _, k := range []string{"venue0/sell=true", "venue0/sell=false", "venue1/sell=true", "venue1/sell=false"} {
			require.Positive(t, fills[p.name+"/"+k], "%s: no mined %s", p.name, k)
		}
	}
	// A SwapInfo is accepted only by the pool it was quoted on.
	res, err := seqSims[b].calcAmountOut(amountIn(seqSims[b], false, 5_000_000), int(VenueSwap))
	require.NoError(t, err)
	foreign := seqSims[a].CloneState().(*PoolSimulator)
	foreign.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
	_, err = foreign.calcAmountOut(amountIn(foreign, true, 5_000), -1)
	require.ErrorIs(t, err, ErrSwapInfoMismatch, "B's SwapInfo on the c104 simulator")
	fresh := map[*mpPool]*PoolSimulator{}
	for _, p := range pools {
		fresh[p] = p.track(t, armed)
		sameState(t, seqSims[p], fresh[p])
		m.requireOwnHooks(t, fresh[p], p)
		t.Logf("%s after the sequence: state equals a refresh; receipt gas maxima %v", p.name, p.gas)
	}
	b.previewGrid(t, fresh[b], 12)

	// ---- a keeper retunes B's fee row: B's quotes move, the c104 pool's state does not
	row := mpFeeParams{MidFeeWad: 4e16, OutFeeWad: 1e16, GammaWad: 5e16, SigmaRefWad: 4e14, VolBetaWad: 4e18,
		VolMinWad: 5e17, VolMaxWad: 2e18, DirSkewWad: 15e16}
	m.tx(t, m.keeper, b.hook, &m.hookArt.ABI, "setFeeRow", row, uint64(4e18), uint64(6e16))
	retunedA, retunedB := a.track(t, armed), b.track(t, armed)
	sameState(t, fresh[a], retunedA)
	diff := e2eDiff(func() *flammState {
		s := fresh[b].state.clone()
		s.Block, s.Timestamp = retunedB.state.Block, retunedB.state.Timestamp
		return s
	}(), retunedB.state)
	require.Len(t, diff, 1, "only B's swap hook moved: %v", diff)
	require.True(t, strings.HasPrefix(diff[0], "hook:"), diff[0])
	b.strategy.Tuning.Fee, b.strategy.Tuning.InvSkewKappaWad, b.strategy.Tuning.InvSkewBandWad = row, 4e18, 6e16
	m.requireOwnHooks(t, retunedB, b)
	ts := retunedB.state.Timestamp
	fresh[b].nowFn = func() uint64 { return ts }
	requireQuotesDiffer(t, "B before vs after the retune", quoteLines(fresh[b]), quoteLines(retunedB))
	b.previewGrid(t, retunedB, 12)

	// ---- C, registered again, is refreshed onto the market B's fills moved and runs a chained sequence of its own
	m.reg.usePools(b, c)
	seqC := c.track(t, armed)
	m.requireOwnHooks(t, seqC, c)
	cFills := map[string]int{}
	startC := m.f.head().Time
	for i, s := range []struct {
		venue   int
		sell    bool
		amounts []uint64
	}{
		{int(VenueSwap), true, sells}, {int(VenueLever), true, ups}, {int(VenueSwap), false, buys},
		{int(VenueLever), false, downs}, {-1, true, sells}, {int(VenueLever), true, ups}, {-1, false, buys},
		{int(VenueLever), false, downs},
	} {
		if c.seqFill(t, seqC, s.venue, s.sell, s.amounts, startC+uint64(4*(i+1))) {
			cFills[fmt.Sprintf("venue%d/sell=%v", s.venue, s.sell)]++
		}
	}
	t.Logf("C's fills: %v", cFills)
	for _, k := range []string{"venue0/sell=true", "venue0/sell=false", "venue1/sell=true", "venue1/sell=false"} {
		require.Positive(t, cFills[k], "C: no mined %s", k)
	}
	sameState(t, seqC, c.track(t, armed))

	// ---- static data naming another pool's registered hook is refused, by the simulator and by the tracker
	m.reg.usePools(b)
	tampered := func(tracked entity.Pool, edit func(se *StaticExtra)) entity.Pool {
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(tracked.StaticExtra), &se))
		edit(&se)
		raw, err := json.Marshal(&se)
		require.NoError(t, err)
		tracked.StaticExtra = string(raw)
		return tracked
	}
	hotSims := map[*mpPool]*PoolSimulator{a: retunedA, b: retunedB}
	for _, tc := range []struct {
		name   string
		p      *mpPool
		edit   func(se *StaticExtra)
		reason string
	}{
		{"B listing c104's swap hook", b, func(se *StaticExtra) {
			se.Hooks[0], se.Hooks[1], se.Hooks[2], se.Hooks[3] = a.hook, a.hook, a.hook, a.hook
		}, "swap hook " + a.hook.Hex() + " is bound to " + a.pool.Hex()},
		{"B listing c104's leverage hook", b, func(se *StaticExtra) { se.Hooks[4] = a.leverageHook },
			"leverage hook " + a.leverageHook.Hex() + " is bound to " + a.pool.Hex()},
		{"B listing c104's spread hook", b, func(se *StaticExtra) { se.Hooks[5] = a.spreadHook },
			"spread hook " + a.spreadHook.Hex() + " is bound to " + a.pool.Hex()},
		{"B listing c104's whole hook set", b, func(se *StaticExtra) { se.Hooks = a.hookSet() },
			"is bound to " + a.pool.Hex()},
		{"c104 listing B's spread hook", a, func(se *StaticExtra) { se.Hooks[5] = b.spreadHook },
			"spread hook " + b.spreadHook.Hex() + " is bound to " + b.pool.Hex()},
		{"B listing C's leverage hook", b, func(se *StaticExtra) { se.Hooks[4] = c.leverageHook },
			"leverage hook " + c.leverageHook.Hex() + " is not registered"},
	} {
		listing := tampered(tc.p.tracked, tc.edit)
		_, err := NewPoolSimulator(listing)
		require.ErrorIs(t, err, ErrInvalidProfile, tc.name)
		require.Contains(t, err.Error(), tc.reason, tc.name)
		_, err = NewPoolTracker(parityConfig(armed), m.f.client).GetNewPoolState(ctx, listing,
			pool.GetNewPoolStateParams{})
		require.ErrorIs(t, err, ErrInvalidProfile, tc.name)
		require.Contains(t, err.Error(), tc.reason, tc.name)
		// the same edit on a simulator already built, its state naming the same hooks
		hot := hotSims[tc.p].CloneState().(*PoolSimulator)
		tc.edit(&hot.StaticExtra)
		hot.state.Hooks.Addrs = hot.StaticExtra.Hooks
		_, err = hot.calcAmountOut(amountIn(hot, true, 5_000), -1)
		require.ErrorIs(t, err, ErrInvalidProfile, tc.name)
		require.Contains(t, err.Error(), tc.reason, tc.name)
		t.Logf("static tamper %s: refused (%s)", tc.name, tc.reason)
	}
	moved := retunedB.CloneState().(*PoolSimulator)
	moved.Info.Address = lowerHex(a.pool)
	_, err = moved.calcAmountOut(amountIn(moved, true, 5_000), -1)
	require.ErrorIs(t, err, ErrInvalidProfile, "B's listing on the c104 pool")

	// ---- a hook slot rewritten on chain to another pool's hook: B's refresh drifts, the lister refuses B
	slot := flammSlot(slotLevHook)
	var word hexutil.Bytes
	m.f.rpc(&word, "eth_getStorageAt", b.pool, slot, "latest")
	original := common.BytesToHash(word)
	require.Equal(t, common.BytesToHash(b.leverageHook[:]), original, "the FLAMMStore leverage-hook slot holds B's")
	m.f.rpc(nil, "anvil_setStorageAt", b.pool, slot, common.BytesToHash(a.leverageHook[:]))
	drifted, err := NewPoolTracker(parityConfig(armed), m.f.client).GetNewPoolState(ctx, b.listed,
		pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(drifted.Extra), &extra))
	require.False(t, extra.Attested)
	require.Equal(t, "hook set", extra.ProfileDrift)
	require.Equal(t, entity.PoolReserves{"0", "0"}, drifted.Reserves)
	_, err = NewPoolSimulator(drifted)
	require.ErrorIs(t, err, ErrProfileDrift)
	listed, _, logs = m.list(t, nil)
	requireListed(t, listed, a)
	requireRefused(t, logs, b, "leverage hook "+a.leverageHook.Hex()+" is bound to "+a.pool.Hex())
	m.f.rpc(nil, "anvil_setStorageAt", b.pool, slot, original)
	requireAttested(t, b, parityConfig(armed))
	listed, _, _ = m.list(t, nil)
	requireListed(t, listed, a, b)

	t.Logf("multi-pool fork: c104 %d, B %d, C %d adapter fills identical", a.matchedN, b.matchedN, c.matchedN)
}
