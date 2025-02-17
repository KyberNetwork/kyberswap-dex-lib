package ramsesv2

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	// Tx simulate: https://www.tdly.co/shared/simulation/30202958-4fb6-4144-bda4-4099eea6be11
	token0 := "0x912ce59144191c1204e64559fe8253a0e49e6548"
	token1 := "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8"

	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{token0, 1000000000000000000, token1, 1172208},
	}
	p, err := NewPoolSimulator(
		entity.Pool{
			Exchange: "ramses-v2",
			Type:     "ramses-v2",
			SwapFee:  500,
			Reserves: entity.PoolReserves{"269329183753846211200", "526169379"},
			Tokens:   []*entity.PoolToken{{Address: token0, Decimals: 18}, {Address: token1, Decimals: 6}},
			Extra:    "{\"liquidity\":4360306776077439,\"sqrtPriceX96\":85811322860530180084948,\"feeTier\":500,\"tickSpacing\":10,\"tick\":-274728,\"ticks\":[{\"index\":-283380,\"liquidityGross\":17166285019404,\"liquidityNet\":17166285019404},{\"index\":-279780,\"liquidityGross\":977381896105089,\"liquidityNet\":977381896105089},{\"index\":-278630,\"liquidityGross\":157248791282830,\"liquidityNet\":157248791282830},{\"index\":-278550,\"liquidityGross\":7351763429974,\"liquidityNet\":7351763429974},{\"index\":-276800,\"liquidityGross\":380989062434636,\"liquidityNet\":380989062434636},{\"index\":-276680,\"liquidityGross\":1196219220219038,\"liquidityNet\":1196219220219038},{\"index\":-276330,\"liquidityGross\":7351763429974,\"liquidityNet\":-7351763429974},{\"index\":-276170,\"liquidityGross\":294632869974088,\"liquidityNet\":294632869974088},{\"index\":-276070,\"liquidityGross\":826497613613152,\"liquidityNet\":826497613613152},{\"index\":-275100,\"liquidityGross\":510171037429202,\"liquidityNet\":510171037429202},{\"index\":-274550,\"liquidityGross\":157248791282830,\"liquidityNet\":-157248791282830},{\"index\":-274500,\"liquidityGross\":510171037429202,\"liquidityNet\":-510171037429202},{\"index\":-274170,\"liquidityGross\":1196219220219038,\"liquidityNet\":-1196219220219038},{\"index\":-274030,\"liquidityGross\":294632869974088,\"liquidityNet\":-294632869974088},{\"index\":-273320,\"liquidityGross\":826497613613152,\"liquidityNet\":-826497613613152},{\"index\":-272280,\"liquidityGross\":380989062434636,\"liquidityNet\":-380989062434636},{\"index\":-271750,\"liquidityGross\":977381896105089,\"liquidityNet\":-977381896105089},{\"index\":-269510,\"liquidityGross\":17166285019404,\"liquidityNet\":-17166285019404}]}",
		}, 1)
	require.Nil(t, err)

	assert.Equal(t, []string{token1}, p.CanSwapTo(token0))
	assert.Equal(t, []string{token0}, p.CanSwapTo(token1))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			in := pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
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

// 17/02/2025 : https://dashboard.tenderly.co/tenderly_kyber/linh/simulator/41caabe3-9852-44b9-a0d8-917dbcbab064 (fixed)
func TestPoolSimulator_CalcAmountOutError(t *testing.T) {
	token0 := "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38"
	token1 := "0x674a430f531847a6f8976a900f8ace765f896a1b"
	amtIn, _ := new(big.Int).SetString("755610569226346150671", 10)

	testcases := []struct {
		in            string
		inAmount      *big.Int
		out           string
		expectedError error
	}{
		{token0, amtIn, token1, ErrInvalidSqrtPrice},
	}
	p, err := NewPoolSimulator(
		entity.Pool{
			Exchange: "shadow-dex",
			Type:     "ramses-v2",
			SwapFee:  10000,
			Reserves: entity.PoolReserves{"3459491664175213073021", "7579973495468464590"},
			Tokens:   []*entity.PoolToken{{Address: token0, Decimals: 18}, {Address: token1, Decimals: 18}},
			Extra:    "{\"liquidity\":7775562727683235800570,\"sqrtPriceX96\":1383888070797649867899831499,\"feeTier\":10000,\"tickSpacing\":100,\"tick\":-80953,\"ticks\":[{\"index\":-83400,\"liquidityGross\":2164530642277561489,\"liquidityNet\":2164530642277561489},{\"index\":-82100,\"liquidityGross\":7773398197040958239081,\"liquidityNet\":7773398197040958239081},{\"index\":-80800,\"liquidityGross\":7775562727683235800570,\"liquidityNet\":-7775562727683235800570},{\"index\":-75700,\"liquidityGross\":10112012916483406268,\"liquidityNet\":10112012916483406268},{\"index\":-74900,\"liquidityGross\":10112012916483406268,\"liquidityNet\":-10112012916483406268}],\"unlocked\":true}",
		}, 1)
	require.Nil(t, err)

	assert.Equal(t, []string{token1}, p.CanSwapTo(token0))
	assert.Equal(t, []string{token0}, p.CanSwapTo(token1))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			in := pool.TokenAmount{Token: tc.in, Amount: tc.inAmount}
			_, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			require.ErrorIs(t, err, tc.expectedError)
		})
	}
}
