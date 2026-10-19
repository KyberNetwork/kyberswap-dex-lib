package erc4626_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	erc4626lazy "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626/lazy"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc7575"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/frax/frxusd"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// A vault without depositsPausedMethod/maxSupplyMethod must behave exactly as before they existed: same RPC
// calls, same Extra JSON (no "p"/"mH") and same simulator results. The golden was captured from origin/main.
// Regenerate (only from a tree without the pause/cap feature) with UPDATE_GOLDEN=1.
const goldenPath = "testdata/unconfigured_golden.json"

const (
	vaultAddr = "0x00000000000000000000000000000000000000a1"
	shareAddr = "0x00000000000000000000000000000000000000b2"
)

var cfgs = map[string]erc4626.VaultCfg{
	"both":        {Gas: erc4626.GasCfg{Deposit: 80000, Redeem: 90000}},
	"depositOnly": {Gas: erc4626.GasCfg{Deposit: 80000}, MinDepositMethod: "minAmount"},
	"redeemOnly":  {Gas: erc4626.GasCfg{Redeem: 90000}},
}

// recordCalls runs fn against a fake node that records every eth_call and fails it.
func recordCalls(t *testing.T, fn func(c *ethrpc.Client)) []string {
	t.Helper()
	var (
		mu    sync.Mutex
		calls []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage
			Method string
			Params []json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method == "eth_call" {
			mu.Lock()
			calls = append(calls, string(req.Params[0]))
			mu.Unlock()
		}
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"recorded"}}`, req.ID)
	}))
	defer srv.Close()
	fn(ethrpc.New(srv.URL))
	return calls
}

func poolFor(token0 string) entity.Pool {
	return entity.Pool{
		Address: vaultAddr, Exchange: erc4626.DexType, Type: erc4626.DexType, Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{{Address: token0, Swappable: true}, {Address: "0xasset", Swappable: true}},
		Extra:  `{"g":{"d":80000,"r":90000}}`,
	}
}

// fillLazy answers every call of a lazy request with a distinct value and returns the resulting entity.
func fillLazy(t *testing.T, req pool.ILazyRequest, apply func(*big.Int) (entity.Pool, error)) (calls []string, p entity.Pool) {
	t.Helper()
	for i, u := range req.GetUnpacks() {
		c := req.GetEthRpcCall(i)
		calls = append(calls, fmt.Sprintf("%s %s %v %s", c.Target, c.Method, c.Params, hex.EncodeToString(req.GetCallMsgs()[i].Data)))
		out := c.ABI.Methods[c.Method].Outputs
		var v any = new(big.Int).Mul(big.NewInt(int64(1000+i)), big.NewInt(1e15))
		if c.Method == erc4626.Erc4626MethodPreviewDeposit || c.Method == erc4626.Erc4626MethodPreviewRedeem {
			v = new(big.Int).Add(c.Params[0].(*big.Int), big.NewInt(int64(i))) // rate around 1:1
		}
		packed, err := out.Pack(v)
		require.NoError(t, err)
		require.NoError(t, u(packed))
	}
	p, err := apply(big.NewInt(123))
	require.NoError(t, err)
	return calls, p
}

func simResults(t *testing.T, extra string) []string {
	t.Helper()
	p := poolFor("0xvault")
	p.Extra = extra
	sim, err := erc4626.NewPoolSimulator(p)
	require.NoError(t, err)
	var out []string
	run := func(s *erc4626.PoolSimulator, tag string) {
		for _, as := range []string{"1", "1000000", "1000000000000", "3000000000000000", "1000000000000000000", "500000000000000000000", "1000000000000000000000000000000"} {
			amt, _ := new(big.Int).SetString(as, 10)
			for _, d := range [][2]string{{"0xasset", "0xvault"}, {"0xvault", "0xasset"}} {
				o, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: d[0], Amount: amt}, TokenOut: d[1]})
				if err != nil {
					out = append(out, fmt.Sprintf("%s out %s>%s %v err=%v", tag, d[0], d[1], amt, err))
				} else {
					out = append(out, fmt.Sprintf("%s out %s>%s %v => %v gas=%d", tag, d[0], d[1], amt, o.TokenAmountOut.Amount, o.Gas))
				}
				in, err := s.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: d[1], Amount: amt}, TokenIn: d[0]})
				if err != nil {
					out = append(out, fmt.Sprintf("%s in %s>%s %v err=%v", tag, d[0], d[1], amt, err))
				} else {
					out = append(out, fmt.Sprintf("%s in %s>%s %v => %v gas=%d", tag, d[0], d[1], amt, in.TokenAmountIn.Amount, in.Gas))
				}
			}
		}
	}
	run(sim, "fresh")
	cl := sim.CloneState().(*erc4626.PoolSimulator)
	for _, d := range [][2]string{{"0xasset", "0xvault"}, {"0xvault", "0xasset"}} {
		cl.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: d[0], Amount: big.NewInt(1e15)},
			TokenAmountOut: pool.TokenAmount{Token: d[1], Amount: big.NewInt(1e15)},
		})
	}
	run(cl, "afterUpdate")
	run(sim, "originalAfterCloneUpdate")
	return out
}

func TestUnconfiguredVaultMatchesOriginMain(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	got := map[string]any{}
	for name, cfg := range cfgs {
		cfgs := &erc4626.Config{DexId: "x", Vaults: map[string]erc4626.VaultCfg{vaultAddr: cfg}}

		// plain tracker, backup path
		got["plain/"+name] = recordCalls(t, func(c *ethrpc.Client) {
			_, _ = erc4626.NewPoolTracker(cfgs, c).GetNewPoolState(ctx, poolFor(vaultAddr), pool.GetNewPoolStateParams{})
		})
		// lazy FetchAssetAndState: standard (share == vault) and erc7575 (separate share)
		got["lazyFetch/"+name] = recordCalls(t, func(c *ethrpc.Client) {
			_, _, _ = erc4626lazy.FetchAssetAndState(ctx, c, vaultAddr, "", cfg, true, nil)
		})
		got["lazyFetchShare/"+name] = recordCalls(t, func(c *ethrpc.Client) {
			_, _, _ = erc4626lazy.FetchAssetAndState(ctx, c, vaultAddr, shareAddr, cfg, true, nil)
		})
		// erc7575 and frxusd trackers
		got["erc7575/"+name] = recordCalls(t, func(c *ethrpc.Client) {
			_, _ = erc7575.NewPoolTracker(cfgs, c).GetNewPoolState(ctx, poolFor(shareAddr), pool.GetNewPoolStateParams{})
		})
		got["frxusd/"+name] = recordCalls(t, func(c *ethrpc.Client) {
			_, _ = frxusd.NewPoolTracker(cfgs, c).GetNewPoolState(ctx, poolFor(vaultAddr), pool.GetNewPoolStateParams{})
		})

		// lazy tracker (primary): calls and the Extra it writes
		for tag, tokenAddr := range map[string]string{"": "", "Share": shareAddr} {
			c := ethrpc.New("http://127.0.0.1:0")
			p := poolFor(vaultAddr)
			req, apply := erc4626lazy.Lazycall(ctx, &p, c, vaultAddr, tokenAddr, cfg, false, nil)
			calls, np := fillLazy(t, req, apply)
			got["lazy"+tag+"/"+name] = map[string]any{"calls": calls, "extra": np.Extra}
			assert.NotContains(t, np.Extra, `"p"`)
			assert.NotContains(t, np.Extra, `"mH"`)
		}

		// UpdateEntityState with a fixed state, shared by plain/frxusd/erc7575
		st := &erc4626.PoolState{
			MaxDeposit: big.NewInt(1e18), MaxRedeem: big.NewInt(2e18), MinDeposit: big.NewInt(5), TotalAssets: big.NewInt(7e18),
			DepositRates: make([]*big.Int, len(erc4626.PrefetchAmounts)), RedeemRates: make([]*big.Int, len(erc4626.PrefetchAmounts)),
		}
		for i, a := range erc4626.PrefetchAmounts {
			st.DepositRates[i], st.RedeemRates[i] = a.ToBig(), new(big.Int).Add(a.ToBig(), big.NewInt(1))
		}
		p := poolFor(vaultAddr)
		require.NoError(t, erc4626.UpdateEntityState(&p, cfg, st))
		got["updateEntity/"+name] = p.Extra
		assert.NotContains(t, p.Extra, `"p"`)
		assert.NotContains(t, p.Extra, `"mH"`)

		got["sim/"+name] = simResults(t, p.Extra)
	}

	gotJSON, err := json.MarshalIndent(got, "", " ")
	require.NoError(t, err)
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(goldenPath, append(gotJSON, '\n'), 0o644))
		return
	}
	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	require.JSONEq(t, string(want), string(gotJSON))
}

// A vault with a redeem side and maxSupplyMethod needs totalSupply once: the cap read serves the redeem
// fallback too.
func TestTotalSupplyReadOnce(t *testing.T) {
	t.Parallel()
	const totalSupplySel = "18160ddd"
	cfg := erc4626.VaultCfg{Gas: erc4626.GasCfg{Deposit: 80000, Redeem: 90000}, MaxSupplyMethod: "maxSupply"}
	cfgs := &erc4626.Config{DexId: "x", Vaults: map[string]erc4626.VaultCfg{vaultAddr: cfg}}

	count := func(calls []string) int {
		n := 0
		for _, c := range calls {
			n += strings.Count(c, totalSupplySel)
		}
		return n
	}
	plain := recordCalls(t, func(c *ethrpc.Client) {
		_, _ = erc4626.NewPoolTracker(cfgs, c).GetNewPoolState(context.Background(), poolFor(vaultAddr), pool.GetNewPoolStateParams{})
	})
	assert.Equal(t, 1, count(plain), "plain tracker")
	assert.Equal(t, 1, count(recordCalls(t, func(c *ethrpc.Client) {
		_, _, _ = erc4626lazy.FetchAssetAndState(context.Background(), c, vaultAddr, shareAddr, cfg, false, nil)
	})), "lazy tracker")

	// without maxSupplyMethod the redeem block still reads it
	cfg.MaxSupplyMethod = ""
	cfgs.Vaults[vaultAddr] = cfg
	assert.Equal(t, 1, count(recordCalls(t, func(c *ethrpc.Client) {
		_, _, _ = erc4626lazy.FetchAssetAndState(context.Background(), c, vaultAddr, shareAddr, cfg, false, nil)
	})), "redeem only read")
}
