package plain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOutPlain(t *testing.T) {
	pools := []string{
		// plain3basic: http://etherscan.io/address/0xe7a3b38c39f97e977723bd1239c3470702568e7b
		"{\"address\":\"0xe7a3b38c39f97e977723bd1239c3470702568e7b\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708682750,\"reserves\":[\"103902458912250371998101\",\"96026429950922739854657\",\"90684626303\",\"289489289998600589912023\"],\"tokens\":[{\"address\":\"0xee586e7eaad39207f0549bc65f19e336942c992f\",\"symbol\":\"cEUR\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x1a7e4e63778b4f12a199c062f3efdd288afcbce8\",\"symbol\":\"agEUR\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x1abaea1f7c830bd89acc67ec4af516284b1bc33c\",\"symbol\":\"EURC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"1000\\\",\\\"FutureA\\\":\\\"1000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0xe7A3b38c39F97E977723bd1239C3470702568e7B\\\"}\"}",

		// plain2ethema: https://etherscan.io/address/0x94b17476a93b3262d87b9a326965d1e91f9c13e7#readContract
		"{\"address\":\"0x94b17476a93b3262d87b9a326965d1e91f9c13e7\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708930755,\"reserves\":[\"8189776041162322264444\",\"9661706603857954240258\",\"17827858048153259470189\"],\"tokens\":[{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"symbol\":\"ETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x856c4efb76c1d1ae02e20ceb03a2a6a08b0b8dc3\",\"symbol\":\"OETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"40000\\\",\\\"FutureA\\\":\\\"40000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0x94B17476A93b3262d87B9a326965D1E91f9c13E7\\\"}\"}",

		// plain3balances: https://etherscan.io/address/0xb9446c4Ef5EBE66268dA6700D26f96273DE3d571#code
		"{\"address\":\"0xb9446c4ef5ebe66268da6700d26f96273de3d571\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708930755,\"reserves\":[\"549022857960890312641141\",\"1075362632212\",\"46720010\",\"2069614823685039402821670\"],\"tokens\":[{\"address\":\"0x1a7e4e63778b4f12a199c062f3efdd288afcbce8\",\"symbol\":\"agEUR\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xc581b735a1688071a1746c968e0798d642ede491\",\"symbol\":\"EURT\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xdb25f211ab05b1c97d595516f45794528a807ad8\",\"symbol\":\"EURS\",\"decimals\":2,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"20000\\\",\\\"FutureA\\\":\\\"20000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0xb9446c4Ef5EBE66268dA6700D26f96273DE3d571\\\"}\"}",

		// plain4optimized: https://etherscan.io/address/0xda5b670ccd418a187a3066674a8002adc9356ad1#readContract
		"{\"address\":\"0xda5b670ccd418a187a3066674a8002adc9356ad1\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708930755,\"reserves\":[\"310644979221390280\",\"2806169166643327027\",\"360381510649494878\",\"218999711791367011\",\"3256514088341791400\"],\"tokens\":[{\"address\":\"0xd533a949740bb3306d119cc777fa900ba034cd52\",\"symbol\":\"CRV\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x9d409a0a012cfba9b15f6d4b36ac57a46966ab9a\",\"symbol\":\"yvBOOST\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x62b9c7356a2dc64a1969e19c23e4f579f9810aa7\",\"symbol\":\"cvxCRV\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xd38aeb759891882e78e957c80656572503d8c1b1\",\"symbol\":\"sCRV\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"1000\\\",\\\"FutureA\\\":\\\"1000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0xDa5B670CcD418a187a3066674A8002Adc9356Ad1\\\"}\"}",

		// plain2price: https://etherscan.io/address/0x1539c2461d7432cc114b0903f1824079bfca2c92#readContract
		// the stored_rates change fast, use a script to fetch all test cases together at once
		"{\"address\":\"0x1539c2461d7432cc114b0903f1824079bfca2c92\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708942235,\"reserves\":[\"207488005116042557636229\",\"47921035344869338429831\",\"256666057306386486195311\"],\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"symbol\":\"crvUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x83f20f44975d03b1b09e64809b757c47f942beea\",\"symbol\":\"sDAI\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"150000\\\",\\\"FutureA\\\":\\\"150000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\", \\\"RateMultipliers\\\":[\\\"1000000000000000000\\\", \\\"1057419823498475822\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0x1539c2461d7432cc114b0903f1824079BfCA2C92\\\"}\"}",

		// plain oracle: https://arbiscan.io/address/0x6eb2dc694eb516b16dc9fbc678c60052bbdd7d80#readContract
		"{\"address\":\"0x6eb2dc694eb516b16dc9fbc678c60052bbdd7d80\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1709021551,\"reserves\":[\"171562283322052190070\",\"159666449951883581558\",\"344265475511890460140\"],\"tokens\":[{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"symbol\":\"ETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x5979d7b546e38e414f7e9822514be443a4800529\",\"symbol\":\"wstETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"5000\\\",\\\"FutureA\\\":\\\"5000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1158379174506084879\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0xDbcD16e622c95AcB2650b38eC799f76BFC557a0b\\\",\\\"Oracle\\\":\\\"0xb1552c5e96b312d0bf8b554186f846c40614a540\\\"}\"}",
	}

	testcases := []struct {
		poolIdx           int
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{0, "0xee586e7eaad39207f0549bc65f19e336942c992f", 500000000000000, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 496315310333753},
		{0, "0xee586e7eaad39207f0549bc65f19e336942c992f", 50000000000000000, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 49631528847328147},
		{0, "0xee586e7eaad39207f0549bc65f19e336942c992f", 5000000000000000000, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 4963131024216305126},
		{0, "0xee586e7eaad39207f0549bc65f19e336942c992f", 50000000000000000, "0x1abaea1f7c830bd89acc67ec4af516284b1bc33c", 49364},
		{0, "0xee586e7eaad39207f0549bc65f19e336942c992f", 5000000000000000000, "0x1abaea1f7c830bd89acc67ec4af516284b1bc33c", 4936384},
		{0, "0x1abaea1f7c830bd89acc67ec4af516284b1bc33c", 5000, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 5025079238286081},
		{0, "0x1abaea1f7c830bd89acc67ec4af516284b1bc33c", 500045, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 502552896527853049},

		{1, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", 500000000000000, "0x856c4efb76c1d1ae02e20ceb03a2a6a08b0b8dc3", 500008393783392},
		{1, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", 500000000000000000, "0x856c4efb76c1d1ae02e20ceb03a2a6a08b0b8dc3", 500008321109370981},
		{1, "0x856c4efb76c1d1ae02e20ceb03a2a6a08b0b8dc3", 500000000000000000, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", 499591620339898039},
		{1, "0x856c4efb76c1d1ae02e20ceb03a2a6a08b0b8dc3", 5000000000000000000, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", 4995909662946962499},

		{2, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 500000000000000, "0xc581b735a1688071a1746c968e0798d642ede491", 501},
		{2, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 50000000000000000, "0xc581b735a1688071a1746c968e0798d642ede491", 50169},
		{2, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 5000000000000000000, "0xdb25f211ab05b1c97d595516f45794528a807ad8", 499},
		{2, "0xdb25f211ab05b1c97d595516f45794528a807ad8", 1020, "0xc581b735a1688071a1746c968e0798d642ede491", 10248489},

		// off by 1-2 wei, should be acceptable
		{3, "0xd533a949740bb3306d119cc777fa900ba034cd52", 500000000000000, "0xd38aeb759891882e78e957c80656572503d8c1b1", 395568275467422},
		{3, "0xd533a949740bb3306d119cc777fa900ba034cd52", 50000000000000000, "0xd38aeb759891882e78e957c80656572503d8c1b1", 35467697034453723},
		{3, "0x62b9c7356a2dc64a1969e19c23e4f579f9810aa7", 50000000000000000, "0x9d409a0a012cfba9b15f6d4b36ac57a46966ab9a", 96997844562253156},
		{3, "0x62b9c7356a2dc64a1969e19c23e4f579f9810aa7", 50000000000000123, "0x9d409a0a012cfba9b15f6d4b36ac57a46966ab9a", 96997844562253382},
		{3, "0x62b9c7356a2dc64a1969e19c23e4f579f9810aa7", 50000000000000123, "0xd38aeb759891882e78e957c80656572503d8c1b1", 32813108411779300},

		{4, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", 50000000000000123, "0x83f20f44975d03b1b09e64809b757c47f942beea", 47184327843196292},
		{4, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", 500000000000123, "0x83f20f44975d03b1b09e64809b757c47f942beea", 471843279437909},
		{4, "0x83f20f44975d03b1b09e64809b757c47f942beea", 500000000000123, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", 529730979292097},
		{4, "0x83f20f44975d03b1b09e64809b757c47f942beea", 50000000000000123, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", 52973097809547429},

		// off by 1 wei, should be acceptable
		{5, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", 50000000000000123, "0x5979d7b546e38e414f7e9822514be443a4800529", 43210033023565492},
		{5, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", 500000000000123, "0x5979d7b546e38e414f7e9822514be443a4800529", 432102703851362},
		{5, "0x5979d7b546e38e414f7e9822514be443a4800529", 500000000000123, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", 578103325620533},
		{5, "0x5979d7b546e38e414f7e9822514be443a4800529", 50000000000000123, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", 57809965215424298},
	}

	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(poolRedis), &poolEntity)
		require.Nil(t, err)
		p, err := NewPoolSimulator(poolEntity)
		require.Nil(t, err)
		return p
	})

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			p := sims[tc.poolIdx]
			out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}

func TestCalcAmountOutPlainError(t *testing.T) {
	pools := []string{
		// zero balance: https://arbiscan.io/address/0xedce214e7a52c77914342b072230ac971149eb00#readContract
		`{"address":"0xedce214e7a52c77914342b072230ac971149eb00","exchange":"curve-stable-plain","type":"curve-stable-plain","timestamp":1709178100,"reserves":["0","0","0"],"tokens":[{"address":"0x730d5ab5a375c3a6cdc22a9d3bec1573fdea97d6","symbol":"GDC","decimals":18,"swappable":true},{"address":"0xaf88d065e77c8cc2239327c5edb3a432268e5831","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"10000\",\"FutureA\":\"10000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"4000000\",\"AdminFee\":\"5000000000\"}","staticExtra":"{\"APrecision\":\"100\",\"LpToken\":\"0xedCe214e7a52c77914342B072230ac971149Eb00\"}","blockNumber":185550266}`,

		// skewed balance: https://arbiscan.io/address/0x1c5ffa4fb4907b681c61b8c82b28c4672ceb1974#readContract
		`{"address":"0x1c5ffa4fb4907b681c61b8c82b28c4672ceb1974","reserveUsd":368.49875138508617,"amplifiedTvl":368.49875138508617,"exchange":"curve-stable-plain","type":"curve-stable-plain","timestamp":1709176810,"reserves":["14581731602","7584641092575167553","297354791","63473254","678722435250454329942"],"tokens":[{"address":"0x13780e6d5696dd91454f6d3bbc2616687fea43d0","symbol":"UST","decimals":6,"swappable":true},{"address":"0x17fc002b466eec40dae837fc4be5c67993ddbd6f","symbol":"FRAX","decimals":18,"swappable":true},{"address":"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8","symbol":"USDC.e","decimals":6,"swappable":true},{"address":"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"20000\",\"FutureA\":\"20000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"4000000\",\"AdminFee\":\"5000000000\"}","staticExtra":"{\"APrecision\":\"100\",\"LpToken\":\"0x1C5ffa4FB4907B681c61B8c82b28C4672ceb1974\"}","blockNumber":185545245}`,

		`{"address":"0x7c1aa4989df27970381196d3ef32a7410e3f2748","reserveUsd":0.8184891049392782,"amplifiedTvl":0.8184891049392782,"exchange":"curve-stable-plain","type":"curve-stable-plain","timestamp":1709547325,"reserves":["72225117545986","327323206812225085","106781139290107161","44963556946649101174851397","1001002528959797128391"],"tokens":[{"address":"0x7ceb23fd6bc0add59e62ac25578270cff1b9f619","name":"","symbol":"WETH","decimals":18,"weight":0,"swappable":true},{"address":"0xe0b52e49357fd4daf2c15e02058dce6bc0057db4","name":"","symbol":"agEUR","decimals":18,"weight":0,"swappable":true},{"address":"0x3a58a54c066fdc0f2d55fc9c89f0415c92ebf3c4","name":"","symbol":"stMATIC","decimals":18,"weight":0,"swappable":true},{"address":"0x7d645cbbcade2a130bf1bf0528b8541d32d3f8cf","name":"","symbol":"ALRTO","decimals":18,"weight":0,"swappable":true}],"extra":"{\"InitialA\":\"20000\",\"FutureA\":\"20000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"4000000\",\"AdminFee\":\"5000000000\"}","staticExtra":"{\"APrecision\":\"100\",\"LpToken\":\"0x7C1aa4989DF27970381196D3EF32A7410E3F2748\",\"IsNativeCoin\":[false,false,false,false]}"}`,
	}

	testcases := []struct {
		poolIdx  int
		in       string
		inAmount int64
		out      string
	}{
		{0, "0x730d5ab5a375c3a6cdc22a9d3bec1573fdea97d6", 1000000, "0xaf88d065e77c8cc2239327c5edb3a432268e5831"},
		{0, "0xaf88d065e77c8cc2239327c5edb3a432268e5831", 1000000, "0x730d5ab5a375c3a6cdc22a9d3bec1573fdea97d6"},

		{1, "0x13780e6d5696dd91454f6d3bbc2616687fea43d0", 1000000, "0x17fc002b466eec40dae837fc4be5c67993ddbd6f"},
		{1, "0x13780e6d5696dd91454f6d3bbc2616687fea43d0", 1, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9"},
		{1, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8", 1, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9"},
		{1, "0x17fc002b466eec40dae837fc4be5c67993ddbd6f", 4443317428734351594, "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8"},

		{2, "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619", 1000000000000000000, "0xe0b52e49357fd4daf2c15e02058dce6bc0057db4"},
	}

	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(poolRedis), &poolEntity)
		require.Nil(t, err)
		p, err := NewPoolSimulator(poolEntity)
		require.Nil(t, err)
		return p
	})

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			p := sims[tc.poolIdx]
			out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return pool.CalcAmountOut(p, pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}, tc.out, nil)
			})
			if out != nil && out.TokenAmountOut != nil {
				fmt.Println(out.TokenAmountOut.Amount)
			}
			require.NotNil(t, err)
		})
	}
}

func BenchmarkCalcAmountOut(b *testing.B) {
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "A", Decimals: 18}, {Address: "B", Decimals: 18}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\"}",
			"3000000",    // 0.0003
			"5000000000", // 0.5
			150000, 150000),
		StaticExtra: "{\"lpToken\": \"LP\", \"aPrecision\": \"100\"}",
	})
	require.Nil(b, err)
	ain := big.NewInt(5000)

	for i := 0; i < b.N; i++ {
		_, err := p.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "A", Amount: ain},
			TokenOut:      "B",
			Limit:         nil,
		})
		require.Nil(b, err)
	}
}

