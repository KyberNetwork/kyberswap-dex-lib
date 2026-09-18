package prmfun

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type quoteFixture struct {
	Buy    bool   `json:"buy"`
	Input  string `json:"input"`
	Output string `json:"output"`
	Used   string `json:"used"`
	Fee    string `json:"fee"`
}
type marketFixture struct {
	Pair   string         `json:"pair"`
	Pool   entity.Pool    `json:"pool"`
	Quotes []quoteFixture `json:"quotes"`
	After  *Extra         `json:"after,omitempty"`
}

// Explicit opt-in: the deterministic tests do not need RPC access or credentials.
// Each quote is compared at the exact L2 block returned by the tracker.
func TestLiveMainnetQuoteParity(t *testing.T) {
	rpcURL := os.Getenv("PREMIUM_RPC_URL")
	if rpcURL == "" {
		t.Skip("set PREMIUM_RPC_URL for read-only onchain parity")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client := ethrpc.New(rpcURL).SetMulticallContract(multicall3)
	chain, err := client.GetETHClient().ChainID(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(4663), chain.Int64())
	cfg := &Config{DexId: DexType, ChainId: 4663, FactoryAddress: factoryAddress, RouterAddress: routerAddress, NewPoolLimit: 100}
	lister := NewPoolsListUpdater(cfg, client)
	var metadata []byte
	var candidates []entity.Pool
	for {
		batch, next, err := lister.GetNewPools(ctx, metadata)
		require.NoError(t, err)
		candidates = append(candidates, batch...)
		if string(next) == string(metadata) {
			break
		}
		metadata = next
	}
	tracker, err := NewPoolTracker(cfg, client)
	require.NoError(t, err)
	seen := map[string]bool{}
	fixtures := []marketFixture{}
	for _, candidate := range candidates {
		pair := candidate.Tokens[0].Address
		if seen[pair] {
			continue
		}
		p, err := tracker.GetNewPoolState(ctx, candidate, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		s, err := NewPoolSimulator(p)
		require.NoError(t, err)
		if s.phase != PhaseTrading {
			continue
		}
		amounts := []*big.Int{new(big.Int).Div(s.graduationDesk.ToBig(), big.NewInt(1000)), new(big.Int).Mul(s.graduationDesk.ToBig(), big.NewInt(2))}
		f := marketFixture{Pair: pair, Pool: p}
		for _, amount := range amounts {
			var onchain struct{ MemeOut, DeskUsed, Fee *big.Int }
			_, err = client.NewRequest().SetContext(ctx).SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber)).AddCall(&ethrpc.Call{ABI: memeCurveABI, Target: p.Address, Method: "quoteBuy", Params: []any{amount}}, []any{&onchain}).Call()
			require.NoError(t, err)
			f.Quotes = append(f.Quotes, quoteFixture{Buy: true, Input: amount.String(), Output: onchain.MemeOut.String(), Used: onchain.DeskUsed.String(), Fee: onchain.Fee.String()})
		}
		if !s.memeSold.IsZero() {
			amount := new(big.Int).Div(s.memeSold.ToBig(), big.NewInt(1000))
			if amount.Sign() == 0 {
				amount.SetInt64(1)
			}
			var onchain struct{ DeskOut, Fee *big.Int }
			_, err = client.NewRequest().SetContext(ctx).SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber)).AddCall(&ethrpc.Call{ABI: memeCurveABI, Target: p.Address, Method: "quoteSell", Params: []any{amount}}, []any{&onchain}).Call()
			require.NoError(t, err)
			f.Quotes = append(f.Quotes, quoteFixture{Input: amount.String(), Output: onchain.DeskOut.String(), Used: amount.String(), Fee: onchain.Fee.String()})
		}
		checkFixture(t, f)
		fixtures = append(fixtures, f)
		seen[pair] = true
		t.Logf("pair=%s curve=%s L2 block=%d compared=%d", pair, p.Address, p.BlockNumber, len(f.Quotes))
	}
	require.NotEmpty(t, fixtures)
	if output := os.Getenv("PREMIUM_FIXTURE_OUTPUT"); output != "" {
		encoded, err := json.MarshalIndent(fixtures, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(filepath.Dir(output), 0755))
		require.NoError(t, os.WriteFile(output, append(encoded, '\n'), 0644))
	}
}

func checkFixture(t *testing.T, f marketFixture) {
	t.Helper()
	s, err := NewPoolSimulator(f.Pool)
	require.NoError(t, err)
	for _, q := range f.Quotes {
		amount, ok := new(big.Int).SetString(q.Input, 10)
		require.True(t, ok)
		input, output := f.Pool.Tokens[1].Address, f.Pool.Tokens[0].Address
		if q.Buy {
			input, output = output, input
		}
		got, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: input, Amount: amount}, TokenOut: output})
		if q.Output == "0" {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, q.Output, got.TokenAmountOut.Amount.String())
		require.Equal(t, q.Fee, got.Fee.Amount.String())
		used := new(big.Int).Set(amount)
		if got.RemainingTokenAmountIn != nil {
			used.Sub(used, got.RemainingTokenAmountIn.Amount)
		}
		require.Equal(t, q.Used, used.String())
		if f.After != nil {
			require.Len(t, f.Quotes, 1)
			s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: got.SwapInfo})
			require.Equal(t, *f.After, s.stateCopy())
		}
	}
}

func TestPinnedMainnetQuoteFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/mainnet-quotes.json")
	require.NoError(t, err)
	var fixtures []marketFixture
	require.NoError(t, json.Unmarshal(data, &fixtures))
	require.NotEmpty(t, fixtures)
	for _, f := range fixtures {
		t.Run(f.Pair, func(t *testing.T) { checkFixture(t, f) })
	}
}

func TestPinnedForkSettlementFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/fork-quotes.json")
	require.NoError(t, err)
	var fixtures []marketFixture
	require.NoError(t, json.Unmarshal(data, &fixtures))
	require.Len(t, fixtures, 9)
	for _, f := range fixtures {
		t.Run(f.Pair, func(t *testing.T) { require.NotNil(t, f.After); checkFixture(t, f) })
	}
}

// Compare gas allowances with real local-fork receipts, not another implementation estimate.
func TestGasAllowanceCoversForkReceipts(t *testing.T) {
	data, err := os.ReadFile("testdata/fork-settlement.json")
	require.NoError(t, err)
	var evidence struct {
		Settlements []struct {
			Pair    string
			Action  string
			GasUsed string
		}
	}
	require.NoError(t, json.Unmarshal(data, &evidence))
	require.Len(t, evidence.Settlements, 9)
	for _, receipt := range evidence.Settlements {
		used, ok := new(big.Int).SetString(receipt.GasUsed, 10)
		require.True(t, ok)
		allowance := int64(buyGas)
		if receipt.Action == "sell" {
			allowance = sellGas
		}
		if receipt.Action == "final-buy" {
			allowance += graduationGas
		}
		require.LessOrEqual(t, used.Int64(), allowance, "%s %s", receipt.Pair, receipt.Action)
	}
}
