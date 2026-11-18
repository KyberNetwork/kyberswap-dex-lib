package nightlyprop

// These tests compare Go quotes with the compiled pool and router in a local EVM.

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"math/big"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

// evmGasLimit bounds each call, as a node bounds eth_call.
const evmGasLimit = 30_000_000

// evmABI holds the pool's own quote, which the test compares with the simulator.
var evmABI = lo.Must(abi.JSON(strings.NewReader(`[
 {"type":"function","name":"quoteSwapExactIn","stateMutability":"nonpayable",
  "inputs":[{"name":"tokenIn","type":"address"},{"name":"amountIn","type":"uint256"}],
  "outputs":[{"name":"","type":"tuple","components":[{"name":"amountIn","type":"uint256"},
   {"name":"amountOut","type":"uint256"},{"name":"endPosition","type":"tuple","components":[
   {"name":"levelId","type":"int16"},{"name":"fraction","type":"uint104"}]}]}]}
]`)))

// fixtureAccounts is testdata/router-fixture.json.
type fixtureAccounts struct {
	Router, Partner, WethUsdc, Weth, Usdc common.Address
	BlockNumber, BlockTimestamp           uint64
}

// evmFixture is the router fixture's state in an in-process EVM, which client reaches over
// JSON-RPC, so the tracker runs as it runs against a node.
type evmFixture struct {
	accounts fixtureAccounts
	client   *ethrpc.Client

	mu    sync.Mutex
	state *state.StateDB
}

func loadEVMFixture(t testing.TB) *evmFixture {
	t.Helper()
	var accounts fixtureAccounts
	accountsJSON, err := os.ReadFile("testdata/router-fixture.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(accountsJSON, &accounts))

	file, err := os.Open("testdata/router-fixture-state.json.gz")
	require.NoError(t, err)
	defer func() { require.NoError(t, file.Close()) }()
	reader, err := gzip.NewReader(file)
	require.NoError(t, err)
	var dump map[common.Address]struct {
		Nonce   hexutil.Uint64
		Balance *hexutil.Big
		Code    hexutil.Bytes
		Storage map[common.Hash]common.Hash
	}
	require.NoError(t, json.NewDecoder(reader).Decode(&dump))

	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)
	for address, account := range dump {
		statedb.SetNonce(address, uint64(account.Nonce), tracing.NonceChangeUnspecified)
		statedb.SetBalance(address, uint256.MustFromBig(account.Balance.ToInt()), tracing.BalanceChangeUnspecified)
		statedb.SetCode(address, account.Code, tracing.CodeChangeUnspecified)
		for key, value := range account.Storage {
			statedb.SetState(address, key, value)
		}
	}
	statedb.Finalise(true)

	f := &evmFixture{accounts: accounts, state: statedb}
	server := httptest.NewServer(http.HandlerFunc(f.serveRPC))
	t.Cleanup(server.Close)
	f.client = ethrpc.New(server.URL)
	return f
}

// config is the fixture's block: its number and time, with every fork through Osaka, which the
// contracts are compiled for.
func (f *evmFixture) config(origin common.Address, statedb *state.StateDB) *runtime.Config {
	return &runtime.Config{
		ChainConfig: params.MergedTestChainConfig,
		Origin:      origin,
		BlockNumber: new(big.Int).SetUint64(f.accounts.BlockNumber),
		Time:        f.accounts.BlockTimestamp,
		GasLimit:    evmGasLimit,
		BaseFee:     new(big.Int),
		BlobBaseFee: new(big.Int),
		Random:      &common.Hash{},
		State:       statedb,
	}
}

// call runs a call on a copy of the state, as eth_call does.
func (f *evmFixture) call(origin, to common.Address, input []byte) ([]byte, error) {
	f.mu.Lock()
	statedb := f.state.Copy()
	f.mu.Unlock()
	output, _, err := runtime.Call(to, input, f.config(origin, statedb))
	return output, err
}

