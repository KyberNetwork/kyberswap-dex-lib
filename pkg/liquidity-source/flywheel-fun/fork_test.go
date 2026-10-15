package flywheelfun

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The settlement entry points aggregator-encoding calls through executeGeneric.
var settlementABI = mustABI(`[{"type":"function","name":"buy","inputs":[{"type":"address"},{"type":"uint256"},{"type":"uint256"},{"type":"uint256"},{"type":"bytes"}],"outputs":[{"type":"uint256"}]},{"type":"function","name":"buyWithRefund","inputs":[{"type":"address"},{"type":"uint256"},{"type":"uint256"},{"type":"uint256"},{"type":"bytes"},{"type":"uint256"},{"type":"bytes"}],"outputs":[{"type":"uint256"}]},{"type":"function","name":"sell","inputs":[{"type":"address"},{"type":"uint256"},{"type":"uint256"},{"type":"uint256"},{"type":"uint256"},{"type":"bytes"}],"outputs":[{"type":"uint256"}]}]`)
var erc20ABI = mustABI(`[{"type":"function","name":"approve","inputs":[{"type":"address"},{"type":"uint256"}],"outputs":[{"type":"bool"}]}]`)

// Opt-in integration test. Transactions are sent ONLY to a locally spawned Anvil instance
// or a Tenderly virtual testnet; never to the chain RPC.
func TestLocalForkQuoteExecutionParity(t *testing.T) {
	url := os.Getenv("FLYWHEEL_LOCAL_TEST_RPC")
	if url == "" {
		t.Skip("run node testdata/runner/run-forks.cjs for local-fork parity")
	}
	require.True(t, strings.HasPrefix(url, "http://127.0.0.1:") || strings.HasPrefix(url, "https://virtual."), "local Anvil or a Tenderly virtual testnet only")
	ctx := context.Background()
	raw, err := rpc.DialContext(ctx, url)
	require.NoError(t, err)
	defer raw.Close()
	var version string
	require.NoError(t, raw.CallContext(ctx, &version, "web3_clientVersion"))
	// Anvil and Tenderly virtual testnets (archive-backed forks) expose the same cheats under different names.
	cheat := "tenderly_"
	if strings.Contains(strings.ToLower(version), "anvil") {
		cheat = "anvil_"
	}
	eth := ethclient.NewClient(raw)
	chain, err := eth.ChainID(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(4663), chain.Int64())
	settlement := common.HexToAddress(Settlement)
	// A dedicated empty-code account plays the executor: it calls the settlement directly, like
	// executeGeneric, and receives the tokens, ETH proceeds and graduation refund. Public Anvil
	// default addresses can have EIP-7702 code on the forked chain.
	exec := common.HexToAddress("0x10000000000000000000000000000000feedf00d")
	require.NoError(t, raw.CallContext(ctx, nil, cheat+"setCode", exec, "0x"))
	var who any = exec
	if cheat == "tenderly_" {
		who = []common.Address{exec}
	}
	require.NoError(t, raw.CallContext(ctx, nil, cheat+"setBalance", who, "0x3635c9adc5dea00000"))
	if cheat == "anvil_" {
		require.NoError(t, raw.CallContext(ctx, nil, "anvil_impersonateAccount", exec))
	}
	native := common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE")
	client := ethrpc.NewWithClient(eth).SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))
	boomer := "0x73c2de14c7fa0a57cc2d9722b959ea70b881ffe4"
	pons := "0x39dbed3a2bd333467115de45665cc57f813c4571"
	tracker, err := NewPoolTracker(nil, client)
	require.NoError(t, err)
	var captured []entity.Pool
	var lastGas uint64
	lastCost := new(big.Int) // gas paid by the last tx, added back to ETH balance deltas
	send := func(from, to common.Address, data []byte, value *big.Int) common.Hash {
		t.Helper()
		var hash common.Hash
		tx := map[string]any{"from": from, "to": to, "data": hexutil.Bytes(data), "gas": "0x989680"}
		if value != nil {
			tx["value"] = hexutil.EncodeBig(value)
		}
		require.NoError(t, raw.CallContext(ctx, &hash, "eth_sendTransaction", tx))
		receipt, e := eth.TransactionReceipt(ctx, hash)
		for n := 0; e != nil && n < 600; n++ {
			time.Sleep(100 * time.Millisecond)
			receipt, e = eth.TransactionReceipt(ctx, hash)
		}
		require.NoError(t, e)
		require.Equal(t, uint64(1), receipt.Status)
		lastGas = receipt.GasUsed
		lastCost.Mul(new(big.Int).SetUint64(receipt.GasUsed), receipt.EffectiveGasPrice)
		return hash
	}
	balance := func(token, who common.Address) *big.Int {
		t.Helper()
		if token == native {
			v, e := eth.BalanceAt(ctx, who, nil)
			require.NoError(t, e)
			return v
		}
		var v *big.Int
		_, e := client.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{ABI: tokenABI, Target: token.Hex(), Method: "balanceOf", Params: []any{who}}, []any{&v}).Aggregate()
		require.NoError(t, e)
		return v
	}
	cases := []struct{ name, token, quote string }{{"weth_graduated", "0x3245af253dc425459c331b2ff8b2fe170d781ae5", WETH}, {"boomer_graduated", "0x0d451b146208549b4c6bdc86d8ae7830f91efd84", boomer}}
	cases = append(cases, struct{ name, token, quote string }{"weth_graduation_refund", os.Getenv("FLYWHEEL_LOCAL_WETH_CURVE"), WETH}, struct{ name, token, quote string }{"boomer_graduation_refund", os.Getenv("FLYWHEEL_LOCAL_BOOMER_CURVE"), boomer})
	cases = append(cases, struct{ name, token, quote string }{"pons_v3_curve", os.Getenv("FLYWHEEL_LOCAL_PONS_CURVE"), pons})
	cases = append(cases,
		struct{ name, token, quote string }{"one_parent_curve", os.Getenv("FLYWHEEL_LOCAL_CHILD"), os.Getenv("FLYWHEEL_LOCAL_PARENT")},
		struct{ name, token, quote string }{"one_parent_refund", os.Getenv("FLYWHEEL_LOCAL_CHILD_REFUND"), os.Getenv("FLYWHEEL_LOCAL_PARENT")},
		struct{ name, token, quote string }{"two_parent_curve", os.Getenv("FLYWHEEL_LOCAL_GRANDCHILD"), os.Getenv("FLYWHEEL_LOCAL_NESTED_PARENT")},
		struct{ name, token, quote string }{"two_parent_refund", os.Getenv("FLYWHEEL_LOCAL_GRANDCHILD_REFUND"), os.Getenv("FLYWHEEL_LOCAL_NESTED_PARENT")})
	// A separate local provider supplies indexed V3/V4 bases at the same block.
	fixtureBytes, e := os.ReadFile("testdata/replacement-fork-markets.json")
	require.NoError(t, e)
	var fixtures []entity.Pool
	require.NoError(t, json.Unmarshal(fixtureBytes, &fixtures))
	external := map[string]forkBaseHop{}
	for _, fp := range fixtures {
		var x struct {
			Route []forkBaseHop `json:"route"`
		}
		var st StaticExtra
		require.NoError(t, json.Unmarshal([]byte(fp.Extra), &x))
		require.NoError(t, json.Unmarshal([]byte(fp.StaticExtra), &st))
		if len(x.Route) == 1 {
			external[st.Quote] = x.Route[0]
		}
	}
	hydrate := func(ep entity.Pool) (entity.Pool, map[string]pool.IPoolSimulator) {
		t.Helper()
		bases := map[string]pool.IPoolSimulator{}
		var st StaticExtra
		require.NoError(t, json.Unmarshal([]byte(ep.StaticExtra), &st))
		if h, ok := external[st.Quote]; ok {
			id, e := h.id()
			require.NoError(t, e)
			st.BasePools = []string{id}
			b, e := json.Marshal(st)
			require.NoError(t, e)
			ep.StaticExtra = string(b)
		}
		ep, e = tracker.GetNewPoolState(ctx, ep, pool.GetNewPoolStateParams{})
		require.NoError(t, e)
		if h, ok := external[st.Quote]; ok {
			block := new(big.Int).SetUint64(ep.BlockNumber)
			var bp entity.Pool
			if h.Kind == 3 {
				bp, _, e = tracker.forkTrackV3(ctx, h, block)
			} else {
				bp, _, e = tracker.forkTrackV4(ctx, h, block, false)
			}
			require.NoError(t, e)
			bases[bp.Address] = baseSim(t, bp)
		}
		return ep, bases
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			amounts := []int64{10_000_000_000, 100_000_000_000, 1_000_000_000_000}
			if strings.Contains(tc.name, "refund") {
				amounts = []int64{1_000_000_000, 400_000_000_000}
			}
			for _, amount := range amounts {
				st, _ := json.Marshal(StaticExtra{Factory: Factory, Settlement: Settlement, Quote: tc.quote})
				ep := entity.Pool{Address: tc.token, Type: DexType, Exchange: DexType, StaticExtra: string(st), Tokens: []*entity.PoolToken{{Address: WETH}, {Address: tc.token}}}
				ep, bases := hydrate(ep)
				if amount == amounts[0] {
					captured = append(captured, ep)
				}
				sim, e := NewPoolSimulatorWithBases(ep, bases)
				require.NoError(t, e)
				params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: big.NewInt(amount)}, TokenOut: tc.token}
				quote, e := sim.CalcAmountOut(params)
				require.NoError(t, e)
				token := common.HexToAddress(tc.token)
				info := quote.SwapInfo.(SwapInfo)
				before := balance(token, exec)
				beforeETH := balance(native, exec)
				deadline := big.NewInt(time.Now().Unix() + 1800)
				one := big.NewInt(1)
				// Same entry-point choice and minima as aggregator-encoding's PackFlywheelFun.
				var call []byte
				if info.Refunds {
					call, e = settlementABI.Pack("buyWithRefund", token, one, one, deadline, []byte(info.Route), one, []byte(info.Route))
				} else {
					call, e = settlementABI.Pack("buy", token, one, one, deadline, []byte(info.Route))
				}
				require.NoError(t, e)
				send(exec, settlement, call, big.NewInt(amount))
				buyGas := lastGas
				require.GreaterOrEqual(t, quote.Gas, int64(buyGas), "gas estimate must cover measured fork buy")
				received := new(big.Int).Sub(balance(token, exec), before)
				require.Equal(t, quote.TokenAmountOut.Amount.String(), received.String(), "buy quote must match actual base units")
				refund := new(big.Int).Sub(balance(native, exec), beforeETH)
				refund.Add(refund, big.NewInt(amount)).Add(refund, lastCost)
				require.Equal(t, quote.RemainingTokenAmountIn.Amount.String(), refund.String())
				sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: quote.SwapInfo})
				if quote.SwapInfo.(SwapInfo).Next.Graduated && sim.MarketPool == nil {
					require.False(t, sim.Valid)
					require.Positive(t, refund.Sign())
					ep, bases = hydrate(ep)
					sim, e = NewPoolSimulatorWithBases(ep, bases)
					require.NoError(t, e)
				}
				require.True(t, sim.Valid)
				sold := new(big.Int).Div(received, big.NewInt(2))
				sell, e := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tc.token, Amount: sold}, TokenOut: WETH})
				require.NoError(t, e)
				send(exec, token, mustPack(t, erc20ABI, "approve", settlement, sold), nil)
				call, e = settlementABI.Pack("sell", token, sold, one, one, deadline, []byte(sell.SwapInfo.(SwapInfo).Route))
				require.NoError(t, e)
				before = balance(native, exec)
				send(exec, settlement, call, nil)
				require.GreaterOrEqual(t, sell.Gas, int64(lastGas), "gas estimate must cover measured fork sell")
				after := balance(native, exec)
				received.Sub(after, before).Add(received, lastCost)
				require.Equal(t, sell.TokenAmountOut.Amount.String(), received.String(), "sell after UpdateBalance must match actual base units")
				t.Logf("input %d: buy %s; sell half %s; refund %s; gas %d/%d", amount, quote.TokenAmountOut.Amount, received, refund, buyGas, lastGas)
			}
		})
	}
	if os.Getenv("FLYWHEEL_CAPTURE_FIXTURES") == "1" {
		b, e := json.MarshalIndent(captured, "", "  ")
		require.NoError(t, e)
		require.NoError(t, os.WriteFile("testdata/indexed-fork-markets.json", b, 0600))
	}
}

func mustPack(t *testing.T, a abi.ABI, method string, args ...any) []byte {
	t.Helper()
	data, err := a.Pack(method, args...)
	require.NoError(t, err)
	return data
}

func TestCurveRefund(t *testing.T) {
	s, e := NewPoolSimulator(testEntity())
	require.NoError(t, e)
	q, e := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: big.NewInt(2000000)}, TokenOut: s.Info.Address})
	require.NoError(t, e)
	info := q.SwapInfo.(SwapInfo)
	require.True(t, info.Next.Graduated)
	require.Positive(t, info.Refund.Sign())
	require.True(t, info.Refund.Cmp(&info.RefundRouteOutput) >= 0)
	require.True(t, info.Refunds)
	s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: info})
	require.False(t, s.Valid, "requires refreshed pool state after migration")
	amount := uint256.NewInt(1000)
	out, next, used, e := CurveFill(testCurve(), amount, true)
	require.NoError(t, e)
	require.Equal(t, uint64(1000), used.Uint64())
	require.False(t, next.Graduated)
	require.False(t, out.IsZero())
}
