package twocryptong_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	altjson "github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	twocryptong "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/twocrypto-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/twocrypto-ng/lazy"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const tryBlockAndAggregateABI = `[{"name":"tryBlockAndAggregate","type":"function","stateMutability":"nonpayable",
"inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[
{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},
{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`

type multicallResult struct {
	Success    bool
	ReturnData []byte
}

// poolServer answers tryBlockAndAggregate like a twocrypto-ng pool at block 12: each call's answer comes
// from its method name, every view returns value(method), and the methods in reverts revert.
func poolServer(t *testing.T, value func(method string) any, reverts ...string) *httptest.Server {
	t.Helper()
	mc, err := gethabi.JSON(strings.NewReader(tryBlockAndAggregateABI))
	require.NoError(t, err)
	method := mc.Methods["tryBlockAndAggregate"]
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		require.Equal(t, "eth_call", req.Method)
		var msg struct {
			Data  hexutil.Bytes `json:"data"`
			Input hexutil.Bytes `json:"input"`
		}
		require.NoError(t, json.Unmarshal(req.Params[0], &msg))
		data := msg.Data
		if len(data) == 0 {
			data = msg.Input
		}
		args, err := method.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		calls := args[1].([]struct {
			Target   common.Address `json:"target"`
			CallData []byte         `json:"callData"`
		})
		results := make([]multicallResult, len(calls))
		for i, c := range calls {
			m, err := twocryptong.CurveTwocryptoNGABI.MethodById(c.CallData[:4])
			if err != nil {
				m, err = twocryptong.YBTwocryptoPolicyABI.MethodById(c.CallData[:4])
			}
			require.NoError(t, err)
			if slices.Contains(reverts, m.RawName) {
				continue
			}
			v := value(m.RawName)
			args, many := v.([]any)
			if !many {
				args = []any{v}
			}
			out, err := m.Outputs.Pack(args...)
			require.NoError(t, err)
			results[i] = multicallResult{Success: true, ReturnData: out}
		}
		out, err := method.Outputs.Pack(big.NewInt(12), common.Hash{}, results)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(out)})
	}))
}

// packedRebalancingParams is [allowed_extra_profit, adjustment_step, ma_time], 64 bits each, as the
// pool stores them: 0x83f24023 holds 1e8, 1e11 and 866, and its ma_time() view says 601.
func packedRebalancingParams(allowedExtraProfit, adjustmentStep, maTime int64) *big.Int {
	packed := new(big.Int).Lsh(big.NewInt(allowedExtraProfit), 128)
	packed.Or(packed, new(big.Int).Lsh(big.NewInt(adjustmentStep), 64))
	return packed.Or(packed, big.NewInt(maTime))
}

func viewValues(packed *big.Int) func(string) any {
	return func(method string) any {
		switch method {
		case "packed_rebalancing_params":
			return packed
		case "MATH", "POLICY":
			return common.Address{}
		case "version":
			return "v2.0.0"
		case "ma_time":
			return big.NewInt(601)
		case "initial_A_gamma", "future_A_gamma":
			return new(big.Int).Or(new(big.Int).Lsh(big.NewInt(90000), 128), big.NewInt(1000000000000000))
		case "initial_A_gamma_time", "future_A_gamma_time":
			return big.NewInt(1758724751)
		default:
			return big.NewInt(1e18)
		}
	}
}

type tracker interface {
	GetNewPoolState(context.Context, entity.Pool, pool.GetNewPoolStateParams) (entity.Pool, error)
}

func trackers(t *testing.T, url string) map[string]tracker {
	t.Helper()
	client := ethrpc.New(url).SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))
	cfg := &shared.Config{DexID: twocryptong.DexType}
	main, err := twocryptong.NewPoolTracker(cfg, client)
	require.NoError(t, err)
	lz, err := lazy.NewPoolTracker(cfg, client)
	require.NoError(t, err)
	return map[string]tracker{"tracker": main, "lazy tracker": lz}
}

// track runs tr once and returns the Extra it stored, the pool, and the time window it ran in.
func track(t *testing.T, tr tracker) (twocryptong.Extra, entity.Pool, int64, int64) {
	t.Helper()
	p := entity.Pool{
		Address: "0x83f24023d15d835a213df24fd309c47dab5beb32",
		Tokens: []*entity.PoolToken{{Address: "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e"},
			{Address: "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf"}},
		Reserves: entity.PoolReserves{"0", "0"},
		Extra:    "{}",
	}
	before := time.Now().Unix()
	got, err := tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	after := time.Now().Unix()
	require.NoError(t, err)
	var extra twocryptong.Extra
	require.NoError(t, json.Unmarshal([]byte(got.Extra), &extra))
	return extra, got, before, after
}

// The trackers store the EMA's time constant from packed_rebalancing_params (not the ma_time() view)
// and the time they read price_oracle, which is also the pool's timestamp.
func TestPoolTrackers_StorePriceOracleReadTimeAndMaTime(t *testing.T) {
	srv := poolServer(t, viewValues(packedRebalancingParams(1e8, 1e11, 866)))
	defer srv.Close()
	for name, tr := range trackers(t, srv.URL) {
		t.Run(name, func(t *testing.T) {
			extra, got, before, after := track(t, tr)
			require.NotNil(t, extra.MaTime)
			assert.Equal(t, uint64(866), extra.MaTime.Uint64())
			assert.GreaterOrEqual(t, extra.PriceOracleTimestamp, before)
			assert.LessOrEqual(t, extra.PriceOracleTimestamp, after)
			assert.Equal(t, got.Timestamp, extra.PriceOracleTimestamp)
			assert.Equal(t, uint64(12), got.BlockNumber)
			assert.Equal(t, uint64(1e18), extra.AllowedExtraProfit.Uint64(), "allowed_extra_profit still comes from its own view")
		})
	}
}

// A pool without packed_rebalancing_params, or with ma_time 0 in it, leaves both fields out, so the
// simulator uses the stored price_oracle as it did before (TestPriceOracleNotCarriedForward_OldState).
func TestPoolTrackers_NoMaTimeLeavesTheFieldsOut(t *testing.T) {
	for name, value := range map[string]struct {
		values  func(string) any
		reverts []string
	}{
		"packed_rebalancing_params reverts": {viewValues(nil), []string{"packed_rebalancing_params"}},
		"ma_time 0":                         {viewValues(packedRebalancingParams(1e8, 1e11, 0)), nil},
	} {
		srv := poolServer(t, value.values, value.reverts...)
		for trName, tr := range trackers(t, srv.URL) {
			t.Run(name+"/"+trName, func(t *testing.T) {
				extra, got, _, _ := track(t, tr)
				assert.Nil(t, extra.MaTime)
				assert.Zero(t, extra.PriceOracleTimestamp)
				assert.NotContains(t, got.Extra, "MaTime")
				assert.NotContains(t, got.Extra, "PriceOracleTimestamp")
				assert.Equal(t, uint64(12), got.BlockNumber)
			})
		}
		srv.Close()
	}
}

// The trackers store version() and, where the pool has them, last_timestamp and the donation views that
// v2.1.0d's rebalance reads.
func TestPoolTrackers_StoreV21dState(t *testing.T) {
	values := viewValues(packedRebalancingParams(1e6, 5e15, 866))
	srv := poolServer(t, func(method string) any {
		switch method {
		case "version":
			return "v2.1.0d"
		case "last_timestamp":
			return big.NewInt(1793375255)
		case "donation_shares":
			return big.NewInt(395782627665194639)
		case "donation_duration":
			return big.NewInt(604800)
		case "last_donation_release_ts":
			return big.NewInt(1793373191)
		case "donation_protection_expiry_ts":
			return big.NewInt(1793376611)
		case "donation_protection_period":
			return big.NewInt(3600)
		}
		return values(method)
	})
	defer srv.Close()
	for name, tr := range trackers(t, srv.URL) {
		t.Run(name, func(t *testing.T) {
			extra, _, _, _ := track(t, tr)
			assert.Equal(t, "v2.1.0d", extra.Version)
			assert.Equal(t, int64(1793375255), extra.LastTimestamp)
			require.NotNil(t, extra.DonationShares)
			require.NotNil(t, extra.DonationDuration)
			require.NotNil(t, extra.DonationProtectionPeriod)
			assert.Equal(t, "395782627665194639", extra.DonationShares.Dec())
			assert.Equal(t, "604800", extra.DonationDuration.Dec())
			assert.Equal(t, int64(1793373191), extra.LastDonationReleaseTs)
			assert.Equal(t, int64(1793376611), extra.DonationProtectionExpiryTs)
			assert.Equal(t, "3600", extra.DonationProtectionPeriod.Dec())
		})
	}
}

// A v2.0.0 pool has no donation views: the trackers store its version and no donation state.
func TestPoolTrackers_StoreNoDonationStateWithoutTheViews(t *testing.T) {
	srv := poolServer(t, viewValues(packedRebalancingParams(1e8, 1e11, 866)), "donation_shares",
		"donation_duration", "last_donation_release_ts", "donation_protection_expiry_ts", "donation_protection_period")
	defer srv.Close()
	for name, tr := range trackers(t, srv.URL) {
		t.Run(name, func(t *testing.T) {
			extra, got, _, _ := track(t, tr)
			assert.Equal(t, "v2.0.0", extra.Version)
			assert.Nil(t, extra.DonationShares)
			assert.NotContains(t, got.Extra, "Donation")
		})
	}
}

const wbtcPolicy = "0x0436c0648afd66de23f86f23cb42884304743efa"

// v3Views answers a v3.0.0 pool under policy and its policy's views.
func v3Views(policy common.Address) func(string) any {
	values := viewValues(new(big.Int).Or(new(big.Int).Lsh(big.NewInt(1e10), 128),
		new(big.Int).Or(new(big.Int).Lsh(big.NewInt(5e15), 64), big.NewInt(865))))
	n := func(s string) *big.Int { v, _ := new(big.Int).SetString(s, 10); return v }
	return func(method string) any {
		switch method {
		case "version":
			return "v3.0.0"
		case "POLICY":
			return policy
		case "lp_xcp_profit":
			return n("1016209130716660647")
		case "reserved_profit_fraction":
			return n("5000000000")
		case "admin_fee":
			return n("5000000000")
		case "state":
			return []any{big.NewInt(1793379071), n("77646391836179594649312"),
				n("77634216023848666491693"), n("79013462006129642748426"), n("72156952696539490158287")}
		case "FAST_HALF_LIFE":
			return big.NewInt(8100)
		case "SLOW_HALF_LIFE":
			return big.NewInt(57600)
		case "KAPPA":
			return n("1300000000000000000")
		case "DEADBAND":
			return n("800000000000000")
		case "MIN_CAP":
			return n("1200000000000000")
		case "MAX_CAP":
			return n("5400000000000000")
		}
		return values(method)
	}
}

func trackWithPolicy(t *testing.T, tr tracker, previousPolicy string) twocryptong.Extra {
	t.Helper()
	p := entity.Pool{
		Address: "0x31369866",
		Tokens: []*entity.PoolToken{{Address: "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e"},
			{Address: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599"}},
		Reserves: entity.PoolReserves{"0", "0"},
		Extra:    `{"Policy":"` + previousPolicy + `"}`,
	}
	got, err := tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	var extra twocryptong.Extra
	require.NoError(t, altjson.Unmarshal([]byte(got.Extra), &extra))
	return extra
}

// The trackers store v3.0.0's profit and fee-split views, the step range from packed_rebalancing_params and,
// for a YBTwocryptoPolicy the pool already had at the previous refresh, the policy's state and immutables.
func TestPoolTrackers_StoreV3State(t *testing.T) {
	srv := poolServer(t, v3Views(common.HexToAddress(wbtcPolicy)), "allowed_extra_profit")
	defer srv.Close()
	for name, tr := range trackers(t, srv.URL) {
		t.Run(name, func(t *testing.T) {
			extra := trackWithPolicy(t, tr, wbtcPolicy)
			assert.Equal(t, "v3.0.0", extra.Version)
			require.NotNil(t, extra.LpXcpProfit)
			assert.Equal(t, "1016209130716660647", extra.LpXcpProfit.Dec())
			require.NotNil(t, extra.ReservedProfitFraction)
			require.NotNil(t, extra.AdminFee)
			assert.Equal(t, "5000000000", extra.AdminFee.Dec())
			require.NotNil(t, extra.AdjustmentStepMin)
			assert.Equal(t, "10000000000", extra.AdjustmentStepMin.Dec())
			assert.Equal(t, "5000000000000000", extra.AdjustmentStepMax.Dec())
			require.NotNil(t, extra.PolicyState)
			s := extra.PolicyState
			assert.Equal(t, int64(1793379071), s.LastUpdateTs)
			assert.Equal(t, "72156952696539490158287", s.PriceScale.Dec())
			assert.Equal(t, "79013462006129642748426", s.SlowEma.Dec())
			assert.Equal(t, "5400000000000000", s.MaxCap.Dec())
		})
	}
}

// No policy state when the pool's policy changed since the previous refresh, or is not a YBTwocryptoPolicy.
func TestPoolTrackers_StoreNoPolicyStateUnlessKnown(t *testing.T) {
	srv := poolServer(t, v3Views(common.HexToAddress(wbtcPolicy)), "allowed_extra_profit")
	defer srv.Close()
	for name, tr := range trackers(t, srv.URL) {
		for _, previous := range []string{"", "0x038f549d21b9c22a26ada4da2ebf7cfd1c3ca22d",
			"0xaea95d551154443e73565be724d59d63d72f8a7d"} {
			t.Run(name+" "+previous, func(t *testing.T) {
				extra := trackWithPolicy(t, tr, previous)
				assert.Equal(t, wbtcPolicy, extra.Policy)
				assert.Nil(t, extra.PolicyState)
			})
		}
	}
}

// v3.0.0 pools as the trackers store them at block 26307080 (anvil fork): the tracker's read, with the
// stableswap math selected and the pool's POLICY. The BTC pools' policies are YBTwocryptoPolicy, whose get_fee is
// pure and returns 0 (the pool's own fee applies); the WETH pool's is YBOraclizedPolicy, whose fee depends on the
// trade and a signed price report.
const (
	poolV3WBTC       = `{"address":"0x313698667d7fdd6789a9bc70821309ff891e729a","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1793600892,"reserves":["34253607650019964044778625","31224663074"],"tokens":[{"address":"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599","symbol":"WBTC","decimals":8,"swappable":true}],"extra":"{\"InitialA\":\"50000\",\"InitialGamma\":\"11111111111\",\"FutureA\":\"50000\",\"FutureGamma\":\"11111111111\",\"FutureAGammaTime\":0,\"D\":\"56575839380905789716926673\",\"PriceScale\":[\"72156952696539490158287\"],\"PriceOracle\":[\"77644190031455962596650\"],\"LastPrices\":[\"77646391836179594649312\"],\"PriceOracleTimestamp\":1793600892,\"MaTime\":\"865\",\"FeeGamma\":\"30000000000000000\",\"MidFee\":\"130000000\",\"OutFee\":\"130000000\",\"LpSupply\":\"104126966628027347437547\",\"XcpProfit\":\"1036450864470785641\",\"VirtualPrice\":\"1011343532348392001\",\"AllowedExtraProfit\":null,\"AdjustmentStep\":\"100000000\",\"UseCustomMath\":true,\"Policy\":\"0x0436c0648afd66de23f86f23cb42884304743efa\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}","blockNumber":26307080}`
	poolV3cbBTCFresh = `{"address":"0x862cb4e988fb66e72f128d1183829f8c05b6c6a0","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1793600896,"reserves":["27756702231009188438738141","34993243729"],"tokens":[{"address":"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf","symbol":"cbBTC","decimals":8,"swappable":true}],"extra":"{\"InitialA\":\"50000\",\"InitialGamma\":\"11111111111\",\"FutureA\":\"50000\",\"FutureGamma\":\"11111111111\",\"FutureAGammaTime\":0,\"D\":\"54801030512832732734229143\",\"PriceScale\":[\"77286628838017654881551\"],\"PriceOracle\":[\"77621960495970162655012\"],\"LastPrices\":[\"77621960447248580232137\"],\"PriceOracleTimestamp\":1793600896,\"MaTime\":\"865\",\"FeeGamma\":\"30000000000000000\",\"MidFee\":\"130000000\",\"OutFee\":\"130000000\",\"LpSupply\":\"97097012358894162425702\",\"XcpProfit\":\"1047730358031930301\",\"VirtualPrice\":\"1015080957106381580\",\"AllowedExtraProfit\":null,\"AdjustmentStep\":\"100000000\",\"UseCustomMath\":true,\"Policy\":\"0x038f549d21b9c22a26ada4da2ebf7cfd1c3ca22d\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}","blockNumber":26307080}`
	poolV3tBTCFresh  = `{"address":"0x4f52c3a81e33521e5a9a47fd9d3be475d2279c2e","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1793600899,"reserves":["21203026424257775725825133","191704654592031099106"],"tokens":[{"address":"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0x18084fba666a33d37592fa2633fd49a74dd93a88","symbol":"tBTC","decimals":18,"swappable":true}],"extra":"{\"InitialA\":\"50000\",\"InitialGamma\":\"11111111111\",\"FutureA\":\"50000\",\"FutureGamma\":\"11111111111\",\"FutureAGammaTime\":0,\"D\":\"34866851668698472812011087\",\"PriceScale\":[\"71980553859879763736828\"],\"PriceOracle\":[\"77615373277783070471157\"],\"LastPrices\":[\"77615373230567813312479\"],\"PriceOracleTimestamp\":1793600899,\"MaTime\":\"865\",\"FeeGamma\":\"30000000000000000\",\"MidFee\":\"130000000\",\"OutFee\":\"130000000\",\"LpSupply\":\"64345677359056618444004\",\"XcpProfit\":\"1032418260550179630\",\"VirtualPrice\":\"1009847441433321294\",\"AllowedExtraProfit\":null,\"AdjustmentStep\":\"100000000\",\"UseCustomMath\":true,\"Policy\":\"0xe99ee179f1cc62d4a1d3e67dd5541ecb63aca396\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}","blockNumber":26307080}`
	poolV3WETH       = `{"address":"0x656341ef90b622c6634e0573772ffb7f3669b9f3","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1793600902,"reserves":["14491129749364977540372168","6303251288201586488073"],"tokens":[{"address":"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"InitialA\":\"50000\",\"InitialGamma\":\"11111111111\",\"FutureA\":\"50000\",\"FutureGamma\":\"11111111111\",\"FutureAGammaTime\":0,\"D\":\"28502785833946151738901397\",\"PriceScale\":[\"2223031549528268495568\"],\"PriceOracle\":[\"2236082244245389421581\"],\"LastPrices\":[\"2235519201282954769369\"],\"PriceOracleTimestamp\":1793600902,\"MaTime\":\"865\",\"FeeGamma\":\"4961947600000000\",\"MidFee\":\"136000000\",\"OutFee\":\"282000000\",\"LpSupply\":\"286514690749384893175527\",\"XcpProfit\":\"1096995254713366610\",\"VirtualPrice\":\"1054963917514326009\",\"AllowedExtraProfit\":null,\"AdjustmentStep\":\"100000000\",\"UseCustomMath\":true,\"Policy\":\"0xaea95d551154443e73565be724d59d63d72f8a7d\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}","blockNumber":26307080}`

	errStateAfterTrade = "state after a trade not simulated for this pool version"
)

func newPool(t *testing.T, raw string, edit func(*twocryptong.Extra)) *twocryptong.PoolSimulator {
	t.Helper()
	var p entity.Pool
	require.NoError(t, altjson.Unmarshal([]byte(raw), &p))
	if edit != nil {
		var extra twocryptong.Extra
		require.NoError(t, altjson.Unmarshal([]byte(p.Extra), &extra))
		edit(&extra)
		b, err := altjson.Marshal(extra)
		require.NoError(t, err)
		p.Extra = string(b)
	}
	sim, err := twocryptong.NewPoolSimulator(p)
	require.NoError(t, err)
	return sim
}

func quote(sim *twocryptong.PoolSimulator, in, out int, amount string) (*pool.CalcAmountOutResult, error) {
	a, _ := new(big.Int).SetString(amount, 10)
	tokens := sim.GetTokens()
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokens[in], Amount: a},
		TokenOut:      tokens[out],
	})
}

// The first quote on a v3.0.0 pool with a native-fee policy is the chain's get_dy at the same block, to the wei,
// for three BTC pools, both directions, at a thousandth, 1e-5 and 1e-8 of the input reserve.
func TestPoolV3_PreTradeQuoteMatchesChain(t *testing.T) {
	for _, c := range []struct {
		raw     string
		in, out int
		amount  string
		chain   string
	}{
		{poolV3WBTC, 0, 1, "34253607650019964044778", "43531297"},
		{poolV3WBTC, 0, 1, "342536076500199640447", "435413"},
		{poolV3WBTC, 0, 1, "342536076500199640", "436"},
		{poolV3WBTC, 1, 0, "31224663", "23925722939546208647405"},
		{poolV3WBTC, 1, 0, "312246", "239295539779109175824"},
		{poolV3WBTC, 1, 0, "312", "239107404484530853"},
		{poolV3cbBTCFresh, 0, 1, "27756702231009188438738", "35288012"},
		{poolV3cbBTCFresh, 0, 1, "277567022310091884387", "352939"},
		{poolV3cbBTCFresh, 0, 1, "277567022310091884", "353"},
		{poolV3cbBTCFresh, 1, 0, "34993243", "26804907462161353016327"},
		{poolV3cbBTCFresh, 1, 0, "349932", "268092523319356777672"},
		{poolV3cbBTCFresh, 1, 0, "349", "267378933175467140"},
		{poolV3tBTCFresh, 0, 1, "21203026424257775725825", "269566218156893725"},
		{poolV3tBTCFresh, 0, 1, "212030264242577757258", "2696287756601877"},
		{poolV3tBTCFresh, 0, 1, "212030264242577757", "2696294066582"},
		{poolV3tBTCFresh, 1, 0, "191704654592031099", "14683385645485019202552"},
		{poolV3tBTCFresh, 1, 0, "1917046545920310", "146857742062176878965"},
		{poolV3tBTCFresh, 1, 0, "1917046545920", "146857983239426093"},
	} {
		t.Run(fmt.Sprintf("%s %d to %d %s", c.raw[12:22], c.in, c.out, c.amount), func(t *testing.T) {
			res, err := quote(newPool(t, c.raw, nil), c.in, c.out, c.amount)
			require.NoError(t, err)
			assert.Equal(t, c.chain, res.TokenAmountOut.Amount.String())
		})
	}
}

func TestPolicy_StoredByTrackers(t *testing.T) {
	policy := common.HexToAddress("0xe99ee179f1cc62d4a1d3e67dd5541ecb63aca396")
	values := viewValues(packedRebalancingParams(1e8, 1e8, 865))
	srv := poolServer(t, func(method string) any {
		if method == "POLICY" {
			return policy
		}
		return values(method)
	}, "allowed_extra_profit")
	defer srv.Close()
	for name, tr := range trackers(t, srv.URL) {
		t.Run(name, func(t *testing.T) {
			extra, _, _, _ := track(t, tr)
			assert.Equal(t, "0xe99ee179f1cc62d4a1d3e67dd5541ecb63aca396", extra.Policy)
		})
	}
}

// Pools without the view (v2.x) store no policy.
func TestPolicy_NotStoredWithoutTheView(t *testing.T) {
	srv := poolServer(t, viewValues(packedRebalancingParams(1e8, 1e11, 866)), "POLICY")
	defer srv.Close()
	for name, tr := range trackers(t, srv.URL) {
		t.Run(name, func(t *testing.T) {
			extra, got, _, _ := track(t, tr)
			assert.Empty(t, extra.Policy)
			assert.NotContains(t, got.Extra, "Policy")
		})
	}
}

// Quotes stay refused where the first quote is not known to be the chain's: a policy whose fee depends on the
// trade, a policy not audited, no policy read, an A/gamma ramp, the old math, and the exact-out direction.
func TestPoolV3_RefusedWhereTheQuoteIsNotKnown(t *testing.T) {
	for name, c := range map[string]struct {
		raw  string
		edit func(*twocryptong.Extra)
	}{
		"oraclized policy (WETH)": {poolV3WETH, nil},
		"unknown policy":          {poolV3tBTCFresh, func(e *twocryptong.Extra) { e.Policy = "0x1111111111111111111111111111111111111111" }},
		"no policy read":          {poolV3tBTCFresh, func(e *twocryptong.Extra) { e.Policy = "" }},
		"ramp scheduled":          {poolV3tBTCFresh, func(e *twocryptong.Extra) { e.FutureAGammaTime = 4102444800 }},
		"ramp ended":              {poolV3tBTCFresh, func(e *twocryptong.Extra) { e.FutureAGammaTime = 1700000000 }},
		"cryptoswap math":         {poolV3tBTCFresh, func(e *twocryptong.Extra) { e.UseCustomMath = false }},
	} {
		t.Run(name, func(t *testing.T) {
			sim := newPool(t, c.raw, c.edit)
			_, err := quote(sim, 0, 1, "212030264242577757258")
			assert.ErrorIs(t, err, twocryptong.ErrPoolVersionNotSupported)
		})
	}
	t.Run("exact out", func(t *testing.T) {
		sim := newPool(t, poolV3tBTCFresh, nil)
		tokens := sim.GetTokens()
		_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: tokens[1], Amount: big.NewInt(2696287756601877)},
			TokenIn:        tokens[0],
		})
		assert.ErrorIs(t, err, twocryptong.ErrPoolVersionNotSupported)
	})
}

// After UpdateBalance the pool may have rebalanced by v3.0.0's rule, which this package does not run: every
// later quote on that state is refused, in both directions, on the pool and on its clones. The original is
// untouched.
func TestPoolV3_StateAfterTradeIsRefused(t *testing.T) {
	sim := newPool(t, poolV3tBTCFresh, nil)
	res, err := quote(sim, 0, 1, "212030264242577757258")
	require.NoError(t, err)

	traded := sim.CloneState().(*twocryptong.PoolSimulator)
	tokens := sim.GetTokens()
	in, _ := new(big.Int).SetString("212030264242577757258", 10)
	traded.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokens[0], Amount: in},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})

	assert.Equal(t, new(big.Int).Add(sim.GetReserves()[0], in), traded.GetReserves()[0])
	assert.Equal(t, new(big.Int).Sub(sim.GetReserves()[1], res.TokenAmountOut.Amount), traded.GetReserves()[1])
	for _, s := range []*twocryptong.PoolSimulator{traded, traded.CloneState().(*twocryptong.PoolSimulator)} {
		_, err = quote(s, 0, 1, "212030264242577757258")
		assert.EqualError(t, err, errStateAfterTrade)
		_, err = quote(s, 1, 0, "1917046545920310")
		assert.EqualError(t, err, errStateAfterTrade)
	}

	again, err := quote(sim, 0, 1, "212030264242577757258")
	require.NoError(t, err)
	assert.Equal(t, "2696287756601877", again.TokenAmountOut.Amount.String())
}

// The pool reverts a trade that leaves its balances 1000:1 or worse (_assert_balance); the quote refuses it.
func TestPoolV3_RefusesATradeThePoolReverts(t *testing.T) {
	sim := newPool(t, poolV3tBTCFresh, nil)
	in := new(big.Int).Mul(sim.GetReserves()[1], big.NewInt(1000))
	_, err := quote(sim, 1, 0, in.String())
	assert.Error(t, err)
}

// Pools at version v3.0.0 as the pool tracker stores them, at block 26306778. They have no allowed_extra_profit
// view, so the field is null.
const (
	poolV3tBTC  = `{"address":"0x4f52c3a81e33521e5a9a47fd9d3be475d2279c2e","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1793377825,"reserves":["21203026424257775725825133","191704654592031099106"],"tokens":[{"address":"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0x18084fba666a33d37592fa2633fd49a74dd93a88","symbol":"tBTC","decimals":18,"swappable":true}],"extra":"{\"InitialA\":\"50000\",\"InitialGamma\":\"11111111111\",\"FutureA\":\"50000\",\"FutureGamma\":\"11111111111\",\"FutureAGammaTime\":0,\"D\":\"34866851668698472812011087\",\"PriceScale\":[\"71980553859879763736828\"],\"PriceOracle\":[\"77615376390119083331081\"],\"LastPrices\":[\"77615373230567813312479\"],\"FeeGamma\":\"30000000000000000\",\"MidFee\":\"130000000\",\"OutFee\":\"130000000\",\"LpSupply\":\"64345677359056618444004\",\"XcpProfit\":\"1032418260550179630\",\"VirtualPrice\":\"1009847441433321294\",\"AllowedExtraProfit\":null,\"AdjustmentStep\":\"100000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}","blockNumber":26306778}`
	poolV3cbBTC = `{"address":"0x862cb4e988fb66e72f128d1183829f8c05b6c6a0","exchange":"curve-twocrypto-ng","type":"curve-twocrypto-ng","timestamp":1793379686,"reserves":["27756702231009188438738141","34993243729"],"tokens":[{"address":"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e","symbol":"crvUSD","decimals":18,"swappable":true},{"address":"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf","symbol":"cbBTC","decimals":8,"swappable":true}],"extra":"{\"InitialA\":\"50000\",\"InitialGamma\":\"11111111111\",\"FutureA\":\"50000\",\"FutureGamma\":\"11111111111\",\"FutureAGammaTime\":0,\"D\":\"54801030512832732734229143\",\"PriceScale\":[\"77286628838017654881551\"],\"PriceOracle\":[\"77621960826918139510669\"],\"LastPrices\":[\"77621960447248580232137\"],\"FeeGamma\":\"30000000000000000\",\"MidFee\":\"130000000\",\"OutFee\":\"130000000\",\"LpSupply\":\"97097012358894162425702\",\"XcpProfit\":\"1047730358031930301\",\"VirtualPrice\":\"1015080957106381580\",\"AllowedExtraProfit\":null,\"AdjustmentStep\":\"100000000\"}","staticExtra":"{\"IsNativeCoins\":[false,false]}","blockNumber":26306932}`

	mathStableswapV010 = "0x79839c2D74531A8222C0F555865aAc1834e82e51"
	mathStableswapV011 = "0xBfDdF58Cb6ef84e115fF47c10e49A80B2653EA13" // v3.0.0 pools' math
	mathCryptoswapV200 = "0x2005995a71243be9FB995DaB4742327dc76564Df"

	errPoolVersion = "pool version not supported"
)

func TestUseCustomMath(t *testing.T) {
	assert.True(t, twocryptong.UseCustomMath(common.HexToAddress(mathStableswapV010)))
	assert.True(t, twocryptong.UseCustomMath(common.HexToAddress(mathStableswapV011)))
	assert.False(t, twocryptong.UseCustomMath(common.HexToAddress(mathCryptoswapV200)))
}

// Both trackers select the stableswap math for a v3.0.0 pool and keep the missing view as null.
func TestPoolTrackers_V3PoolUsesStableswapMath(t *testing.T) {
	values := viewValues(packedRebalancingParams(1e8, 1e8, 865))
	srv := poolServer(t, func(method string) any {
		if method == "MATH" {
			return common.HexToAddress(mathStableswapV011)
		}
		return values(method)
	}, "allowed_extra_profit")
	defer srv.Close()
	for name, tr := range trackers(t, srv.URL) {
		t.Run(name, func(t *testing.T) {
			extra, _, _, _ := track(t, tr)
			assert.True(t, extra.UseCustomMath)
			assert.Nil(t, extra.AllowedExtraProfit)
		})
	}
}

func newPoolV3(t *testing.T, raw string, customMath bool) *twocryptong.PoolSimulator {
	t.Helper()
	var p entity.Pool
	require.NoError(t, altjson.Unmarshal([]byte(raw), &p))
	if customMath {
		var extra twocryptong.Extra
		require.NoError(t, altjson.Unmarshal([]byte(p.Extra), &extra))
		extra.UseCustomMath = true
		b, err := altjson.Marshal(extra)
		require.NoError(t, err)
		p.Extra = string(b)
	}
	sim, err := twocryptong.NewPoolSimulator(p)
	require.NoError(t, err)
	return sim
}

// A v3.0.0 pool's tweak_price is not this package's: quotes are refused, on the state the pool tracker stores today
// and on the state it stores once the math is selected. Before the refusal the first reached tweak_price and
// dereferenced the null field for a thousandth of the reserves, and the second for every size.
func TestPoolV3_QuotesAreRefusedNotPanicked(t *testing.T) {
	for name, tc := range map[string]struct {
		raw        string
		customMath bool
		divisor    int64
	}{
		"stored state, standard math, reserves/1e3": {poolV3cbBTC, false, 1e3},
		"stored state, standard math, reserves/1e5": {poolV3cbBTC, false, 1e5},
		"stableswap math, reserves/1e3":             {poolV3tBTC, true, 1e3},
		"stableswap math, reserves/1e5":             {poolV3tBTC, true, 1e5},
	} {
		for _, d := range [][2]int{{0, 1}, {1, 0}} {
			t.Run(fmt.Sprintf("%s, %d to %d", name, d[0], d[1]), func(t *testing.T) {
				sim := newPoolV3(t, tc.raw, tc.customMath)
				tokens, reserves := sim.GetTokens(), sim.GetReserves()
				amount := func(i int) *big.Int { return new(big.Int).Div(reserves[i], big.NewInt(tc.divisor)) }

				require.NotPanics(t, func() {
					_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: tokens[d[0]], Amount: amount(d[0])},
						TokenOut:      tokens[d[1]],
					})
					assert.EqualError(t, err, errPoolVersion)

					_, err = sim.CalcAmountIn(pool.CalcAmountInParams{
						TokenAmountOut: pool.TokenAmount{Token: tokens[d[1]], Amount: amount(d[1])},
						TokenIn:        tokens[d[0]],
					})
					assert.EqualError(t, err, errPoolVersion)
				})
			})
		}
	}
}
