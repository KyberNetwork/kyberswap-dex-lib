//go:build integration

package spireprop

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestLiveSnapshot(t *testing.T) {
	endpoint := os.Getenv("SPIRE_RPC_URL")
	if endpoint == "" {
		t.Fatal("SPIRE_RPC_URL is required for the explicit integration build")
	}
	_ = logger.SetLogLevel("fatal")
	fail := func(stage string, err error) {
		t.Helper()
		t.Fatalf("%s: %s", stage, strings.ReplaceAll(err.Error(), endpoint, "<rpc>"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	client := ethrpc.New(endpoint).SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))
	cfg := &Config{DexID: DexType, Entrypoint: "0x98c1d9e102eb2806d902b13186bdc7892ac4ffba", Bases: []string{"0x4200000000000000000000000000000000000006"}}
	lister := NewPoolsListUpdater(cfg, client)
	listed, metadata, err := lister.GetNewPools(ctx, nil)
	if err != nil {
		fail("list", err)
	}
	if len(listed) != 1 || listed[0].Reserves[0] != "0" || listed[0].Tokens[0].Decimals != 0 {
		t.Fatal("discovery contract")
	}
	duplicate, _, err := lister.GetNewPools(ctx, metadata)
	if err != nil {
		fail("list again", err)
	}
	if len(duplicate) != 0 {
		t.Fatal("duplicated discovery")
	}
	p, err := NewPoolTracker(client).GetNewPoolState(ctx, listed[0], pool.GetNewPoolStateParams{})
	if err != nil {
		fail("track", err)
	}
	simulator, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	if err != nil {
		fail("construct", err)
	}
	fixture := QuoteFixture{Pool: p}
	req := client.NewRequest().SetContext(ctx).SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber))
	inputs := []string{"1", "2", "100", "10000", "1000000", "25000000", "100000000", "500000000", "1000000000", "10000000000", "1000000000000", "100000000000000", "1000000000000000", "10000000000000000", "100000000000000000", "1000000000000000000", "1000000000000000000000"}
	outputs := make([]*big.Int, 2*len(inputs))
	for side := 0; side < 2; side++ {
		for j, input := range inputs {
			amount, _ := new(big.Int).SetString(input, 10)
			i := side*len(inputs) + j
			req.AddCall(&ethrpc.Call{ABI: curveBookABI, Target: simulator.StaticExtra.CurveBook, Method: "quote", Params: []any{common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[side].Address), amount}}, []any{&outputs[i]})
		}
	}
	if _, err = req.Aggregate(); err != nil {
		fail("reference quotes", err)
	}
	success := 0
	for side := 0; side < 2; side++ {
		for j, input := range inputs {
			expected := outputs[side*len(inputs)+j]
			amount, _ := new(big.Int).SetString(input, 10)
			result, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[side].Address, Amount: amount}, TokenOut: p.Tokens[1-side].Address})
			executable := expected.Sign() > 0 && expected.Cmp(simulator.Info.Reserves[1-side]) <= 0
			if executable {
				if err != nil {
					fail("simulate executable reference", err)
				}
				if result.TokenAmountOut.Amount.Cmp(expected) != 0 {
					t.Fatalf("quote mismatch side=%d input=%s actual=%s expected=%s", side, input, result.TokenAmountOut.Amount, expected)
				}
				success++
			} else if err == nil {
				t.Fatalf("accepted non-executable quote side=%d input=%s", side, input)
			}
			fixture.Quotes = append(fixture.Quotes, QuoteCase{IndexIn: side, AmountIn: input, AmountOut: expected.String(), Executable: executable})
		}
	}
	if success < 10 {
		t.Fatal("too few executable comparison cases")
	}
	if path := os.Getenv("SPIRE_FIXTURE_PATH"); path != "" {
		raw, err := json.MarshalIndent(fixture, "", "  ")
		if err != nil {
			fail("fixture encode", err)
		}
		if err := os.WriteFile(path, append(raw, '\n'), 0644); err != nil {
			fail("fixture write", err)
		}
	}
	t.Logf("same-block comparison: block=%d cases=%d executable=%d; discovery replay passed", p.BlockNumber, len(fixture.Quotes), success)
}
