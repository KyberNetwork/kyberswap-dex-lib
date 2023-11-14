package tricrypto

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestCalcAmountOut(t *testing.T) {
	// test data from https://etherscan.io/address/0xd51a44d3fae010294c616388b506acda1bfaae46#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 1, "C", 600295188},
		{"B", 1, "C", 153286543005},
		{"B", 1, "A", 255},
		{"A", 1000, "B", 3},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		Reserves:    entity.PoolReserves{"54743954382801", "212871488312", "32759437840549558629494"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra:       "{\"A\":\"1707629\",\"D\":\"162458225493710120387117207\",\"gamma\":\"11809167828997\",\"priceScale\":[\"25182439404844022315525\",\"1651754874918630176109\",\"\"],\"lastPrices\":[\"25550848343816062635020\",\"1663587698754935470890\",\"\"],\"priceOracle\":[\"25509537194730788716548\",\"1663683592023356857621\",\"\"],\"feeGamma\":\"500000000000000\",\"midFee\":\"3000000\",\"outFee\":\"30000000\",\"futureAGammaTime\":0,\"futureAGamma\":\"581076037942835227425498917514114728328226821\",\"initialAGammaTime\":1633548703,\"initialAGamma\":\"183752478137306770270222288013175834186240000\",\"lastPricesTimestamp\":1686880115,\"lpSupply\":\"151463393077555004737648\",\"xcpProfit\":\"1063768763992698993\",\"virtualPrice\":\"1031885802695565056\",\"allowedExtraProfit\":\"2000000000000\",\"adjustmentStep\":\"490000000000000\",\"maHalfTime\":\"600\"}",
		StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1000000000000\",\"10000000000\",\"1\"]}",
	})
	require.Nil(t, err)

	assert.Equal(t, []string{}, p.CanSwapTo("LP"))
	assert.Equal(t, []string{"B", "C"}, p.CanSwapTo("A"))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := p.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)},
				TokenOut:      tc.out,
				Limit:         nil,
			})
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}

func TestUpdateBalance(t *testing.T) {
	// test data from https://etherscan.io/address/0xd51a44d3fae010294c616388b506acda1bfaae46#readContract
	// use foundry test to call `exchange` and record updated balance
	testcases := []struct {
		in               string
		inAmount         int64
		out              string
		expectedBalances []string
	}{
		{"B", 1, "A", []string{"54622071905365", "212612125597", "32702943198449356152968"}},
		{"B", 1, "C", []string{"54622071905365", "212612125598", "32702943198296147252136"}},
		{"A", 1, "C", []string{"54622071905366", "212612125598", "32702943198295546556685"}},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange:    "",
		Type:        "",
		Reserves:    entity.PoolReserves{"54622071905620", "212612125596", "32702943198449356152968"},
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
		Extra:       "{\"A\":\"1707629\",\"D\":\"162178081891452839666627043\",\"gamma\":\"11809167828997\",\"priceScale\":[\"25182439404844022315525\",\"1651754874918630176109\",\"\"],\"lastPrices\":[\"25500942865479281498021\",\"1663587698754935470890\",\"\"],\"priceOracle\":[\"25539624777171725534648\",\"1663613751394784561740\",\"\"],\"feeGamma\":\"500000000000000\",\"midFee\":\"3000000\",\"outFee\":\"30000000\",\"futureAGammaTime\":0,\"futureAGamma\":\"581076037942835227425498917514114728328226821\",\"initialAGammaTime\":1633548703,\"initialAGamma\":\"183752478137306770270222288013175834186240000\",\"lastPricesTimestamp\":1686881243,\"lpSupply\":\"151202189871784267102739\",\"xcpProfit\":\"1063768898620289638\",\"virtualPrice\":\"1031885933288137559\",\"allowedExtraProfit\":\"2000000000000\",\"adjustmentStep\":\"490000000000000\",\"maHalfTime\":\"600\"}",
		StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1000000000000\",\"10000000000\",\"1\"]}",
	})
	require.Nil(t, err)
	assert.Equal(t, []string{"B", "C"}, p.CanSwapTo("A"))
	assert.Equal(t, 0, len(p.CanSwapTo("LP")))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			amountIn := pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}
			out, err := p.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: amountIn,
				TokenOut:      tc.out,
			})
			require.Nil(t, err)

			fmt.Println(out.TokenAmountOut)
			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  amountIn,
				TokenAmountOut: *out.TokenAmountOut,
				Fee:            *out.Fee,
				SwapInfo:       out.SwapInfo,
			})

			fmt.Println(p.Info.Reserves)
			for i, expBalance := range tc.expectedBalances {
				assert.Equal(t, p.Info.Reserves[i], bignumber.NewBig10(expBalance))
			}
		})
	}
}
