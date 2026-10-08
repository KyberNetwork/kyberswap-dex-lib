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

var executionABI = mustABI(`[{"type":"function","name":"executeFlywheelNative","inputs":[{"type":"bytes"},{"type":"uint256"},{"type":"address"},{"type":"address"},{"type":"address"}],"outputs":[{"type":"uint256"},{"type":"uint256"}]},{"type":"function","name":"transfer","inputs":[{"type":"address"},{"type":"uint256"}],"outputs":[{"type":"bool"}]}]`)

// Opt-in integration test. Transactions are sent ONLY to a locally spawned
// Anvil instance, behind a read-only upstream proxy; never to the chain RPC.
func TestLocalForkQuoteExecutionParity(t *testing.T) {
	url := os.Getenv("FLYWHEEL_LOCAL_TEST_RPC")
	if url == "" {
		t.Skip("run node testdata/runner/run-forks.cjs quotes for local-fork parity")
	}
	require.True(t, strings.HasPrefix(url, "http://127.0.0.1:"))
	ctx := context.Background()
	raw, err := rpc.DialContext(ctx, url)
	require.NoError(t, err)
	defer raw.Close()
	var version string
	require.NoError(t, raw.CallContext(ctx, &version, "web3_clientVersion"))
	require.Contains(t, strings.ToLower(version), "anvil")
	eth := ethclient.NewClient(raw)
	chain, err := eth.ChainID(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(4663), chain.Int64())
	adapter := common.HexToAddress(os.Getenv("FLYWHEEL_LOCAL_ADAPTER"))
	require.NotEqual(t, common.Address{}, adapter)
	var accounts []common.Address
	require.NoError(t, raw.CallContext(ctx, &accounts, "eth_accounts"))
	require.GreaterOrEqual(t, len(accounts), 2)
	sender, recipient := accounts[0], common.HexToAddress("0x10000000000000000000000000000000feedf00d")
	// Public Anvil default addresses can have EIP-7702 code on the forked chain.
	// Use a dedicated empty-code local receiver for balance-delta assertions.
	require.NoError(t, raw.CallContext(ctx, nil, "anvil_setCode", recipient, "0x"))
	require.NoError(t, raw.CallContext(ctx, nil, "anvil_setBalance", recipient, "0x3635c9adc5dea00000"))
	require.NoError(t, raw.CallContext(ctx, nil, "anvil_impersonateAccount", recipient))
	native := common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE")
	client := ethrpc.NewWithClient(eth).SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))
	boomer := "0x73c2de14c7fa0a57cc2d9722b959ea70b881ffe4"
	pons := "0x39dbed3a2bd333467115de45665cc57f813c4571"
	tracker, err := NewPoolTracker(nil, client)
	require.NoError(t, err)
	var captured []entity.Pool
	var lastGas uint64
	send := func(from, to common.Address, data []byte, value *big.Int) common.Hash {
		t.Helper()
		var hash common.Hash
		tx := map[string]any{"from": from, "to": to, "data": hexutil.Bytes(data), "gas": "0x989680"}
		if value != nil {
			tx["value"] = hexutil.EncodeBig(value)
		}
		require.NoError(t, raw.CallContext(ctx, &hash, "eth_sendTransaction", tx))
		receipt, e := eth.TransactionReceipt(ctx, hash)
		for n := 0; e != nil && n < 100; n++ {
			time.Sleep(100 * time.Millisecond)
			receipt, e = eth.TransactionReceipt(ctx, hash)
		}
		require.NoError(t, e)
		require.Equal(t, uint64(1), receipt.Status)
		lastGas = receipt.GasUsed
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
				before := balance(token, recipient)
				beforeRefund := balance(native, adapter)
				data, e := EncodeTradeData(quote.SwapInfo.(SwapInfo), 0, uint64(time.Now().Unix()), uint64(time.Now().Unix()+1800))
				require.NoError(t, e)
				call, e := executionABI.Pack("executeFlywheelNative", data, big.NewInt(amount), native, token, recipient)
				require.NoError(t, e)
				send(sender, adapter, call, big.NewInt(amount))
				buyGas := lastGas
				require.GreaterOrEqual(t, quote.Gas, int64(buyGas), "gas estimate must cover measured fork buy")
				received := new(big.Int).Sub(balance(token, recipient), before)
				require.Equal(t, quote.TokenAmountOut.Amount.String(), received.String(), "buy quote must match actual base units")
				refund := new(big.Int).Sub(balance(native, adapter), beforeRefund)
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
				transfer, e := executionABI.Pack("transfer", adapter, sold)
				require.NoError(t, e)
				send(recipient, token, transfer, nil)
				data, e = EncodeTradeData(sell.SwapInfo.(SwapInfo), 0, uint64(time.Now().Unix()), uint64(time.Now().Unix()+1800))
				require.NoError(t, e)
				call, e = executionABI.Pack("executeFlywheelNative", data, sold, token, native, recipient)
				require.NoError(t, e)
				before = balance(native, recipient)
				send(sender, adapter, call, nil)
				require.GreaterOrEqual(t, sell.Gas, int64(lastGas), "gas estimate must cover measured fork sell")
				after := balance(native, recipient)
				received.Sub(after, before)
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

func TestCurveRefundAndCalldata(t *testing.T) {
	s, e := NewPoolSimulator(testEntity())
	require.NoError(t, e)
	q, e := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: big.NewInt(2000000)}, TokenOut: s.Info.Address})
	require.NoError(t, e)
	info := q.SwapInfo.(SwapInfo)
	require.True(t, info.Next.Graduated)
	require.Positive(t, info.Refund.Sign())
	require.True(t, info.Refund.Cmp(&info.RefundRouteOutput) >= 0)
	data, e := EncodeTradeData(info, 100, 100, 400)
	require.NoError(t, e)
	unpacked, e := tradeArguments.Unpack(data)
	require.NoError(t, e)
	require.Len(t, unpacked, 1)
	s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: info})
	require.False(t, s.Valid, "requires refreshed pool state after migration")
	_, e = EncodeTradeData(info, 10000, 100, 400)
	require.Error(t, e)
	_, e = EncodeTradeData(info, 0, 400, 100)
	require.Error(t, e)
	info.RefundRouteOutput.Clear()
	_, e = EncodeTradeData(info, 0, 100, 400)
	require.Error(t, e)
	amount := uint256.NewInt(1000)
	out, next, used, e := CurveFill(testCurve(), amount, true)
	require.NoError(t, e)
	require.Equal(t, uint64(1000), used.Uint64())
	require.False(t, next.Graduated)
	require.False(t, out.IsZero())
}

