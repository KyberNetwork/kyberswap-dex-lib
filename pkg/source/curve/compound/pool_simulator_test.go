package compound

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	// test data from https://etherscan.io/address/0xA2B47E3D5c44877cca798226B7B8118F9BFb7A56#readContract
	// create a foundry test to call to get_dy_underlying, and record the rates/supply/block
	testcases := []struct {
		in                string
		inAmount          string
		out               string
		expectedOutAmount int64
	}{
		{"Bu", "1", "Au", 999157177347},
		{"Au", "100000000000000", "Bu", 100},
	}

	precisionA := big.NewInt(1)             // DAI
	precisionB := big.NewInt(1000000000000) // USDC

	// we cannot use the rate from factory as is (it's just exchangeRateStored, without supplyRatePerBlock... like in actual contract)
	// so manually calculate the rates instead
	curBlock := big.NewInt(17484284)
	rateStoredA, _ := new(big.Int).SetString("b839d9be811a1fd7f6ad81", 16)
	supplyRateA, _ := new(big.Int).SetString("1393db059", 16)
	oldBlockA, _ := new(big.Int).SetString("10ac9ba", 16)
	rateStoredB, _ := new(big.Int).SetString("d02a08ebd736", 16)
	supplyRateB, _ := new(big.Int).SetString("2292c55b6", 16)
	oldBlockB, _ := new(big.Int).SetString("010ac9ea", 16)

	storedRateA := new(big.Int).Add(rateStoredA,
		new(big.Int).Div(
			new(big.Int).Mul(new(big.Int).Mul(rateStoredA, supplyRateA), new(big.Int).Sub(curBlock, oldBlockA)),
			bignumber.BONE,
		),
	)
	storedRateB :=
		new(big.Int).Add(rateStoredB,
			new(big.Int).Div(
				new(big.Int).Mul(new(big.Int).Mul(rateStoredB, supplyRateB), new(big.Int).Sub(curBlock, oldBlockB)),
				bignumber.BONE,
			),
		)

	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"6821027635846033", "21272421810258792"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"a\": \"%v\", \"rates\": [\"%v\", \"%v\"]}",
			"4000000",
			"5000000000",
			4500,
			storedRateA.String(), storedRateB.String(),
		),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"underlyingTokens\": [\"%v\", \"%v\"]}",
			precisionA.String(), precisionB.String(),
			"Au", "Bu"),
	})
	require.Nil(t, err)
	assert.Equal(t, []string{"Bu"}, p.CanSwapTo("Au"))
	assert.Equal(t, []string{"Au"}, p.CanSwapTo("Bu"))
	assert.Equal(t, 0, len(p.CanSwapTo("A")))
	assert.Equal(t, 0, len(p.CanSwapTo("LP")))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)},
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

func TestCalcAmountIn(t *testing.T) {
	pools := []string{
		// https://etherscan.io/address/0xA2B47E3D5c44877cca798226B7B8118F9BFb7A56#readContract
		// block: 19830916
		"{\"address\":\"0xa2b47e3d5c44877cca798226b7b8118f9bfb7a56\",\"reserveUsd\":1028727.8013863643,\"amplifiedTvl\":1028727.8013863643,\"exchange\":\"curve\",\"type\":\"curve-compound\",\"timestamp\":1715238785,\"reserves\":[\"2227675983821834\",\"2139162891206994\"],\"tokens\":[{\"address\":\"0x5d3a536e4d6dbd6114cc1ead35777bab948e3643\",\"name\":\"\",\"symbol\":\"\",\"decimals\":0,\"weight\":1,\"swappable\":true},{\"address\":\"0x39aa39c021dfbae8fac545936693ac917d5e7563\",\"name\":\"\",\"symbol\":\"\",\"decimals\":0,\"weight\":1,\"swappable\":true}],\"extra\":\"{\\\"a\\\":\\\"4500\\\",\\\"swapFee\\\":\\\"4000000\\\",\\\"adminFee\\\":\\\"5000000000\\\",\\\"rates\\\":[\\\"232745748058708534419750515\\\",\\\"238684392278386\\\"]}\",\"staticExtra\":\"{\\\"lpToken\\\":\\\"0x845838df265dcd2c412a1dc9e959c7d08537f8a2\\\",\\\"underlyingTokens\\\":[\\\"0x6b175474e89094c44da98b954eedeac495271d0f\\\",\\\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\\\"],\\\"precisionMultipliers\\\":[\\\"1\\\",\\\"1000000000000\\\"]}\"}",
	}

	testcases := []struct {
		poolIdx          int
		tokenIn          string
		tokenOut         string
		amountOut        *big.Int
		expectedAmountIn *big.Int
		expectedFee      *big.Int
	}{
		// ? USDC -> 1000 DAI
		{
			0,
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"0x6b175474e89094c44da98b954eedeac495271d0f",
			bignumber.NewBig10("1000000000000000000000"),
			bignumber.NewBig10("1000397180"),
			bignumber.NewBig10("400160064025610244"),
		},

		// ? USDC -> 10000 DAI
		{
			0,
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"0x6b175474e89094c44da98b954eedeac495271d0f",
			bignumber.NewBig10("10000000000000000000000"),
			bignumber.NewBig10("10004010702"),
			bignumber.NewBig10("4001600640256102440"),
		},

		// ? DAI -> 1000 USDC
		{
			0,
			"0x6b175474e89094c44da98b954eedeac495271d0f",
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			bignumber.NewBig10("1000000000"),
			bignumber.NewBig10("1000404004115773636535"),
			bignumber.NewBig10("400160"),
		},

		// ? DAI -> 10000 USDC
		{
			0,
			"0x6b175474e89094c44da98b954eedeac495271d0f",
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			bignumber.NewBig10("10000000000"),
			bignumber.NewBig10("10004078989796075967607"),
			bignumber.NewBig10("4001600"),
		},
	}

	sims := lo.Map(pools, func(poolRedis string, _ int) *CompoundPool {
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
			amountIn, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
				return p.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{Token: tc.tokenOut, Amount: tc.amountOut},
					TokenIn:        tc.tokenIn,
					Limit:          nil,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, tc.tokenIn, amountIn.TokenAmountIn.Token)
			assert.Equalf(t, tc.expectedAmountIn, amountIn.TokenAmountIn.Amount, "expected: %v, got: %v", tc.expectedAmountIn.String(), amountIn.TokenAmountIn.Amount.String())
			assert.Equalf(t, tc.expectedFee, amountIn.Fee.Amount, "expected: %v, got: %v", tc.expectedFee.String(), amountIn.Fee.Amount.String())
		})
	}
}
