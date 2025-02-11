package algebrav1

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	// test data from https://polygonscan.com/address/0xd372b5067fe9cbac932af47406fdb9c64666295b#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
		calcInThreshold   int64
	}{
		{"A", 10, "B", 12418116005823, 10},
		{"B", 100000000000000000, "A", 70148, 1},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":2822091172725,"globalState":{"price":93065132232889433968150957834858946,"tick":279543,"feeZto":2985,"feeOtz":2985,"timepoint_index":65,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-887220,"LiquidityGross":2822091172725,"LiquidityNet":2822091172725},{"Index":273540,"LiquidityGross":116315447200034,"LiquidityNet":116315447200034},{"Index":279120,"LiquidityGross":116315447200034,"LiquidityNet":-116315447200034},{"Index":285480,"LiquidityGross":2822091172725,"LiquidityNet":-2822091172725}],"tickSpacing":60}`,
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				in := pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
					TokenOut:      tc.out,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			threshold := big.NewInt(tc.calcInThreshold)
			approx, err := pool.ApproxAmountIn(p, pool.ApproxAmountInParams{
				ExpectedTokenOut: *out.TokenAmountOut,
				TokenIn:          tc.in,
				MaxLoop:          3,
				Threshold:        threshold,
			})
			require.Nil(t, err)
			diff := new(big.Int).Abs(new(big.Int).Sub(approx.TokenAmountOut.Amount, out.TokenAmountOut.Amount))
			assert.Truef(t, diff.Cmp(threshold) < 0, "ApproxAmountIn not exact enough: %v vs %v",
				approx.TokenAmountOut.Amount, out.TokenAmountOut.Amount)
			fmt.Println("approx", approx.TokenAmountIn.Amount, approx.TokenAmountOut.Amount)
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	_ = logger.SetLogLevel("debug")
	// test data from https://polygonscan.com/address/0xd372b5067fe9cbac932af47406fdb9c64666295b#readContract
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{"A", "10", "B", "12418116005823"},
		{"A", "100", "B", "136593135772329"},
		{"A", "1000", "B", "1374962214882655"},
		{"B", "100000000000000000", "A", "70212"},
		{"B", "10000000000000000", "A", "6796"},
		{"B", "10000000000000000", "A", "6756"},

		{"A", "1000000000000000000", "B", "35998532399555197330"},
		{"B", "100000", "A", "12290889471038619"},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":2822091172725,"globalState":{"price":93065132232889433968150957834858946,"tick":279543,"feeZto":2979,"feeOtz":2979,"timepoint_index":65,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-887220,"LiquidityGross":2822091172725,"LiquidityNet":2822091172725},{"Index":273540,"LiquidityGross":116315447200034,"LiquidityNet":116315447200034},{"Index":279120,"LiquidityGross":116315447200034,"LiquidityNet":-116315447200034},{"Index":285480,"LiquidityGross":2822091172725,"LiquidityNet":-2822091172725}],"tickSpacing":60}`,
	})
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			in := pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)}
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
					TokenOut:      tc.out,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  in,
				TokenAmountOut: *out.TokenAmountOut,
				Fee:            *out.Fee,
				SwapInfo:       out.SwapInfo,
			})
		})
	}
}

func TestPoolSimulator_CalcAmountOut_SPL(t *testing.T) {
	_ = logger.SetLogLevel("debug")
	// test data from https://polygonscan.com/address/0x63aefd3aefeedce0860a5ef21c1af548641620dd#readContract
	testcases := []struct {
		in       string
		inAmount int64
		out      string
	}{
		{"A", 10, "B"},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"10963601168695220226", "357336560175387760"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":0,"globalState":{"price":4295128740,"tick":-887272,"feeZto":1622,"feeOtz":1622,"timepoint_index":2497,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-3420,"LiquidityGross":3425867281055637406,"LiquidityNet":3425867281055637406},{"Index":-1680,"LiquidityGross":54492387444405553633,"LiquidityNet":54492387444405553633},{"Index":-1500,"LiquidityGross":11191922902152224210,"LiquidityNet":11191922902152224210},{"Index":0,"LiquidityGross":2148740956490219135,"LiquidityNet":2148740956490219135},{"Index":60,"LiquidityGross":5964987541425314734,"LiquidityNet":5964987541425314734},{"Index":120,"LiquidityGross":5964987541425314734,"LiquidityNet":-5964987541425314734},{"Index":180,"LiquidityGross":2148740956490219135,"LiquidityNet":-2148740956490219135},{"Index":1200,"LiquidityGross":54492387444405553633,"LiquidityNet":-54492387444405553633},{"Index":1380,"LiquidityGross":11191922902152224210,"LiquidityNet":-11191922902152224210},{"Index":2160,"LiquidityGross":3425867281055637406,"LiquidityNet":-3425867281055637406}],"tickSpacing":60}`,
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			_, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				in := pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
					TokenOut:      tc.out,
				})
			})
			require.NotNil(t, err)
			assert.Contains(t, err.Error(), ErrSPL.Error())
		})
	}
}

