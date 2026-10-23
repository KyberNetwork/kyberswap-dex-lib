package thogprop

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type contractCase struct {
	Method             string `json:"method"`
	InputIndex         int    `json:"inputIndex"`
	OutputIndex        int    `json:"outputIndex"`
	Amount             string `json:"amount"`
	GasPrice           string `json:"gasPrice"`
	AuctionHandlerWarm bool   `json:"auctionHandlerWarm"`
	Success            bool   `json:"success"`
	Result             string `json:"result"`
}
type contractSnapshot struct {
	Snapshot string         `json:"snapshot"`
	Cases    []contractCase `json:"cases"`
}
type contractFixture struct {
	Proxy     string             `json:"proxy"`
	Snapshots []contractSnapshot `json:"snapshots"`
}

func fixture(t testing.TB, file string) contractFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/" + file)
	require.NoError(t, err)
	z, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer z.Close()
	data, err = io.ReadAll(z)
	require.NoError(t, err)
	var f contractFixture
	require.NoError(t, json.Unmarshal(data, &f))
	return f
}
func raw(t testing.TB, s string) []byte {
	t.Helper()
	data, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	require.NoError(t, err)
	return data
}
func stateFrom(t testing.TB, encoded string) State {
	t.Helper()
	p, err := decodeSnapshot(raw(t, encoded))
	require.NoError(t, err)
	s, err := snapshotState(p)
	require.NoError(t, err)
	return s
}
func number(t testing.TB, s string) uint256.Int {
	t.Helper()
	s = strings.TrimLeft(strings.TrimPrefix(s, "0x"), "0")
	if s == "" {
		s = "0"
	}
	n, err := uint256.FromHex("0x" + s)
	require.NoError(t, err)
	return *n
}
func encodedWords(values ...uint256.Int) []byte {
	var result []byte
	for _, v := range values {
		b := v.Bytes32()
		result = append(result, b[:]...)
	}
	return result
}
func TestDeployedContractParity(t *testing.T) {
	f := fixture(t, "deployed-v7.json.gz")
	tested, success := 0, 0
	for _, snapshot := range f.Snapshots {
		s := stateFrom(t, snapshot.Snapshot)
		for index, c := range snapshot.Cases {
			amount := number(t, c.Amount)
			var got []byte
			var err error
			switch c.Method {
			case "quote":
				var q Quote
				q, err = s.QuoteExactInput(c.InputIndex, c.OutputIndex, amount)
				got = encodedWords(q.AmountOut, u(q.PostedBlock))
			case "price":
				var p Fraction
				p, err = s.MarginalPrice(c.InputIndex, c.OutputIndex, amount)
				got = encodedWords(u(32), u(1), p.Numerator, p.Denominator)
			case "limits":
				var a, b uint256.Int
				a, b, err = s.Limits(c.InputIndex, c.OutputIndex)
				got = encodedWords(u(32), u(2), a, b)
			default:
				t.Fatal(c.Method)
			}
			require.Equal(t, c.Success, err == nil, "block %d case %d (%+v): %v", s.BlockNumber, index, c, err)
			if c.Success {
				require.Equal(t, raw(t, c.Result), got, "block %d case %d", s.BlockNumber, index)
				success++
			}
			tested++
		}
	}
	require.Equal(t, 2016, tested)
	require.Equal(t, 1308, success)
}
func TestDeployedExecutionParity(t *testing.T) {
	f := fixture(t, "execution-v7.json.gz")
	tested, success := 0, 0
	for _, snapshot := range f.Snapshots {
		s := stateFrom(t, snapshot.Snapshot)
		for index, c := range snapshot.Cases {
			amount := number(t, c.Amount)
			ctx := ExecutionContext{BlockNumber: s.BlockNumber, BaseFee: s.BaseFee, GasPrice: number(t, c.GasPrice), AuctionHandlerWarm: c.AuctionHandlerWarm}
			var q Quote
			var err error
			var got uint256.Int
			if c.Method == "execution" {
				q, err = s.QuoteExecution(c.InputIndex, c.OutputIndex, amount, ctx)
				got = q.AmountOut
			} else {
				q, err = s.QuoteExactOutput(c.InputIndex, c.OutputIndex, amount, ctx)
				got = q.AmountIn
			}
			require.Equal(t, c.Success, err == nil, "case %d (%+v): %v", index, c, err)
			if c.Success {
				require.Equal(t, raw(t, c.Result)[:32], encodedWords(got), "case %d (%+v)", index, c)
				success++
			}
			tested++
		}
	}
	require.Equal(t, 1680, tested)
	require.Equal(t, 1261, success)
}
func entityFrom(t testing.TB, s State) entity.Pool {
	t.Helper()
	extra, err := json.Marshal(&Extra{State: s})
	require.NoError(t, err)
	p := entity.Pool{Address: "0x80c74517bcc2d67ffe02d3ed886796272f647210", Type: DexType, Extra: string(extra), StaticExtra: `{"poolId":"0xce389e78282dedac7b18ba7f775b7602d2ab3ab171bbd6711eb0239be6ef4dcc"}`, BlockNumber: s.BlockNumber}
	for i, token := range s.Tokens {
		p.Tokens = append(p.Tokens, &entity.PoolToken{Address: token.Address, Decimals: token.Decimals, Swappable: true})
		p.Reserves = append(p.Reserves, s.Balances[i].Dec())
	}
	return p
}
func TestSimulatorParityPurityAndUpdates(t *testing.T) {
	f := fixture(t, "deployed-v7.json.gz")
	s := stateFrom(t, f.Snapshots[0].Snapshot)
	sim, err := NewPoolSimulator(entityFrom(t, s))
	require.NoError(t, err)
	for index, c := range f.Snapshots[0].Cases {
		if c.Method != "quote" {
			continue
		}
		amount := number(t, c.Amount)
		before := sim.state
		result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Tokens[c.InputIndex].Address, Amount: amount.ToBig()}, TokenOut: s.Tokens[c.OutputIndex].Address})
		require.Equal(t, c.Success, err == nil, "case %d: %v", index, err)
		require.Equal(t, before, sim.state)
		if !c.Success {
			continue
		}
		require.Equal(t, raw(t, c.Result)[:32], encodedWords(*uint256.MustFromBig(result.TokenAmountOut.Amount)))
		clone := sim.CloneState().(*PoolSimulator)
		clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})
		require.Equal(t, before, sim.state)
		swap := result.SwapInfo.(SwapInfo)
		require.Equal(t, swap.InputBalance, clone.state.Balances[c.InputIndex])
		require.Equal(t, swap.OutputBalance, clone.state.Balances[c.OutputIndex])
		// Later quotes derive risk and wrapper inventory from the updated balances.
		rebuilt, err := NewPoolSimulator(entityFrom(t, clone.state))
		require.NoError(t, err)
		q1, e1 := clone.state.QuoteExactInput(c.InputIndex, c.OutputIndex, amount)
		q2, e2 := rebuilt.state.QuoteExactInput(c.InputIndex, c.OutputIndex, amount)
		require.Equal(t, e1, e2)
		require.Equal(t, q1, q2)
	}
}
func BenchmarkQuoteExactInput(b *testing.B) {
	f := fixture(b, "deployed-v7.json.gz")
	s := stateFrom(b, f.Snapshots[0].Snapshot)
	amount := u(100000000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.QuoteExactInput(0, 1, amount)
		if err != nil {
			b.Fatal(err)
		}
	}
}
func TestDeployedBoundaryParity(t *testing.T) {
	f := fixture(t, "boundaries-v7.json.gz")
	tested, success := 0, 0
	for _, snapshot := range f.Snapshots {
		s := stateFrom(t, snapshot.Snapshot)
		for index, c := range snapshot.Cases {
			amount := number(t, c.Amount)
			var got []byte
			var err error
			switch c.Method {
			case "quote":
				var q Quote
				q, err = s.QuoteExactInput(c.InputIndex, c.OutputIndex, amount)
				got = encodedWords(q.AmountOut, u(q.PostedBlock))
			case "price":
				var p Fraction
				p, err = s.MarginalPrice(c.InputIndex, c.OutputIndex, amount)
				got = encodedWords(u(32), u(1), p.Numerator, p.Denominator)
			case "limits":
				var a, b uint256.Int
				a, b, err = s.Limits(c.InputIndex, c.OutputIndex)
				got = encodedWords(u(32), u(2), a, b)
			default:
				t.Fatal(c.Method)
			}
			require.Equal(t, c.Success, err == nil, "block %d case %d (%+v): %v", s.BlockNumber, index, c, err)
			if c.Success {
				require.Equal(t, raw(t, c.Result), got, "block %d case %d", s.BlockNumber, index)
				success++
			}
			tested++
		}
	}
	require.Equal(t, 5544, tested)
	require.Greater(t, success, 2000)
}