var tradeArguments = abi.Arguments{{Type: abiType("tuple", []abi.ArgumentMarshaling{{Name: "token", Type: "address"}, {Name: "minQuote", Type: "uint256"}, {Name: "minOutput", Type: "uint256"}, {Name: "deadline", Type: "uint256"}, {Name: "route", Type: "bytes"}, {Name: "minRefundETH", Type: "uint256"}, {Name: "refundRoute", Type: "bytes"}})}}

type AdapterTrade struct {
	Token        common.Address
	MinQuote     *big.Int
	MinOutput    *big.Int
	Deadline     *big.Int
	Route        []byte
	MinRefundETH *big.Int
	RefundRoute  []byte
}

// EncodeTradeData returns abi.encode(Trade) for the local fork harness's execution module.
// Refund protection uses the reverse swap's ETH output, not the larger refund that also
// includes returned platform fees.
func EncodeTradeData(info SwapInfo, slippageBps uint16, now, deadline uint64) ([]byte, error) {
	if slippageBps >= 10000 || deadline <= now || deadline-now > 3600 || !common.IsHexAddress(info.Token) || common.HexToAddress(info.Token) == (common.Address{}) || info.MinQuote.IsZero() || info.AmountOut.IsZero() {
		return nil, ErrAmount
	}
	minimum := func(x *uint256.Int) *big.Int {
		v := new(big.Int).Mul(x.ToBig(), big.NewInt(int64(10000-slippageBps)))
		v.Div(v, big.NewInt(10000))
		if v.Sign() == 0 {
			v.SetInt64(1)
		}
		return v
	}
	t := AdapterTrade{Token: common.HexToAddress(info.Token), MinQuote: minimum(&info.MinQuote), MinOutput: minimum(&info.AmountOut), Deadline: new(big.Int).SetUint64(deadline), Route: info.Route, MinRefundETH: new(big.Int)}
	if !info.Refund.IsZero() {
		if !info.Buy || info.RefundRouteOutput.IsZero() {
			return nil, ErrAmount
		}
		t.MinRefundETH = minimum(&info.RefundRouteOutput)
		t.RefundRoute = info.Route
	}
	return tradeArguments.Pack(t)
}