func TestPoolSimulator_CalcAmountOut_CommFee(t *testing.T) {
	// test data from https://bscscan.com/address/0x0137a5ba1dfa5d6d9a5896251f3d06b2e6669c3a#readContract
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
		calcInThreshold   int64
	}{
		{"A", "10", "B", "3546", 1},
		{"A", "100", "B", "38618", 1},
		{"A", "1000", "B", "389338", 1},
		{"B", "100000000000000000", "A", "250953133732636", 100},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"4972738711862929441043", "1959593146565760679885786"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":98714460437307995596273,"globalState":{"price":1572768200222810245774927517376,"tick":59768,"feeZto":11076,"feeOtz":11076,"timepoint_index":45,"community_fee_token0":1000,"community_fee_token1":1000,"unlocked":true},"ticks":[{"Index":-887220,"LiquidityGross":98714460437307995596273,"LiquidityNet":98714460437307995596273},{"Index":887220,"LiquidityGross":98714460437307995596273,"LiquidityNet":-98714460437307995596273}],"tickSpacing":60}`,
	})
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				in := pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)}
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
					TokenOut:      tc.out,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			threshold := big.NewInt(tc.calcInThreshold)
			approx, err := pool.ApproxAmountIn(p, pool.ApproxAmountInParams{
				ExpectedTokenOut: *out.TokenAmountOut,
				TokenIn:          tc.in,
				MaxLoop:          3,
				Threshold:        threshold,
			})
			require.Nil(t, err)
			diff := new(big.Int).Abs(new(big.Int).Sub(approx.TokenAmountOut.Amount, out.TokenAmountOut.Amount))
			assert.Truef(t, diff.Cmp(threshold) < 0, "ApproxAmountIn not exact enough: %v vs %v",
				approx.TokenAmountOut.Amount, out.TokenAmountOut.Amount)
			fmt.Println("approx", approx.TokenAmountIn.Amount, approx.TokenAmountOut.Amount)
		})
	}
}

func TestPoolSimulator_CalcAmountOut_v1_9(t *testing.T) {
	// test data from https://ftmscan.com/address/0x2fbb6b6c054ef35f20c91fd29d6579cb3c642195#code
	// tickSpacing=5
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{"A", "10", "B", "3"},
		{"A", "10000000000", "B", "4041064818"},
		{"B", "10000000000", "A", "24373699676"},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"21265875874493991905878", "10344609910613908943698"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":299344339249801237803452,"globalState":{"price":50556054571765543459252266509,"tick":-8986,"feeZto":7550,"feeOtz":7550,"timepoint_index":4,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-23040,"LiquidityGross":18101291400643986804037,"LiquidityNet":18101291400643986804037},{"Index":-9495,"LiquidityGross":281243047849157250999415,"LiquidityNet":281243047849157250999415},{"Index":-8940,"LiquidityGross":281243047849157250999415,"LiquidityNet":-281243047849157250999415},{"Index":16080,"LiquidityGross":18101291400643986804037,"LiquidityNet":-18101291400643986804037}],"tickSpacing":5}`,
	})
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				in := pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)}
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
					TokenOut:      tc.out,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}

