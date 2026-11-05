package thogprop

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"os"
	"path/filepath"
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
	MeasuredGas        int64  `json:"measuredGas"`
	QuoteBlock         uint64 `json:"quoteBlock,omitempty"`
}
type contractSnapshot struct {
	Snapshot         string                  `json:"snapshot"`
	ShareBalances    map[string]ShareBalance `json:"shareBalances"`
	BlockHash        string                  `json:"blockHash"`
	Cases            []contractCase          `json:"cases"`
	historicalModel2 bool
}
type contractFixture struct {
	Proxy     string             `json:"proxy"`
	Snapshots []contractSnapshot `json:"snapshots"`
}

func fixture(t testing.TB, file string) contractFixture {
	t.Helper()
	dir := "testdata"
	// The published full corpora can be checked without adding grid dumps to
	// this dependency. Normal tests use the committed representative excerpts.
	if external := os.Getenv("THOGAMM_FIXTURES"); external != "" {
		dir = external
	}
	data, err := os.ReadFile(filepath.Join(dir, file))
	require.NoError(t, err)
	z, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer func() { require.NoError(t, z.Close()) }()
	data, err = io.ReadAll(z)
	require.NoError(t, err)
	var f contractFixture
	require.NoError(t, json.Unmarshal(data, &f))
	require.NotEmpty(t, f.Snapshots)
	for i := range f.Snapshots {
		f.Snapshots[i].historicalModel2 = file == "deployed-v8.json.gz" || file == "boundaries-v8.json.gz"
	}
	return f
}
func raw(t testing.TB, s string) []byte {
	t.Helper()
	data, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	require.NoError(t, err)
	return data
}
func fixturePage(t testing.TB, snapshot contractSnapshot) poolData {
	t.Helper()
	p, err := decodeSnapshot(raw(t, snapshot.Snapshot))
	require.NoError(t, err)
	if snapshot.historicalModel2 {
		// Model 3 leaves the read-only curve unchanged. Migrate only the model
		// identity in these historical model-2 fixtures, never in runtime reads.
		require.Equal(t, uint16(8), p.SchemaVersion)
		require.Equal(t, uint16(8), p.Words[6].Slot)
		old := new(big.Int).Rsh(new(big.Int).Set(p.Words[6].Value), 176).Uint64() & 255
		require.Equal(t, uint64(2), old)
		p.Words[6].Value = new(big.Int).Add(p.Words[6].Value, new(big.Int).Lsh(big.NewInt(1), 176))
	}
	return p
}
func stateFrom(t testing.TB, snapshot contractSnapshot) State {
	t.Helper()
	s, err := snapshotState(fixturePage(t, snapshot), snapshot.ShareBalances)
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
func checkReadOnly(t *testing.T, file string) {
	t.Helper()
	f := fixture(t, file)
	tested, success := 0, 0
	for _, snapshot := range f.Snapshots {
		s := stateFrom(t, snapshot)
		for index, c := range snapshot.Cases {
			amount := number(t, c.Amount)
			var got []byte
			var err error
			switch c.Method {
			case "quote":
				var q Quote
				block := s.BlockNumber
				if c.QuoteBlock != 0 {
					block = c.QuoteBlock
				}
				q, err = s.QuoteExactInputAt(c.InputIndex, c.OutputIndex, amount, block)
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
	require.Positive(t, tested)
	require.Positive(t, success)
	t.Logf("%s: %d cases, %d exact successes, %d matching rejections", file, tested, success, tested-success)
}
func TestHistoricalReadOnlyContractParity(t *testing.T) { checkReadOnly(t, "deployed-v8.json.gz") }
func TestHistoricalBoundaryParity(t *testing.T)         { checkReadOnly(t, "boundaries-v8.json.gz") }

func TestModel3ExecutionParity(t *testing.T) {
	f := fixture(t, "execution-model3.json.gz")
	tested, success := 0, 0
	for _, snapshot := range f.Snapshots {
		s := stateFrom(t, snapshot)
		for index, c := range snapshot.Cases {
			amount := number(t, c.Amount)
			ctx := ExecutionContext{BlockNumber: s.BlockNumber, BaseFee: s.BaseFee, GasPrice: number(t, c.GasPrice), AuctionHandlerWarm: c.AuctionHandlerWarm}
			var q Quote
			var err error
			var got uint256.Int
			switch c.Method {
			case "execution":
				q, err = s.QuoteExecution(c.InputIndex, c.OutputIndex, amount, ctx)
				got = q.AmountOut
			case "exactOutput":
				q, err = s.QuoteExactOutput(c.InputIndex, c.OutputIndex, amount, ctx)
				got = q.AmountIn
			default:
				t.Fatal(c.Method)
			}
			require.Equal(t, c.Success, err == nil, "block %d case %d (%+v): %v", s.BlockNumber, index, c, err)
			if c.Success {
				require.Equal(t, raw(t, c.Result)[:32], encodedWords(got), "case %d (%+v)", index, c)
				success++
			}
			tested++
		}
	}
	require.Positive(t, tested)
	require.Positive(t, success)
	t.Logf("model 3: %d cases, %d exact successes, %d matching rejections", tested, success, tested-success)
}
func entityFrom(t testing.TB, s State) entity.Pool {
	t.Helper()
	extra, err := json.Marshal(&Extra{State: s, CapturedAtMs: 1})
	require.NoError(t, err)
	p := entity.Pool{Address: "0x80c74517bcc2d67ffe02d3ed886796272f647210", Type: DexType, Extra: string(extra), StaticExtra: `{"poolId":"0xce389e78282dedac7b18ba7f775b7602d2ab3ab171bbd6711eb0239be6ef4dcc"}`, BlockNumber: s.BlockNumber}
	for i, token := range s.Tokens {
		p.Tokens = append(p.Tokens, &entity.PoolToken{Address: token.Address, Decimals: token.Decimals, Swappable: true})
		p.Reserves = append(p.Reserves, s.Balances[i].Dec())
	}
	return p
}
func TestSimulatorOneBlockQuotesPurityAndUpdates(t *testing.T) {
	f := fixture(t, "execution-model3.json.gz")
	checked := 0
	for _, snapshot := range f.Snapshots {
		s := stateFrom(t, snapshot)
		sim, err := NewPoolSimulator(entityFrom(t, s))
		require.NoError(t, err)
		for _, c := range snapshot.Cases {
			if c.Method != "execution" || !c.Success {
				continue
			}
			amount := number(t, c.Amount)
			before := sim.state
			expected, expectedErr := s.QuoteExactInputAt(c.InputIndex, c.OutputIndex, amount, s.BlockNumber+1)
			params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Tokens[c.InputIndex].Address, Amount: amount.ToBig()}, TokenOut: s.Tokens[c.OutputIndex].Address}
			result, err := sim.quoteAtCapture(params)
			if sim.state.BlockNumber+1-sim.state.small(2, 36, 32) > sim.state.small(2, 138, 8) {
				expectedErr = ErrStaleSnapshot
			}
			require.Equal(t, expectedErr, err)
			require.Equal(t, before, sim.state)
			if err != nil {
				continue
			}
			require.Equal(t, expected.AmountOut.ToBig(), result.TokenAmountOut.Amount)
			clone := sim.CloneState().(*PoolSimulator)
			clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})
			require.Equal(t, before, sim.state)
			swap := result.SwapInfo.(SwapInfo)
			require.Equal(t, swap.InputBalance, clone.state.Balances[c.InputIndex])
			require.Equal(t, swap.OutputBalance, clone.state.Balances[c.OutputIndex])
			require.Equal(t, swap.InputUnits, clone.state.packed(107, c.InputIndex, 128, 2))
			require.Equal(t, swap.OutputUnits, clone.state.packed(107, c.OutputIndex, 128, 2))
			rebuilt, err := NewPoolSimulator(entityFrom(t, clone.state))
			require.NoError(t, err)
			q1, e1 := clone.quoteAtCapture(params)
			q2, e2 := rebuilt.quoteAtCapture(params)
			require.Equal(t, e1, e2)
			require.Equal(t, q1, q2)
			checked++
			if checked >= 100 {
				return
			}
		}
	}
	require.Positive(t, checked)
}
func BenchmarkQuoteExactInput(b *testing.B) {
	f := fixture(b, "execution-model3.json.gz")
	s := stateFrom(b, f.Snapshots[0])
	amount := u(1000000)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := s.QuoteExactInputAt(0, 1, amount, s.BlockNumber+1); err != nil {
			b.Fatal(err)
		}
	}
}