// old tests from curve-base (with decimal added)

func TestCalcAmountOut(t *testing.T) {
	// test data from https://etherscan.io/address/0x1005f7406f32a61bd760cfa14accd2737913d546#readContract
	// 	call balances, totalSupply to get pool params
	// 	call get_dy to get amount out
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
		expectedFeeAmount int64
	}{
		{"A", 5000, "B", 4998, 1},
		{"A", 50000, "B", 49986, 15},
		{"B", 50000, "A", 49983, 14},
		{"B", 51000, "A", 50982, 15},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "A", Decimals: 6}, {Address: "B", Decimals: 6}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\"}",
			"3000000",    // 0.0003
			"5000000000", // 0.5
			150000, 150000),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"aPrecision\": \"%v\"}",
			"100"),
	})
	require.Nil(t, err)
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))
	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))
	assert.Equal(t, 0, len(p.CanSwapTo("LP")))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
			assert.Equal(t, big.NewInt(tc.expectedFeeAmount), out.Fee.Amount)
		})
	}
}

func TestCalcAmountOut_interpolate_from_initialA_and_futureA(t *testing.T) {
	// if A is getting ramped up then it should interpolate A correctly
	// 100k at zero to 200k at now*2, so now should be 150k, so the same as the contract above -> get expected output from contract get_dy
	now := time.Now().Unix()
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "A", Decimals: 6}, {Address: "B", Decimals: 6}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\", \"futureATime\": %v}",
			"3000000",    // 0.0003
			"5000000000", // 0.5
			100000, 200000,
			now*2),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"0x0\", \"aPrecision\": \"%v\"}",
			"100"),
	})
	require.Nil(t, err)

	out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
		return p.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "A", Amount: big.NewInt(510000)},
			TokenOut:      "B",
			Limit:         nil,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, big.NewInt(509863), out.TokenAmountOut.Amount)
	assert.Equal(t, big.NewInt(153), out.Fee.Amount)
}