func TestPoolSimulator_CalcAmountOut_DirFee(t *testing.T) {
	// test data from https://arbiscan.io/address/0x2f0bcb4a8bd714953eefd5339326ee0ff62c5b62#readContract
	// different fee for 2 directions
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 10000000000, "B", 11273265321},
		{"B", 10000000000, "A", 8843048322},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":954140562773509808028,"globalState":{"price":84125210470736011805469300802,"tick":1199,"feeZto":100,"feeOtz":3000,"timepoint_index":104,"community_fee_token0":150,"community_fee_token1":150,"unlocked":true},"ticks":[{"Index":480,"LiquidityGross":954140562773509808028,"LiquidityNet":954140562773509808028},{"Index":1200,"LiquidityGross":954140562773509808028,"LiquidityNet":-954140562773509808028}],"tickSpacing":60}`,
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				in := pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
					TokenOut:      tc.out,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}

func TestPoolSimulator_UpdateBalance_DirFee(t *testing.T) {
	_ = logger.SetLogLevel("debug")
	// test data from https://arbiscan.io/address/0x2f0bcb4a8bd714953eefd5339326ee0ff62c5b62#readContract
	// different fee for 2 directions
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		{"A", "10", "B", "10"},
		{"A", "100", "B", "111"},
		{"A", "1000", "B", "1126"},
		{"B", "10000000000000", "A", "8843048235909"},
		{"B", "100000000000", "A", "88430481480"},
		{"B", "100000000000", "A", "88430481462"},

		{"A", "100000000000000", "B", "112732642939385"},
		{"B", "100000", "A", "88430"},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    `{"liquidity":954140562773509808028,"globalState":{"price":84125210470736011805469300802,"tick":1199,"feeZto":100,"feeOtz":3000,"timepoint_index":104,"community_fee_token0":150,"community_fee_token1":150,"unlocked":true},"ticks":[{"Index":480,"LiquidityGross":954140562773509808028,"LiquidityNet":954140562773509808028},{"Index":1200,"LiquidityGross":954140562773509808028,"LiquidityNet":-954140562773509808028}],"tickSpacing":60}`,
	})
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			in := pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)}
			out, err := p.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: in,
				TokenOut:      tc.out,
			})
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  in,
				TokenAmountOut: *out.TokenAmountOut,
				Fee:            *out.Fee,
				SwapInfo:       out.SwapInfo,
			})
		})
	}
}

