package btr

import (
	"bytes"
	"encoding/json"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestQuoteDifferential compares the sampled-ladder simulator against the
// canonical btr-quote service. Requires BTR_QUOTE_URL and a Monad BTR_RPC_URL;
// skipped in hermetic CI.
//
// Invariants:
//   - the simulator must refuse whenever the service refuses (gated/halted),
//     including the SWAP_GATED live core;
//   - when both quote, divergence <= 1 bps.
func TestQuoteDifferential(t *testing.T) {
	base := os.Getenv("BTR_QUOTE_URL")
	rpcURL := os.Getenv("BTR_RPC_URL")
	if base == "" || rpcURL == "" {
		t.Skip("BTR_QUOTE_URL and BTR_RPC_URL required")
	}

	client := ethrpc.New(rpcURL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tracker := NewPoolTracker(NewConfig(ChainIDMonad), client)
	ep := entity.Pool{
		Address: CreditPoolCore,
		Tokens: []*entity.PoolToken{
			{Address: USDC, Decimals: 6, Swappable: true},
			{Address: WMON, Decimals: 18, Swappable: true},
		},
		Reserves: entity.PoolReserves{"0", "0"},
	}
	tracked, err := tracker.GetNewPoolState(t.Context(), ep, pool.GetNewPoolStateParams{})
	if err != nil {
		t.Fatalf("track: %v", err)
	}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: tracked})
	if err != nil {
		t.Fatalf("simulator: %v", err)
	}
	t.Logf("block %d flags %v", tracked.BlockNumber, sim.extra.Flags)
	// An empty ladder would let the gate mask a broken tracker decode.
	ladder := sim.extra.Ladders[pairKey(WMON, USDC)]
	if len(ladder) == 0 {
		t.Fatalf("tracker built no WMON->USDC ladder: %s", tracked.Extra)
	}
	t.Logf("WMON->USDC ladder %v", ladder)

	httpc := &http.Client{Timeout: 10 * time.Second}
	sizes := []string{"100000000000000000", "1000000000000000000"}
	for _, s := range sizes {
		amount, _ := new(big.Int).SetString(s, 10)

		res, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: WMON, Amount: amount},
			TokenOut:      USDC,
		})

		reqBody, _ := json.Marshal(map[string]any{
			"chain_id": 143, "token_in": WMON, "token_out": USDC,
			"amounts_in": []string{"0x" + amount.Text(16)},
			"slippage":   map[string]any{"mode": "spread", "pct": 50},
		})
		resp, herr := httpc.Post(base+"/chain/quote", "application/json", bytes.NewReader(reqBody))
		if herr != nil {
			t.Fatalf("quote request: %v", herr)
		}
		var out struct {
			Amounts []struct {
				AmountOut string `json:"amount_out"`
			} `json:"amounts"`
			Flags struct {
				SwapEnabled bool `json:"swap_enabled"`
				Halted      bool `json:"halted"`
			} `json:"flags"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()

		serviceRefused := out.Flags.Halted || !out.Flags.SwapEnabled || len(out.Amounts) == 0 || out.Amounts[0].AmountOut == ""
		if serviceRefused {
			if simErr == nil {
				t.Fatalf("service refused but adapter quoted %s", res.TokenAmountOut.Amount)
			}
			continue
		}
		if simErr != nil {
			// The live core is SWAP_GATED, so the adapter refuses by design
			// even though the side-effect-free service returns a number.
			if gate(liveCoreFlags) != simErr {
				t.Fatalf("adapter refused a service-quoted leg: %v", simErr)
			}
			// Still check the sampled ladder against the service so the
			// pricing path is exercised while the gate is up.
			got, ierr := interpolate(ladder, amount)
			if ierr != nil {
				t.Fatalf("interpolate: %v", ierr)
			}
			res = &pool.CalcAmountOutResult{TokenAmountOut: &pool.TokenAmount{Token: USDC, Amount: got}}
		}
		if res == nil {
			t.Fatal("no adapter quote")
		}
		want, _ := new(big.Int).SetString(trim0x(out.Amounts[0].AmountOut), 16)
		diff := new(big.Int).Sub(res.TokenAmountOut.Amount, want)
		diff.Abs(diff)
		if diff.Sign() > 0 && new(big.Int).Mul(diff, big.NewInt(10_000)).Cmp(want) > 0 {
			t.Fatalf("divergence > 1 bps: adapter=%s service=%s", res.TokenAmountOut.Amount, want)
		}
	}
}

func trim0x(s string) string {
	if len(s) > 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}