func TestGetDyVirtualPrice(t *testing.T) {
	// test data from https://etherscan.io/address/0x1005f7406f32a61bd760cfa14accd2737913d546#readContract
	testcases := []struct {
		i      int
		j      int
		dx     string
		expOut string
	}{
		{0, 1, "100000000000000", "140254485"},
		{0, 1, "1000000000000", "140254485"},
		{0, 1, "100000000", "99936588"},
		{0, 1, "100002233", "99938816"},
		{1, 0, "20000", "19982"},
		{1, 0, "3000200", "2997456"},
		{1, 0, "88001800", "69067695"},
		{1, 0, "100000000000000", "69244498"},
	}
	poolRedis := `{
		"address": "0x1005f7406f32a61bd760cfa14accd2737913d546",
		"reserveUsd": 209.42969262729198,
		"amplifiedTvl": 209.42969262729198,
		"exchange": "curve",
		"type": "curve-base",
		"timestamp": 1705393976,
		"reserves": ["69265278", "140296574", "208111994100559113335"],
		"tokens": [
			{ "address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "weight": 1, "swappable": true, "decimals": 6 },
			{ "address": "0xdac17f958d2ee523a2206206994597c13d831ec7", "weight": 1, "swappable": true, "decimals": 6 }
		],
		"extra": "{\"initialA\":\"150000\",\"futureA\":\"150000\",\"initialATime\":0,\"futureATime\":0,\"swapFee\":\"3000000\",\"adminFee\":\"5000000000\"}",
		"staticExtra": "{\"lpToken\":\"0x1005f7406f32a61bd760cfa14accd2737913d546\",\"aPrecision\":\"100\"}"
	}`
	var poolEntity entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEntity)
	require.Nil(t, err)
	p, err := NewPoolSimulator(poolEntity)
	require.Nil(t, err)

	v, dCached, err := p.GetVirtualPrice()
	require.Nil(t, err)
	assert.Equal(t, bignumber.NewBig10("1006923185919753102"), v)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			dy, err := testutil.MustConcurrentSafe[*big.Int](t, func() (any, error) {
				dy, _, err := p.GetDy(tc.i, tc.j, bignumber.NewBig10(tc.dx), nil)
				return dy, err
			})
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expOut), dy)

			// test using cached D
			dy, err = testutil.MustConcurrentSafe[*big.Int](t, func() (any, error) {
				dy, _, err := p.GetDy(tc.i, tc.j, bignumber.NewBig10(tc.dx), dCached)
				return dy, err
			})
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expOut), dy)
		})
	}
}

