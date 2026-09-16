package everlongflamm

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// An anvil fork of Base with the EverlongFlammAdapter runtime from its forge artifact, driven the way the KyberSwap
// executor drives an adapter: the input is already in the adapter when executeEverlongFlamm runs. Single fills are
// eth_calls with the adapter's code and balance as state overrides (nothing is mined); sequential fills are mined
// transactions at the timestamp the simulator quoted them at.

// forkDeployer is anvil's first unlocked account.
var forkDeployer = common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")

// forkAdapter is where the adapter runtime is placed.
var forkAdapter = common.HexToAddress("0x00000000000000000000000000000000000ada97")

// forkRecipient receives every output.
var forkRecipient = common.HexToAddress("0x00000000000000000000000000000000000000a1")

// fiatTokenBalanceSlot is FiatTokenV2_2's balanceAndBlacklistStates mapping (slot 9); both cbBTC and USDC on Base
// keep balances there (checked against balanceOf before use).
const fiatTokenBalanceSlot = 9

const forkABIJSON = `[
 {"type":"function","name":"executeEverlongFlamm","stateMutability":"payable","inputs":[{"name":"data","type":"bytes"},{"name":"amountIn","type":"uint256"},{"name":"tokenIn","type":"address"},{"name":"tokenOut","type":"address"},{"name":"recipient","type":"address"}],"outputs":[{"name":"amountUnused","type":"uint256"},{"name":"amountOut","type":"uint256"}]},
 {"type":"function","name":"balanceOf","stateMutability":"view","inputs":[{"name":"who","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
 {"type":"function","name":"core","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
 {"type":"function","name":"owner","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
 {"type":"function","name":"keeper","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
 {"type":"function","name":"setLevPaused","stateMutability":"nonpayable","inputs":[{"name":"p","type":"bool"}],"outputs":[]},
 {"type":"function","name":"setSpread","stateMutability":"nonpayable","inputs":[{"name":"s","type":"uint24"}],"outputs":[]},
 {"type":"function","name":"setMaxSpreadAge","stateMutability":"nonpayable","inputs":[{"name":"a","type":"uint32"}],"outputs":[]},
 {"type":"function","name":"setLoanConfig","stateMutability":"nonpayable","inputs":[{"name":"idx","type":"uint8"},{"name":"bandWad","type":"uint64"},{"name":"feeFloorWad","type":"uint64"},{"name":"maxSwapNotional","type":"uint256"},{"name":"reserveTarget","type":"uint256"}],"outputs":[]}
]`

var forkABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(forkABIJSON))
	if err != nil {
		panic(err)
	}
	return a
}()

type anvilFork struct {
	t      *testing.T
	url    string
	rc     *rpc.Client
	geth   *ethclient.Client
	client *ethrpc.Client
}