const poolEncoded = `{"address":"0x521aa84ab3fcc4c05cabac24dc3682339887b126","reserveUsd":13330.614158641827,"amplifiedTvl":2.10340308337267e+40,"exchange":"camelot-v3","type":"algebra-v1","timestamp":1732709569,"reserves":["1226299351799797623","9090962928"],"tokens":[{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","name":"Wrapped Ether","symbol":"WETH","decimals":18,"weight":50,"swappable":true},{"address":"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8","name":"USD Coin (Arb1)","symbol":"USDC","decimals":6,"weight":50,"swappable":true}],"extra":"{\"liquidity\":4522972368611078,\"globalState\":{\"price\":4651302444251465498324557,\"tick\":-194869,\"feeZto\":150,\"feeOtz\":150,\"timepoint_index\":46821,\"community_fee_token0\":150,\"community_fee_token1\":150,\"unlocked\":true},\"ticks\":[{\"Index\":-887270,\"LiquidityGross\":240327733778,\"LiquidityNet\":240327733778},{\"Index\":-887220,\"LiquidityGross\":193890264843,\"LiquidityNet\":193890264843},{\"Index\":-276300,\"LiquidityGross\":90136646,\"LiquidityNet\":90136646},{\"Index\":-260220,\"LiquidityGross\":4868294557,\"LiquidityNet\":4868294557},{\"Index\":-237180,\"LiquidityGross\":4868294557,\"LiquidityNet\":-4868294557},{\"Index\":-230280,\"LiquidityGross\":402744,\"LiquidityNet\":402744},{\"Index\":-207420,\"LiquidityGross\":3042346848,\"LiquidityNet\":3042346848},{\"Index\":-207240,\"LiquidityGross\":6426212,\"LiquidityNet\":6426212},{\"Index\":-207000,\"LiquidityGross\":8975278785,\"LiquidityNet\":8975278785},{\"Index\":-206280,\"LiquidityGross\":10151784,\"LiquidityNet\":-2700640},{\"Index\":-204120,\"LiquidityGross\":108124199889,\"LiquidityNet\":108124199889},{\"Index\":-203940,\"LiquidityGross\":17441880,\"LiquidityNet\":17441880},{\"Index\":-203880,\"LiquidityGross\":2215307,\"LiquidityNet\":2215307},{\"Index\":-203820,\"LiquidityGross\":3725572,\"LiquidityNet\":-3725572},{\"Index\":-203400,\"LiquidityGross\":4922900164051,\"LiquidityNet\":4922900164051},{\"Index\":-203280,\"LiquidityGross\":161014625224,\"LiquidityNet\":161014625224},{\"Index\":-203190,\"LiquidityGross\":35980248141351,\"LiquidityNet\":35980248141351},{\"Index\":-203160,\"LiquidityGross\":5364790,\"LiquidityNet\":5364790},{\"Index\":-203100,\"LiquidityGross\":5350121150,\"LiquidityNet\":5350121150},{\"Index\":-202920,\"LiquidityGross\":93905807442,\"LiquidityNet\":93905807442},{\"Index\":-202860,\"LiquidityGross\":793409450303,\"LiquidityNet\":793409450303},{\"Index\":-202680,\"LiquidityGross\":6945605734,\"LiquidityNet\":6945605734},{\"Index\":-202620,\"LiquidityGross\":77086836310381,\"LiquidityNet\":77086836310381},{\"Index\":-202390,\"LiquidityGross\":456732871712,\"LiquidityNet\":456732871712},{\"Index\":-202260,\"LiquidityGross\":560626977054,\"LiquidityNet\":560626977054},{\"Index\":-202190,\"LiquidityGross\":6782803013,\"LiquidityNet\":6782803013},{\"Index\":-202140,\"LiquidityGross\":100656237771,\"LiquidityNet\":100656237771},{\"Index\":-201960,\"LiquidityGross\":526294009345,\"LiquidityNet\":526294009345},{\"Index\":-201900,\"LiquidityGross\":6837146113307,\"LiquidityNet\":6649334498423},{\"Index\":-201720,\"LiquidityGross\":15151967,\"LiquidityNet\":15151967},{\"Index\":-201660,\"LiquidityGross\":366479614062,\"LiquidityNet\":-366384481360},{\"Index\":-201600,\"LiquidityGross\":979967887662,\"LiquidityNet\":979967887662},{\"Index\":-201480,\"LiquidityGross\":64334868829,\"LiquidityNet\":64334868829},{\"Index\":-201420,\"LiquidityGross\":4922900164051,\"LiquidityNet\":-4922900164051},{\"Index\":-201300,\"LiquidityGross\":161014625224,\"LiquidityNet\":-161014625224},{\"Index\":-201180,\"LiquidityGross\":1927044627443,\"LiquidityNet\":1927044627443},{\"Index\":-201120,\"LiquidityGross\":5829057737,\"LiquidityNet\":-5519123869},{\"Index\":-201060,\"LiquidityGross\":1771204416281,\"LiquidityNet\":-1771204416281},{\"Index\":-201000,\"LiquidityGross\":30174366546,\"LiquidityNet\":29864432678},{\"Index\":-200940,\"LiquidityGross\":155516644253,\"LiquidityNet\":-155516644253},{\"Index\":-200820,\"LiquidityGross\":21237270293858,\"LiquidityNet\":19650442800002},{\"Index\":-200780,\"LiquidityGross\":270216131909,\"LiquidityNet\":270216131909},{\"Index\":-200760,\"LiquidityGross\":100670418612,\"LiquidityNet\":-100670418612},{\"Index\":-200700,\"LiquidityGross\":58756210,\"LiquidityNet\":-58756210},{\"Index\":-200690,\"LiquidityGross\":93377647327,\"LiquidityNet\":93377647327},{\"Index\":-200640,\"LiquidityGross\":15151967,\"LiquidityNet\":-15151967},{\"Index\":-200610,\"LiquidityGross\":61790326486,\"LiquidityNet\":61790326486},{\"Index\":-200600,\"LiquidityGross\":671440569708,\"LiquidityNet\":671440569708},{\"Index\":-200580,\"LiquidityGross\":4031237664797,\"LiquidityNet\":3128915853453},{\"Index\":-200520,\"LiquidityGross\":7225072757410,\"LiquidityNet\":7218988063714},{\"Index\":-200460,\"LiquidityGross\":297280335230,\"LiquidityNet\":-297280335230},{\"Index\":-200400,\"LiquidityGross\":25390039554370,\"LiquidityNet\":-24606162694256},{\"Index\":-200380,\"LiquidityGross\":456732871712,\"LiquidityNet\":-456732871712},{\"Index\":-200340,\"LiquidityGross\":3580546599881,\"LiquidityNet\":-3580546599881},{\"Index\":-200250,\"LiquidityGross\":263346641824,\"LiquidityNet\":-263346641824},{\"Index\":-200190,\"LiquidityGross\":742597762380,\"LiquidityNet\":-742597762380},{\"Index\":-200180,\"LiquidityGross\":6782803013,\"LiquidityNet\":-6782803013},{\"Index\":-200080,\"LiquidityGross\":64334868829,\"LiquidityNet\":-64334868829},{\"Index\":-200070,\"LiquidityGross\":232400363202,\"LiquidityNet\":232400363202},{\"Index\":-199990,\"LiquidityGross\":5911385432,\"LiquidityNet\":5911385432},{\"Index\":-199980,\"LiquidityGross\":177438922446,\"LiquidityNet\":-177438922446},{\"Index\":-199920,\"LiquidityGross\":3321944525,\"LiquidityNet\":-3321944525},{\"Index\":-199860,\"LiquidityGross\":6743229116006,\"LiquidityNet\":-6743229116006},{\"Index\":-199850,\"LiquidityGross\":27926241391113,\"LiquidityNet\":27926241391113},{\"Index\":-199640,\"LiquidityGross\":2005779623559,\"LiquidityNet\":2005779623559},{\"Index\":-199620,\"LiquidityGross\":237359227913,\"LiquidityNet\":-237359227913},{\"Index\":-199560,\"LiquidityGross\":145408078311,\"LiquidityNet\":145408078311},{\"Index\":-199410,\"LiquidityGross\":17932941176461,\"LiquidityNet\":17932941176461},{\"Index\":-199380,\"LiquidityGross\":270216131909,\"LiquidityNet\":-270216131909},{\"Index\":-199290,\"LiquidityGross\":93377647327,\"LiquidityNet\":-93377647327},{\"Index\":-199260,\"LiquidityGross\":17441880,\"LiquidityNet\":-17441880},{\"Index\":-199210,\"LiquidityGross\":61790326486,\"LiquidityNet\":-61790326486},{\"Index\":-199200,\"LiquidityGross\":610840300021,\"LiquidityNet\":-610840300021},{\"Index\":-199190,\"LiquidityGross\":14838739693436,\"LiquidityNet\":14717539154062},{\"Index\":-199000,\"LiquidityGross\":391938430057,\"LiquidityNet\":-391938430057},{\"Index\":-198930,\"LiquidityGross\":562074191841,\"LiquidityNet\":562074191841},{\"Index\":-198910,\"LiquidityGross\":2216624927507,\"LiquidityNet\":-2216624927507},{\"Index\":-198900,\"LiquidityGross\":10488912941965,\"LiquidityNet\":10488912941965},{\"Index\":-198890,\"LiquidityGross\":201700754623,\"LiquidityNet\":201700754623},{\"Index\":-198880,\"LiquidityGross\":10545559039439,\"LiquidityNet\":-10432266844491},{\"Index\":-198750,\"LiquidityGross\":55908660423,\"LiquidityNet\":55908660423},{\"Index\":-198670,\"LiquidityGross\":232400363202,\"LiquidityNet\":-232400363202},{\"Index\":-198660,\"LiquidityGross\":6945605734,\"LiquidityNet\":-6945605734},{\"Index\":-198580,\"LiquidityGross\":5911385432,\"LiquidityNet\":-5911385432},{\"Index\":-198560,\"LiquidityGross\":2948292329,\"LiquidityNet\":2948292329},{\"Index\":-198450,\"LiquidityGross\":27926241391113,\"LiquidityNet\":-27926241391113},{\"Index\":-198230,\"LiquidityGross\":2005779623559,\"LiquidityNet\":-2005779623559},{\"Index\":-198160,\"LiquidityGross\":145408078311,\"LiquidityNet\":-145408078311},{\"Index\":-198090,\"LiquidityGross\":16622570417949,\"LiquidityNet\":16622570417949},{\"Index\":-198020,\"LiquidityGross\":7712627913558,\"LiquidityNet\":7712627913558},{\"Index\":-198010,\"LiquidityGross\":17932941176461,\"LiquidityNet\":-17932941176461},{\"Index\":-197880,\"LiquidityGross\":77086366469625,\"LiquidityNet\":-77086366469625},{\"Index\":-197790,\"LiquidityGross\":14778139423749,\"LiquidityNet\":-14778139423749},{\"Index\":-197770,\"LiquidityGross\":1063472,\"LiquidityNet\":1063472},{\"Index\":-197710,\"LiquidityGross\":1740185805334,\"LiquidityNet\":1740185805334},{\"Index\":-197520,\"LiquidityGross\":562074191841,\"LiquidityNet\":-562074191841},{\"Index\":-197480,\"LiquidityGross\":56646097474,\"LiquidityNet\":-56646097474},{\"Index\":-197350,\"LiquidityGross\":55908660423,\"LiquidityNet\":-55908660423},{\"Index\":-196980,\"LiquidityGross\":9100544433,\"LiquidityNet\":-9100544433},{\"Index\":-196620,\"LiquidityGross\":20277175633365,\"LiquidityNet\":4851919806249},{\"Index\":-196370,\"LiquidityGross\":1063472,\"LiquidityNet\":-1063472},{\"Index\":-196300,\"LiquidityGross\":1740185805334,\"LiquidityNet\":-1740185805334},{\"Index\":-196250,\"LiquidityGross\":7879726501924,\"LiquidityNet\":7879726501924},{\"Index\":-195880,\"LiquidityGross\":16622570417949,\"LiquidityNet\":-16622570417949},{\"Index\":-195870,\"LiquidityGross\":7879726501924,\"LiquidityNet\":-7879726501924},{\"Index\":-195840,\"LiquidityGross\":18637591163,\"LiquidityNet\":-18637591163},{\"Index\":-195760,\"LiquidityGross\":1517529329944,\"LiquidityNet\":1517529329944},{\"Index\":-195590,\"LiquidityGross\":1566588175203947,\"LiquidityNet\":1566588175203947},{\"Index\":-195580,\"LiquidityGross\":5129091383,\"LiquidityNet\":5129091383},{\"Index\":-195550,\"LiquidityGross\":139481400077427,\"LiquidityNet\":139481400077427},{\"Index\":-195540,\"LiquidityGross\":5251588159706,\"LiquidityNet\":5251588159706},{\"Index\":-195520,\"LiquidityGross\":179306966,\"LiquidityNet\":179306966},{\"Index\":-195470,\"LiquidityGross\":44627202303,\"LiquidityNet\":44627202303},{\"Index\":-195390,\"LiquidityGross\":18813302595,\"LiquidityNet\":18813302595},{\"Index\":-195370,\"LiquidityGross\":9477258585,\"LiquidityNet\":9477258585},{\"Index\":-195350,\"LiquidityGross\":4781258114715,\"LiquidityNet\":4781258114715},{\"Index\":-195330,\"LiquidityGross\":4998721600,\"LiquidityNet\":4998721600},{\"Index\":-195290,\"LiquidityGross\":2767170495679902,\"LiquidityNet\":2767170495679902},{\"Index\":-195220,\"LiquidityGross\":12564547719807,\"LiquidityNet\":-12564547719807},{\"Index\":-195200,\"LiquidityGross\":1166656500345,\"LiquidityNet\":1166656500345},{\"Index\":-195080,\"LiquidityGross\":223348729364,\"LiquidityNet\":223348729364},{\"Index\":-195060,\"LiquidityGross\":8995228627,\"LiquidityNet\":-8995228627},{\"Index\":-194830,\"LiquidityGross\":1566588175203947,\"LiquidityNet\":-1566588175203947},{\"Index\":-194700,\"LiquidityGross\":108124199889,\"LiquidityNet\":-108124199889},{\"Index\":-194520,\"LiquidityGross\":2767170495679902,\"LiquidityNet\":-2767170495679902},{\"Index\":-194360,\"LiquidityGross\":1517529329944,\"LiquidityNet\":-1517529329944},{\"Index\":-194340,\"LiquidityGross\":139481400077427,\"LiquidityNet\":-139481400077427},{\"Index\":-194180,\"LiquidityGross\":5129091383,\"LiquidityNet\":-5129091383},{\"Index\":-194140,\"LiquidityGross\":5251588159706,\"LiquidityNet\":-5251588159706},{\"Index\":-194110,\"LiquidityGross\":179306966,\"LiquidityNet\":-179306966},{\"Index\":-194070,\"LiquidityGross\":44627202303,\"LiquidityNet\":-44627202303},{\"Index\":-193980,\"LiquidityGross\":18813302595,\"LiquidityNet\":-18813302595},{\"Index\":-193970,\"LiquidityGross\":9477258585,\"LiquidityNet\":-9477258585},{\"Index\":-193950,\"LiquidityGross\":4781258114715,\"LiquidityNet\":-4781258114715},{\"Index\":-193930,\"LiquidityGross\":4998721600,\"LiquidityNet\":-4998721600},{\"Index\":-193920,\"LiquidityGross\":201700754623,\"LiquidityNet\":-201700754623},{\"Index\":-193800,\"LiquidityGross\":1166656500345,\"LiquidityNet\":-1166656500345},{\"Index\":-193570,\"LiquidityGross\":420719034710896,\"LiquidityNet\":420719034710896},{\"Index\":-193230,\"LiquidityGross\":420719034710896,\"LiquidityNet\":-420719034710896},{\"Index\":-193080,\"LiquidityGross\":223348729364,\"LiquidityNet\":-223348729364},{\"Index\":-192370,\"LiquidityGross\":2948292329,\"LiquidityNet\":-2948292329},{\"Index\":-189320,\"LiquidityGross\":35980248141351,\"LiquidityNet\":-35980248141351},{\"Index\":-115140,\"LiquidityGross\":90136646,\"LiquidityNet\":-90136646},{\"Index\":887220,\"LiquidityGross\":193890264843,\"LiquidityNet\":-193890264843},{\"Index\":887270,\"LiquidityGross\":221690142615,\"LiquidityNet\":-221690142615}],\"tickSpacing\":10}"}`

