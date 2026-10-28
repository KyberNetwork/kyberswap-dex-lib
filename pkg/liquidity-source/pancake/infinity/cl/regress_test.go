package cl

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	factory "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/internal/fakerpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

// Regression guard for the shared CL tracker that squadswap-cl (Thanos) touched. Goldens in
// testdata/regress_cl_golden.json come from origin/main (REGRESS_GEN=<path> go test -run TestRegress).
// Subgraph-mode pools (every existing pancake-infinity-cl / omni-cl dex) must keep their
// p.BlockNumber, and FetchPoolTicks must keep reading ticks at exactly p.BlockNumber.

const (
	regressGolden = "testdata/regress_cl_golden.json"
	clMgr         = "0xa0ffb9c1ce1fe56963b0321b32e7a0302114058b"
	clPoolID      = "0x0000000000000000000000000000000000000000000000000000000000000abc"
)

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
		require.Equal(t, w, got[k], k)
	}
}

// onChainTicks: tick -> (gross, net); tick 0 is uninitialised (gross 0, must be dropped).
var onChainTicks = map[int64][2]int64{-120: {500, 500}, -60: {300, 300}, 0: {0, 0}, 60: {300, -300}, 120: {500, -500}}

func newCLRPC(t *testing.T) *fakerpc.Server {
	rpc := fakerpc.New(t)
	rpc.Handler = func(to string, data []byte) []byte {
		m := shared.CLPoolManagerABI.Methods["getPoolTickInfo"]
		if to != clMgr || string(data[:4]) != string(m.ID) {
			return nil
		}
		args, err := m.Inputs.Unpack(data[4:])
		if err != nil {
			panic(err)
		}
		v := onChainTicks[args[1].(*big.Int).Int64()]
		out, err := m.Outputs.Pack(factory.TickInfo{
			LiquidityGross: big.NewInt(v[0]), LiquidityNet: big.NewInt(v[1]),
			FeeGrowthOutside0X128: big.NewInt(0), FeeGrowthOutside1X128: big.NewInt(0)})
		if err != nil {
			panic(err)
		}
		return out
	}
	slot0, err := shared.CLPoolManagerABI.Methods["getSlot0"].Outputs.Pack(
		new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(0), uint32(0), uint32(3000))
	require.NoError(t, err)
	liq, err := shared.CLPoolManagerABI.Methods["getLiquidity"].Outputs.Pack(big.NewInt(1000))
	require.NoError(t, err)
	rpc.Calls[fakerpc.Key(clMgr, shared.CLPoolManagerABI, "getSlot0")] = slot0
	rpc.Calls[fakerpc.Key(clMgr, shared.CLPoolManagerABI, "getLiquidity")] = liq
	return rpc
}

const clStaticExtra = `{"hsp":false,"0x0":[false,false],"fee":3000,"params":"0x0","tS":60,` +
	`"pm":"0xa0ffb9c1ce1fe56963b0321b32e7a0302114058b","hooks":"0x0000000000000000000000000000000000000000"}`

func regressCLOutputs(t *testing.T) map[string]string {
	out := map[string]string{}

	// FetchPoolTicks pins the refetch to p.BlockNumber (0 = latest).
	for _, bn := range []uint64{0, 5000} {
		rpc := newCLRPC(t)
		tr := NewPoolTracker(&Config{CLPoolManagerAddress: clMgr}, rpc.Client(), nil)
		p, err := tr.FetchPoolTicks(t.Context(), entity.Pool{Address: clPoolID, BlockNumber: bn,
			Extra: `{"ticks":[{"index":120,"liquidityGross":1,"liquidityNet":-1},{"index":-60,"liquidityGross":1,"liquidityNet":1},` +
				`{"index":0,"liquidityGross":1,"liquidityNet":1}]}`})
		require.NoError(t, err)
		var e struct{ Ticks []Tick }
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
		out[fmt.Sprintf("fetchTicks/%d", bn)] = fmt.Sprintf("%+v blocks=%v", e.Ticks, rpc.Blocks())
	}

	// Bootstrap in subgraph mode (every dex but squadswap-cl): ticks from the subgraph, block untouched.
	gql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"ticks":[` +
			`{"tickIdx":"-60","liquidityGross":"300","liquidityNet":"300"},` +
			`{"tickIdx":"60","liquidityGross":"300","liquidityNet":"-300"}]}}`))
	}))
	defer gql.Close()
	rpc := newCLRPC(t)
	cfg := &Config{DexID: "pancake-infinity-cl", SubgraphAPI: gql.URL, CLPoolManagerAddress: clMgr}
	tr := NewPoolTracker(cfg, rpc.Client(), graphqlpkg.NewClient(gql.URL))
	p, err := tr.BootstrapPoolState(t.Context(), entity.Pool{Address: clPoolID, BlockNumber: 777,
		Reserves: entity.PoolReserves{"0", "0"}, StaticExtra: clStaticExtra}, poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)
	out["bootstrap/subgraph"] = fmt.Sprintf("block=%d fee=%v reserves=%v extra=%s", p.BlockNumber, p.SwapFee, p.Reserves,
		strings.ReplaceAll(p.Extra, " ", ""))
	return out
}

func TestRegress(t *testing.T) {
	regressCheck(t, regressCLOutputs(t))
}