// serveRPC answers eth_call, the only method that the tracker sends, with call.
func (f *evmFixture) serveRPC(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ID     json.RawMessage   `json:"id"`
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	response := map[string]any{"jsonrpc": "2.0"}
	defer func() {
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Method != "eth_call" ||
		len(request.Params) == 0 {
		response["error"] = map[string]any{"code": -32601, "message": "only eth_call is served"}
		return
	}
	response["id"] = request.ID
	var message struct {
		From  common.Address `json:"from"`
		To    common.Address `json:"to"`
		Input hexutil.Bytes  `json:"input"`
	}
	if err := json.Unmarshal(request.Params[0], &message); err != nil {
		response["error"] = map[string]any{"code": -32602, "message": err.Error()}
		return
	}
	output, err := f.call(message.From, message.To, message.Input)
	switch {
	case errors.Is(err, vm.ErrExecutionReverted):
		response["error"] = map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Encode(output)}
	case err != nil:
		response["error"] = map[string]any{"code": -32000, "message": err.Error()}
	default:
		response["result"] = hexutil.Encode(output)
	}
}

// record is the pool record that GetNewPools lists for pool.
func (f *evmFixture) record(pool, base, quote common.Address) entity.Pool {
	return entity.Pool{
		Address:  hexutil.Encode(pool[:]),
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(base[:]), Swappable: true},
			{Address: hexutil.Encode(quote[:]), Swappable: true},
		},
		StaticExtra: `{"routerAddress":"` + hexAddress(f.accounts.Router) + `"}`,
	}
}

// poolQuote is the pool's own quoteSwapExactIn, as a simulation from the zero address.
func (f *evmFixture) poolQuote(pool, tokenIn common.Address, amountIn *uint256.Int) (*big.Int, error) {
	input, err := evmABI.Pack("quoteSwapExactIn", tokenIn, amountIn.ToBig())
	if err != nil {
		return nil, err
	}
	output, err := f.call(common.Address{}, pool, input)
	if err != nil {
		return nil, err
	}
	var quote struct {
		Value struct {
			AmountIn, AmountOut *big.Int
			EndPosition         struct {
				LevelId  int16
				Fraction *big.Int
			}
		}
	}
	if err := evmABI.UnpackIntoInterface(&quote, "quoteSwapExactIn", output); err != nil {
		return nil, err
	}
	return quote.Value.AmountOut, nil
}

// track refreshes the record of pool with the tracker, over the fixture's RPC.
func (f *evmFixture) track(t *testing.T, pool, base, quote common.Address) entity.Pool {
	t.Helper()
	tracker, err := NewPoolTracker(&Config{KyberExecutor: hexAddress(f.accounts.Partner)}, f.client)
	require.NoError(t, err)
	tracked, err := tracker.GetNewPoolState(t.Context(), f.record(pool, base, quote),
		poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)
	return tracked
}

// The fixture uses its saved block time for contract calls.
func (f *evmFixture) simulator(t *testing.T, pool, base, quote common.Address) *PoolSimulator {
	t.Helper()
	sim, err := NewPoolSimulator(poolpkg.FactoryParams{EntityPool: f.track(t, pool, base, quote)})
	require.NoError(t, err)
	return sim
}

func evmUnits(amount uint64, decimals uint8) *uint256.Int {
	return new(uint256.Int).Mul(uint256.NewInt(amount), new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(uint64(decimals))))
}

func hexAddress(address common.Address) string {
	return hexutil.Encode(address[:])
}