// not really random but should be enough for testing
func RandNumberString(maxLen int) string {
	sLen := rand.Intn(maxLen-1) + 1
	s := make([]rune, sLen)
	for i := 0; i < sLen; i++ {
		var c int
		if i == 0 {
			c = rand.Intn(9) + 1
		} else {
			c = rand.Intn(10)
		}
		s[i] = rune(c + '0')
	}
	return string(s)
}

func TestMultiUse(t *testing.T) {
	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(poolEncoded), poolEntity)
	require.NoError(t, err)

	var poolSim pool.IPoolSimulator
	poolSim, err = NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	cloned := poolSim.CloneState()

	tokenAmountIn := pool.TokenAmount{
		Token:  "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
		Amount: bignumber.NewBig10("1000000000000000000"),
	}
	tokenOut := "0x82af49447d8a07e3bd95bd0d56f35241523fbab1"

	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmountIn,
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	expectedAmountOut := result.TokenAmountOut.Amount.String()

	t.Run("same outputs for same inputs", func(t *testing.T) {
		result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
		require.NoError(t, err)
		require.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmountIn,
		TokenAmountOut: *result.TokenAmountOut,
		SwapInfo:       result.SwapInfo,
	})

	t.Run("different output after update", func(t *testing.T) {
		result, err = poolSim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
		require.NoError(t, err)
		require.NotEqual(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("same output of cloned", func(t *testing.T) {
		result, err = cloned.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})
		require.NoError(t, err)
		require.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})
}