func TestPoolSimulatorPlain_CalcAmountIn(t *testing.T) {
	pools := []string{
		// plain3basic: http://etherscan.io/address/0xe7a3b38c39f97e977723bd1239c3470702568e7b
		"{\"address\":\"0xe7a3b38c39f97e977723bd1239c3470702568e7b\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708682750,\"reserves\":[\"103902458912250371998101\",\"96026429950922739854657\",\"90684626303\",\"289489289998600589912023\"],\"tokens\":[{\"address\":\"0xee586e7eaad39207f0549bc65f19e336942c992f\",\"symbol\":\"cEUR\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x1a7e4e63778b4f12a199c062f3efdd288afcbce8\",\"symbol\":\"agEUR\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x1abaea1f7c830bd89acc67ec4af516284b1bc33c\",\"symbol\":\"EURC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"1000\\\",\\\"FutureA\\\":\\\"1000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0xe7A3b38c39F97E977723bd1239C3470702568e7B\\\"}\"}",

		// plain2ethema: https://etherscan.io/address/0x94b17476a93b3262d87b9a326965d1e91f9c13e7#readContract
		"{\"address\":\"0x94b17476a93b3262d87b9a326965d1e91f9c13e7\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708930755,\"reserves\":[\"8189776041162322264444\",\"9661706603857954240258\",\"17827858048153259470189\"],\"tokens\":[{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"symbol\":\"ETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x856c4efb76c1d1ae02e20ceb03a2a6a08b0b8dc3\",\"symbol\":\"OETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"40000\\\",\\\"FutureA\\\":\\\"40000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0x94B17476A93b3262d87B9a326965D1E91f9c13E7\\\"}\"}",

		// plain3balances: https://etherscan.io/address/0xb9446c4Ef5EBE66268dA6700D26f96273DE3d571#code
		"{\"address\":\"0xb9446c4ef5ebe66268da6700d26f96273de3d571\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708930755,\"reserves\":[\"549022857960890312641141\",\"1075362632212\",\"46720010\",\"2069614823685039402821670\"],\"tokens\":[{\"address\":\"0x1a7e4e63778b4f12a199c062f3efdd288afcbce8\",\"symbol\":\"agEUR\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xc581b735a1688071a1746c968e0798d642ede491\",\"symbol\":\"EURT\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xdb25f211ab05b1c97d595516f45794528a807ad8\",\"symbol\":\"EURS\",\"decimals\":2,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"20000\\\",\\\"FutureA\\\":\\\"20000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0xb9446c4Ef5EBE66268dA6700D26f96273DE3d571\\\"}\"}",

		// plain4optimized: https://etherscan.io/address/0xda5b670ccd418a187a3066674a8002adc9356ad1#readContract
		"{\"address\":\"0xda5b670ccd418a187a3066674a8002adc9356ad1\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708930755,\"reserves\":[\"310644979221390280\",\"2806169166643327027\",\"360381510649494878\",\"218999711791367011\",\"3256514088341791400\"],\"tokens\":[{\"address\":\"0xd533a949740bb3306d119cc777fa900ba034cd52\",\"symbol\":\"CRV\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x9d409a0a012cfba9b15f6d4b36ac57a46966ab9a\",\"symbol\":\"yvBOOST\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x62b9c7356a2dc64a1969e19c23e4f579f9810aa7\",\"symbol\":\"cvxCRV\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xd38aeb759891882e78e957c80656572503d8c1b1\",\"symbol\":\"sCRV\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"1000\\\",\\\"FutureA\\\":\\\"1000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0xDa5B670CcD418a187a3066674A8002Adc9356Ad1\\\"}\"}",

		// plain2price: https://etherscan.io/address/0x1539c2461d7432cc114b0903f1824079bfca2c92#readContract
		// the stored_rates change fast, use a script to fetch all test cases together at once
		"{\"address\":\"0x1539c2461d7432cc114b0903f1824079bfca2c92\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708942235,\"reserves\":[\"207488005116042557636229\",\"47921035344869338429831\",\"256666057306386486195311\"],\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"symbol\":\"crvUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x83f20f44975d03b1b09e64809b757c47f942beea\",\"symbol\":\"sDAI\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"150000\\\",\\\"FutureA\\\":\\\"150000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\", \\\"RateMultipliers\\\":[\\\"1000000000000000000\\\", \\\"1057419823498475822\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0x1539c2461d7432cc114b0903f1824079BfCA2C92\\\"}\"}",

		// plain oracle: https://arbiscan.io/address/0x6eb2dc694eb516b16dc9fbc678c60052bbdd7d80#readContract
		"{\"address\":\"0x6eb2dc694eb516b16dc9fbc678c60052bbdd7d80\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1709021551,\"reserves\":[\"171562283322052190070\",\"159666449951883581558\",\"344265475511890460140\"],\"tokens\":[{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"symbol\":\"ETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x5979d7b546e38e414f7e9822514be443a4800529\",\"symbol\":\"wstETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"5000\\\",\\\"FutureA\\\":\\\"5000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1158379174506084879\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0xDbcD16e622c95AcB2650b38eC799f76BFC557a0b\\\",\\\"Oracle\\\":\\\"0xb1552c5e96b312d0bf8b554186f846c40614a540\\\"}\"}",
	}

	testcases := []struct {
		poolIdx          int
		tokenOut         string
		amountOut        int64
		tokenIn          string
		expectedAmountIn int64
	}{
		{0, "0xee586e7eaad39207f0549bc65f19e336942c992f", 500000000000000, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 496712601382555},
		{0, "0xee586e7eaad39207f0549bc65f19e336942c992f", 50000000000000000, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 49671262326928066},
		{0, "0xee586e7eaad39207f0549bc65f19e336942c992f", 5000000000000000000, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 4967148119371669181},
		{0, "0xee586e7eaad39207f0549bc65f19e336942c992f", 50000000000000000, "0x1abaea1f7c830bd89acc67ec4af516284b1bc33c", 49403},
		{0, "0xee586e7eaad39207f0549bc65f19e336942c992f", 5000000000000000000, "0x1abaea1f7c830bd89acc67ec4af516284b1bc33c", 4940382},
		{0, "0x1abaea1f7c830bd89acc67ec4af516284b1bc33c", 5000, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 5029100961501386},
		{0, "0x1abaea1f7c830bd89acc67ec4af516284b1bc33c", 500045, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 502955593279445770},

		{1, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", 500000000000000, "0x856c4efb76c1d1ae02e20ceb03a2a6a08b0b8dc3", 500408640776154},
		{1, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", 500000000000000000, "0x856c4efb76c1d1ae02e20ceb03a2a6a08b0b8dc3", 500408713540137816},
		{1, "0x856c4efb76c1d1ae02e20ceb03a2a6a08b0b8dc3", 500000000000000000, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", 499991679027897852},
		{1, "0x856c4efb76c1d1ae02e20ceb03a2a6a08b0b8dc3", 5000000000000000000, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", 4999923335976431559},

		{2, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 500000000000000, "0xc581b735a1688071a1746c968e0798d642ede491", 502},
		{2, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 50000000000000000, "0xc581b735a1688071a1746c968e0798d642ede491", 50209},
		{2, "0x1a7e4e63778b4f12a199c062f3efdd288afcbce8", 5000000000000000000, "0xdb25f211ab05b1c97d595516f45794528a807ad8", 499},
		{2, "0xdb25f211ab05b1c97d595516f45794528a807ad8", 1020, "0xc581b735a1688071a1746c968e0798d642ede491", 10252593},

		// off by 1-2 wei, should be acceptable
		{3, "0xd533a949740bb3306d119cc777fa900ba034cd52", 500000000000000, "0xd38aeb759891882e78e957c80656572503d8c1b1", 396795351756956},
		{3, "0xd533a949740bb3306d119cc777fa900ba034cd52", 50000000000000000, "0xd38aeb759891882e78e957c80656572503d8c1b1", 44717955034190673},
		{3, "0x62b9c7356a2dc64a1969e19c23e4f579f9810aa7", 50000000000000000, "0x9d409a0a012cfba9b15f6d4b36ac57a46966ab9a", 110916436270691561},
		{3, "0x62b9c7356a2dc64a1969e19c23e4f579f9810aa7", 50000000000000123, "0x9d409a0a012cfba9b15f6d4b36ac57a46966ab9a", 110916436270691856},
		{3, "0x62b9c7356a2dc64a1969e19c23e4f579f9810aa7", 50000000000000123, "0xd38aeb759891882e78e957c80656572503d8c1b1", 40190891186356188},

		{4, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", 50000000000000123, "0x83f20f44975d03b1b09e64809b757c47f942beea", 47193766327732350},
		{4, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", 500000000000123, "0x83f20f44975d03b1b09e64809b757c47f942beea", 471937662271307},
		{4, "0x83f20f44975d03b1b09e64809b757c47f942beea", 500000000000123, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", 501065829892009},
		{4, "0x83f20f44975d03b1b09e64809b757c47f942beea", 50000000000000123, "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e", 50106583096228911},

		// off by 1 wei, should be acceptable
		{5, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", 50000000000000123, "0x5979d7b546e38e414f7e9822514be443a4800529", 43245101822358613},
		{5, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", 500000000000123, "0x5979d7b546e38e414f7e9822514be443a4800529", 432448641536457},
		{5, "0x5979d7b546e38e414f7e9822514be443a4800529", 500000000000123, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", 499461806509854},
		{5, "0x5979d7b546e38e414f7e9822514be443a4800529", 50000000000000123, "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", 49946454721780546},
	}

	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(poolRedis), &poolEntity)
		require.Nil(t, err)
		p, err := NewPoolSimulator(poolEntity)
		require.Nil(t, err)
		return p
	})

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			p := sims[tc.poolIdx]
			amountIn, err := testutil.MustConcurrentSafe[*pool.CalcAmountInResult](t, func() (any, error) {
				return p.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{Token: tc.tokenOut, Amount: big.NewInt(tc.amountOut)},
					TokenIn:        tc.tokenIn,
					Limit:          nil,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedAmountIn), amountIn.TokenAmountIn.Amount)
			assert.Equal(t, tc.tokenIn, amountIn.TokenAmountIn.Token)
		})
	}
}