func startFork(t *testing.T, upstream string, block uint64) *anvilFork {
	t.Helper()
	if _, err := exec.LookPath("anvil"); err != nil {
		t.Skip("anvil not installed")
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	cmd := exec.Command("anvil", "--fork-url", upstream, "--fork-block-number", fmt.Sprint(block), "--port",
		fmt.Sprint(port), "--silent", "--retries", "40", "--fork-retry-backoff", "750", "--timeout", "120000",
		"--gas-limit", "300000000", "--code-size-limit", "100000")
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	for i := 0; ; i++ {
		if conn, err := net.DialTimeout("tcp", l.Addr().String(), 200*time.Millisecond); err == nil {
			_ = conn.Close()
			break
		}
		require.Less(t, i, 150, "anvil never came up")
		time.Sleep(200 * time.Millisecond)
	}
	rc, err := rpc.Dial(url)
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	f := &anvilFork{t: t, url: url, rc: rc, geth: ethclient.NewClient(rc)}
	f.client = ethrpc.New(url).SetMulticallContract(multicall3)
	return f
}

func (f *anvilFork) rpc(result any, method string, args ...any) {
	f.t.Helper()
	require.NoError(f.t, f.rc.CallContext(context.Background(), result, method, args...), method)
}

func (f *anvilFork) head() *types.Header {
	f.t.Helper()
	h, err := f.geth.HeaderByNumber(context.Background(), nil)
	require.NoError(f.t, err)
	return h
}

// placeAdapter puts the adapter runtime at forkAdapter (no transaction).
func (f *anvilFork) placeAdapter(artifact string) []byte {
	f.t.Helper()
	raw, err := os.ReadFile(artifact)
	require.NoError(f.t, err)
	var art struct {
		DeployedBytecode struct {
			Object string `json:"object"`
		} `json:"deployedBytecode"`
	}
	require.NoError(f.t, json.Unmarshal(raw, &art))
	code := common.FromHex(art.DeployedBytecode.Object)
	require.NotEmpty(f.t, code)
	f.rpc(nil, "anvil_setCode", forkAdapter, hexutil.Encode(code))
	return code
}

func balanceSlot(holder common.Address) common.Hash {
	var key [64]byte
	copy(key[12:32], holder[:])
	key[63] = fiatTokenBalanceSlot
	return crypto.Keccak256Hash(key[:])
}

func (f *anvilFork) balance(token, who common.Address) *big.Int {
	f.t.Helper()
	data, err := forkABI.Pack("balanceOf", who)
	require.NoError(f.t, err)
	var out hexutil.Bytes
	f.rpc(&out, "eth_call", map[string]any{"to": token, "data": hexutil.Encode(data)}, "latest")
	return new(big.Int).SetBytes(out)
}

// setBalance writes a FiatToken balance and checks balanceOf reads it back.
func (f *anvilFork) setBalance(token, who common.Address, amount *big.Int) {
	f.t.Helper()
	f.rpc(nil, "anvil_setStorageAt", token, balanceSlot(who), common.BigToHash(amount))
	require.Zero(f.t, f.balance(token, who).Cmp(amount), "balance slot of %s", token)
}

// constantPriceCode is runtime code that answers `price()` with one fixed word and reverts on anything else:
//
//	CALLDATALOAD(0) >> 224 == 0xa035b1fe ? jump 0x14 : revert(0,0); PUSH32 value; MSTORE(0); RETURN(0,32)
//
// It is what pinOracles puts at a venue's Morpho market oracle.
func constantPriceCode(value *big.Int) []byte {
	code := []byte{0x60, 0x00, 0x35, 0x60, 0xe0, 0x1c, 0x63, 0xa0, 0x35, 0xb1, 0xfe, 0x14, 0x60, 0x14, 0x57, 0x60,
		0x00, 0x60, 0x00, 0xfd, 0x5b, 0x7f}
	code = append(code, common.BigToHash(value).Bytes()...)
	return append(code, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3)
}

// pinOracles freezes every venue's Morpho market oracle at the answer it gives at the fork head, by replacing its
// code with one that returns exactly that word. Nothing else changes: the pinned answer is the oracle's own, so
// every read, probe and mined fill at the fork head is what it was. It runs once, at the fork's own venue set; a
// venue a test adds later reuses that oracle (fork_venues_test.go addVenue2 creates its market on venue 0's
// oracle and IRM), so it is pinned too.
//
// The reason is upstream, not the port: the BTC/USD feed the c104 market oracle prices from is a Chainlink SVR
// DualAggregator, which withholds each primary round and reveals it once block.timestamp passes it. A fork mines
// its fills at the timestamps the simulator quoted them at, so a reveal can land inside a sequence and move
// `oraclePrice` with no transaction in between -- the state the simulator carries through UpdateBalance then
// differs from a fresh refresh in that one word, which is a property of the upstream feed and not of anything
// under test. The port's own handling of the reveal is Extra.OracleAhead and its refusal (pool_simulator.go
// oracleShifted), which the offline tests cover at the wei.
func (f *anvilFork) pinOracles(t *testing.T, prof *c104Deployment) {
	t.Helper()
	rpcc := &mcRPC{client: f.client}
	venues, _, err := readVenueSet(context.Background(), rpcc, prof.venueScope(), nil)
	require.NoError(t, err)
	for i := range venues {
		v := &venues[i]
		before, err := readOracleAnswer(rpcc, v)
		require.NoError(t, err)
		var out hexutil.Bytes
		err = f.rc.CallContext(context.Background(), &out, "eth_call",
			map[string]any{"to": v.Oracle, "data": "0xa035b1fe"}, "latest")
		if err != nil || len(out) != 32 {
			t.Logf("venue %d oracle %s does not answer price(): %v", i, v.Oracle, err)
			continue
		}
		price := new(big.Int).SetBytes(out)
		f.rpc(nil, "anvil_setCode", v.Oracle, hexutil.Encode(constantPriceCode(price)))
		after, err := readOracleAnswer(rpcc, v)
		require.NoError(t, err)
		require.Equal(t, before, after, "pinning venue %d's oracle changed its answer", i)
		t.Logf("venue %d oracle %s pinned at %s", i, v.Oracle, price)
	}
}

// readOracleAnswer is MorphoBlueAccount.oraclePrice(id), the one way the port reads a venue's market oracle.
func readOracleAnswer(rpcc *mcRPC, v *StaticVenue) (OracleAnswer, error) {
	out, err := readOracleAhead(context.Background(), rpcc, []StaticVenue{*v}, nil, 0)
	if err != nil || len(out) != 1 {
		return OracleAnswer{}, err
	}
	return out[0], nil
}

func adapterData(pool common.Address, venue uint8) []byte {
	data := make([]byte, 64)
	copy(data[12:32], pool[:])
	data[63] = venue
	return data
}

type forkFill struct {
	unused, out *big.Int
	gas         uint64
	revert      []byte
	reverted    bool
}

// callFill eth_calls executeEverlongFlamm at the latest block with the adapter holding exactly amountIn.
func (f *anvilFork) callFill(pool common.Address, venue uint8, tokenIn, tokenOut common.Address, amountIn *big.Int,
	code []byte) forkFill {
	f.t.Helper()
	calldata, err := forkABI.Pack("executeEverlongFlamm", adapterData(pool, venue), amountIn, tokenIn, tokenOut,
		forkRecipient)
	require.NoError(f.t, err)
	overrides := map[common.Address]map[string]any{
		forkAdapter: {"code": hexutil.Encode(code)},
		tokenIn:     {"stateDiff": map[common.Hash]common.Hash{balanceSlot(forkAdapter): common.BigToHash(amountIn)}},
	}
	var out hexutil.Bytes
	err = f.rc.CallContext(context.Background(), &out, "eth_call", map[string]any{"from": forkDeployer,
		"to": forkAdapter, "data": hexutil.Encode(calldata), "gas": hexutil.Uint64(30_000_000)}, "latest", overrides)
	if err != nil {
		data, ok := revertData(err)
		require.True(f.t, ok, "adapter call: %v", err)
		return forkFill{reverted: true, revert: data}
	}
	vals, err := forkABI.Methods["executeEverlongFlamm"].Outputs.Unpack(out)
	require.NoError(f.t, err)
	return forkFill{unused: vals[0].(*big.Int), out: vals[1].(*big.Int)}
}

// execFill funds the adapter with amountIn and mines executeEverlongFlamm at ts: the executor's view of the fill
// is the recipient's output delta and the input left in the adapter.
func (f *anvilFork) execFill(pool common.Address, venue uint8, tokenIn, tokenOut common.Address, amountIn *big.Int,
	ts uint64) forkFill {
	f.t.Helper()
	f.setBalance(tokenIn, forkAdapter, amountIn)
	before := f.balance(tokenOut, forkRecipient)
	calldata, err := forkABI.Pack("executeEverlongFlamm", adapterData(pool, venue), amountIn, tokenIn, tokenOut,
		forkRecipient)
	require.NoError(f.t, err)
	rcpt := f.send(forkDeployer, forkAdapter, calldata, ts)
	require.Equal(f.t, types.ReceiptStatusSuccessful, rcpt.Status, "fill reverted at ts %d", ts)
	out := new(big.Int).Sub(f.balance(tokenOut, forkRecipient), before)
	unused := f.balance(tokenIn, forkAdapter)
	f.setBalance(tokenIn, forkAdapter, new(big.Int)) // the executor sweeps the unused input
	return forkFill{unused: unused, out: out, gas: rcpt.GasUsed}
}

// send mines one transaction at ts from an unlocked (or impersonated) account.
func (f *anvilFork) send(from, to common.Address, data []byte, ts uint64) *types.Receipt {
	f.t.Helper()
	f.rpc(nil, "evm_setNextBlockTimestamp", hexutil.Uint64(ts))
	var hash common.Hash
	f.rpc(&hash, "eth_sendTransaction", map[string]any{"from": from, "to": to, "data": hexutil.Encode(data),
		"gas": hexutil.Uint64(8_000_000)})
	// Anvil mines on receipt of the transaction, but a fill that touches cold state waits on the
	// upstream archive node, which rate-limits: five minutes covers a throttled fetch, where the
	// 30s this used to allow did not.
	var lastErr error
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		rcpt, err := f.geth.TransactionReceipt(context.Background(), hash)
		if err == nil {
			h, err := f.geth.HeaderByHash(context.Background(), rcpt.BlockHash)
			require.NoError(f.t, err)
			require.Equal(f.t, ts, h.Time, "mined at the quoted timestamp")
			return rcpt
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	f.t.Fatalf("tx %s never mined within 5m: %v", hash, lastErr)
	return nil
}

// impersonate unlocks an account and gives it gas money.
func (f *anvilFork) impersonate(who common.Address) {
	f.t.Helper()
	f.rpc(nil, "anvil_impersonateAccount", who)
	f.rpc(nil, "anvil_setBalance", who, "0x56BC75E2D63100000")
}

func (f *anvilFork) view(target common.Address, method string, args ...any) []any {
	f.t.Helper()
	data, err := forkABI.Pack(method, args...)
	require.NoError(f.t, err)
	var out hexutil.Bytes
	f.rpc(&out, "eth_call", map[string]any{"to": target, "data": hexutil.Encode(data)}, "latest")
	vals, err := forkABI.Methods[method].Outputs.Unpack(out)
	require.NoError(f.t, err)
	return vals
}

// forkRevertError maps adapter revert data, which is the pool's own (it bubbles), and Morpho's require strings.
func forkRevertError(data []byte) error {
	if len(data) >= 68 && binary.BigEndian.Uint32(data[:4]) == 0x08c379a0 {
		n := new(big.Int).SetBytes(data[36:68]).Uint64()
		if 68+n <= uint64(len(data)) {
			if e, ok := coreEdgeMorphoStrings[string(data[68:68+n])]; ok {
				return e
			}
		}
		return nil
	}
	return revertError(data)
}
