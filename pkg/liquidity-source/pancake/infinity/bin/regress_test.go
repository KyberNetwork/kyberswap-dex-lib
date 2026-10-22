package bin

import (
	"encoding/json"
	"fmt"
	"maps"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/internal/fakerpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Regression guard for the shared Bin code that squadswap-bin (Thanos) touched. Goldens in
// testdata/regress_bin_golden.json were generated from origin/main (REGRESS_GEN=<path> go test -run
// TestRegress), so every non-Thanos pancake-infinity-bin / omni-bin pool must keep quoting and
// fee-ing exactly as before. Cases listed in changedByDesign are the only intended differences.

const (
	regressGolden = "testdata/regress_bin_golden.json"
	brevisHook    = "0x60fbcafab24bc117b6facecd00d3e8f56ca4d5e9"
)

// changedByDesign: dynamic-fee pool without swap callbacks used to get lpFee = 100% from BaseHook.
var changedByDesign = map[string]string{
	"fee/dynamic-nocallbacks": "swapFee=500",
}

func regressCheck(t *testing.T, got map[string]string) {
	t.Helper()
	if path := os.Getenv("REGRESS_GEN"); path != "" {
		b, _ := json.MarshalIndent(got, "", " ")
		require.NoError(t, os.WriteFile(path, b, 0o644))
		return
	}
	var want map[string]string
	b, err := os.ReadFile(regressGolden)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &want))
	for k, w := range want {
		if d, ok := changedByDesign[k]; ok {
			require.Equal(t, d, got[k], k)
			continue
		}
		require.Equal(t, w, got[k], k)
	}
}

func regressSimOutputs(t *testing.T) map[string]string {
	data, err := os.ReadFile("sample_pool.json")
	require.NoError(t, err)
	var base entity.Pool
	require.NoError(t, json.Unmarshal(data, &base))

	variants := map[string]func(*entity.Pool){
		"plain": func(*entity.Pool) {},
		"brevis": func(p *entity.Pool) { // swap callbacks + registered hook
			p.StaticExtra = strings.Replace(strings.Replace(p.StaticExtra, `"hsp":false`, `"hsp":true`, 1),
				`"hooks":"0x0000000000000000000000000000000000000000"`, `"hooks":"`+brevisHook+`"`, 1)
		},
		"omni": func(p *entity.Pool) { p.Exchange = "omni-bin" },
	}
	amounts := []string{"1", "1000", "1000000000000000", "50000000000000000000", "500000000000000000000000"}
	out := map[string]string{}
	for name, mut := range variants {
		ent := base
		mut(&ent)
		sim, err := NewPoolSimulator(ent, valueobject.ChainIDBSC)
		require.NoError(t, err, name)
		tk := ent.Tokens
		for dir := range 2 {
			in, to := tk[dir].Address, tk[1-dir].Address
			for _, a := range amounts {
				amt, _ := new(big.Int).SetString(a, 10)
				key := fmt.Sprintf("sim/%s/%d/%s", name, dir, a)
				res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt}, TokenOut: to})
				if err != nil {
					out[key+"/calcOut"] = "err:" + err.Error()
				} else {
					out[key+"/calcOut"] = fmt.Sprintf("%s fee=%s gas=%d", res.TokenAmountOut.Amount, res.Fee.Amount, res.Gas)
				}
				res2, err := sim.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{Token: to, Amount: amt}, TokenIn: in})
				if err != nil {
					out[key+"/calcIn"] = "err:" + err.Error()
				} else {
					out[key+"/calcIn"] = fmt.Sprintf("%s fee=%s gas=%d", res2.TokenAmountIn.Amount, res2.Fee.Amount, res2.Gas)
				}
			}
		}
		// sequential swaps: UpdateBalance must move state identically
		cl := sim.CloneState()
		for i := range 3 {
			amt := big.NewInt(1e18)
			res, err := cl.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tk[0].Address, Amount: amt}, TokenOut: tk[1].Address})
			require.NoError(t, err)
			out[fmt.Sprintf("seq/%s/%d", name, i)] = res.TokenAmountOut.Amount.String()
			cl.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  pool.TokenAmount{Token: tk[0].Address, Amount: amt},
				TokenAmountOut: pool.TokenAmount{Token: tk[1].Address, Amount: res.TokenAmountOut.Amount},
				Fee:            *res.Fee, SwapInfo: res.SwapInfo})
		}
	}
	return out
}

func regressFeeOutputs(t *testing.T) map[string]string {
	const mgr = "0xc697d2898e0d09264376196696c51d7abbbaa4a9"
	const poolID = "0xa859b22e97f32d4c7b1d9788044697712cb7183d294ed7aa1832799c1739e5cf"
	origFeeABI, _ := abi.JSON(strings.NewReader(`[{"name":"origFee","type":"function","inputs":[],"outputs":[{"type":"uint24"}]}]`))

	type tc struct {
		name        string
		fee         uint32
		hsp         bool
		hook        string
		protocolFee uint64
		slot0LpFee  uint64
		origFee     uint64
	}
	cases := []tc{
		{name: "fee/fixed", fee: 3000},
		{name: "fee/fixed-protocol", fee: 3000, protocolFee: 131104},
		{name: "fee/dynamic-brevis", fee: shared.DYNAMIC_FEE_FLAG, hsp: true, hook: brevisHook, slot0LpFee: 111, origFee: 2500},
		{name: "fee/dynamic-brevis-protocol", fee: shared.DYNAMIC_FEE_FLAG, hsp: true, hook: brevisHook, protocolFee: 131104, origFee: 2500},
		{name: "fee/dynamic-unregistered-callbacks", fee: shared.DYNAMIC_FEE_FLAG, hsp: true,
			hook: "0x1111111111111111111111111111111111111111", slot0LpFee: 500},
		{name: "fee/dynamic-nocallbacks", fee: shared.DYNAMIC_FEE_FLAG, slot0LpFee: 500},
	}
	out := map[string]string{}
	for _, c := range cases {
		hook := c.hook
		if hook == "" {
			hook = "0x0000000000000000000000000000000000000000"
		}
		slot0, perr := shared.BinPoolManagerABI.Methods["getSlot0"].Outputs.Pack(
			uint32(8388609), uint32(c.protocolFee), uint32(c.slot0LpFee))
		require.NoError(t, perr)
		of, _ := origFeeABI.Methods["origFee"].Outputs.Pack(new(big.Int).SetUint64(c.origFee))
		rpc := fakerpc.New(t)
		rpc.Calls[fakerpc.Key(mgr, shared.BinPoolManagerABI, "getSlot0")] = slot0
		rpc.Calls[fakerpc.Key(hook, origFeeABI, "origFee")] = of
		tr := NewPoolTracker(&Config{BinPoolManagerAddress: mgr}, rpc.Client(), nil)
		se, _ := json.Marshal(StaticExtra{HasSwapPermissions: c.hsp, Fee: c.fee, HooksAddress: common.HexToAddress(hook)})
		res, err := tr.FetchRPCData(t.Context(), &entity.Pool{Address: poolID, StaticExtra: string(se)}, 0)
		require.NoError(t, err, c.name)
		out[c.name] = fmt.Sprintf("swapFee=%d", res.SwapFee)
	}
	return out
}

func TestRegress(t *testing.T) {
	got := regressSimOutputs(t)
	maps.Copy(got, regressFeeOutputs(t))
	regressCheck(t, got)
}