func TestLiveModel3ReadOnlyParity(t *testing.T)    { checkReadOnly(t, "deployed-model3.json.gz") }
func TestOneBlockDelayContractParity(t *testing.T) { checkReadOnly(t, "one-block-model3.json.gz") }

func TestRouterQuotesMatchOneBlockEVM(t *testing.T) {
	f := fixture(t, "one-block-model3.json.gz")
	for _, snapshot := range f.Snapshots {
		s := stateFrom(t, snapshot)
		require.Equal(t, uint64(50), s.small(2, 146, 16))
		sim, err := NewPoolSimulator(entityFrom(t, s))
		require.NoError(t, err)
		count := 0
		for _, c := range snapshot.Cases {
			if c.QuoteBlock != s.BlockNumber+1 {
				continue
			}
			amount := number(t, c.Amount)
			before := sim.state
			result, err := sim.quoteAtCapture(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: s.Tokens[c.InputIndex].Address, Amount: amount.ToBig()},
				TokenOut:      s.Tokens[c.OutputIndex].Address,
			})
			require.Equal(t, c.Success, err == nil, "%+v: %v", c, err)
			require.Equal(t, before, sim.state)
			if c.Success {
				require.Equal(t, raw(t, c.Result)[:32], encodedWords(*uint256.MustFromBig(result.TokenAmountOut.Amount)))
				a, err := s.prepare(c.InputIndex, c.OutputIndex, s.BlockNumber)
				require.NoError(t, err)
				b, err := s.prepare(c.InputIndex, c.OutputIndex, s.BlockNumber+1)
				require.NoError(t, err)
				var delta uint256.Int
				delta.Sub(&b.spread, &a.spread)
				require.Equal(t, u(50), delta)
			}
			count++
		}
		require.Positive(t, count)
	}
}

// Contract corpora quote at the exact capture instant; live clock advancement is tested separately.
func (s *PoolSimulator) quoteAtCapture(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	return s.calcAmountOut(params, s.capturedAtMs)
}