// A saved pool must follow the current executor when its permissions change.
func TestEVM_TrackerUsesConfiguredExecutor(t *testing.T) {
	f := loadEVMFixture(t)
	a := f.accounts
	record := f.record(a.WethUsdc, a.Weth, a.Usdc)
	oldExecutor := "0x000000000000000000000000000000000000cafe"
	record.StaticExtra = `{"routerAddress":"` + hexAddress(a.Router) + `","partnerAddress":"` + oldExecutor + `"}`

	for _, tc := range []struct {
		executor string
		enabled  bool
	}{{oldExecutor, false}, {hexAddress(a.Partner), true}} {
		t.Run(tc.executor, func(t *testing.T) {
			tracker, err := pooltrack.Factory(DexType)(DexType, pooltrack.FactoryParams{
				Properties:   pooltrack.Properties{"kyberExecutor": tc.executor},
				Dependencies: pooltrack.Dependencies{EthrpcClient: f.client},
			})
			require.NoError(t, err)
			tracked, err := tracker.GetNewPoolState(t.Context(), record, poolpkg.GetNewPoolStateParams{})
			require.NoError(t, err)
			require.Equal(t, tc.enabled, tracked.HasReserves())
			require.JSONEq(t, `{"routerAddress":"`+hexAddress(a.Router)+`","kyberExecutor":"`+tc.executor+`"}`, tracked.StaticExtra)
			var extra Extra
			require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
			require.Equal(t, tc.enabled, extra.PartnerRole)
		})
	}
}

// TestEVM_QuotesMatchPool: the tracked state lets the fixture's partner swap, and on every amount,
// from one unit to more than the pool holds, in both directions, the simulator quotes what the
// pool's own quoteSwapExactIn returns, and fails where the pool reverts.
func TestEVM_QuotesMatchPool(t *testing.T) {
	f := loadEVMFixture(t)
	a := f.accounts
	sim := f.simulator(t, a.WethUsdc, a.Weth, a.Usdc)
	require.NoError(t, sim.Extra.swapError())

	type swap struct {
		tokenIn, tokenOut common.Address
		amount            *uint256.Int
	}
	swaps := []swap{
		{a.Usdc, a.Weth, uint256.NewInt(1)},
		{a.Usdc, a.Weth, evmUnits(7_000, 6)},
		// Level 0 drains at 20,010 USDC, and all 20 WETH cost 40,040 USDC.
		{a.Usdc, a.Weth, evmUnits(20_010, 6)},
		{a.Usdc, a.Weth, new(uint256.Int).AddUint64(evmUnits(20_010, 6), 1)},
		{a.Usdc, a.Weth, evmUnits(40_040, 6)},
		{a.Usdc, a.Weth, new(uint256.Int).AddUint64(evmUnits(40_040, 6), 1)},
		{a.Weth, a.Usdc, uint256.NewInt(1)},
		{a.Weth, a.Usdc, evmUnits(5, 18)},
		{a.Weth, a.Usdc, evmUnits(11, 18)},
	}
	r := rand.New(rand.NewPCG(0x5eed, 10))
	for range 300 {
		in, out, limit := a.Usdc, a.Weth, evmUnits(45_000, 6)
		if r.IntN(2) == 0 {
			in, out, limit = a.Weth, a.Usdc, evmUnits(12, 18)
		}
		require.True(t, limit.IsUint64(), "random input limit must fit in uint64")
		amount := uint256.NewInt(r.Uint64N(limit.Uint64()) + 1)
		swaps = append(swaps, swap{in, out, amount})
	}

	matched, reverted := 0, 0
	for _, s := range swaps {
		want, poolErr := f.poolQuote(a.WethUsdc, s.tokenIn, s.amount)
		got, simErr := quote(sim, hexAddress(s.tokenIn), s.amount, hexAddress(s.tokenOut))
		name := hexAddress(s.tokenIn) + " in, amount " + s.amount.Dec()
		if poolErr != nil {
			require.Error(t, simErr, "%s: the pool reverts, the simulator quotes", name)
			reverted++
			continue
		}
		require.NoError(t, simErr, name)
		require.Equal(t, want.String(), got.TokenAmountOut.Amount.String(), name)
		matched++
	}
	t.Logf("%d quotes match the pool, %d revert on both", matched, reverted)
	require.Positive(t, matched)
	require.Positive(t, reverted)
}
