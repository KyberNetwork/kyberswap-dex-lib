package plainoracle

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	// test data from https://optimistic.etherscan.io/address/0xb90b9b1f91a01ea22a182cd84c1e22222e39b415#readContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A", 100000, "B", 88639},
		{"B", 100000, "A", 112726},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"4929038393526761949570", "4622174777771844922336", "9849021650836480441313"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\", \"rates\": [%v, %v]}",
			"4000000",
			"5000000000",
			5000, 5000,
			"1000000000000000000", "1128972205632615487"),
		StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"oracle\": \"%v\"}",
			"100",
			"1", "1",
			"0xe59EBa0D492cA53C6f46015EEa00517F2707dc77"),
	})
	require.Nil(t, err)

	assert.Equal(t, 0, len(p.CanSwapTo("LP")))
	assert.Equal(t, []string{"B"}, p.CanSwapTo("A"))
	assert.Equal(t, []string{"A"}, p.CanSwapTo("B"))

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
		})
	}
}

func TestGetDyVirtualPrice(t *testing.T) {
	// test data from https://optimistic.etherscan.io/address/0xb90b9b1f91a01ea22a182cd84c1e22222e39b415#readContract
	testcases := []struct {
		i      int
		j      int
		dx     string
		expOut string
	}{
		{0, 1, "100000000000000", "86818314418062"},
		{0, 1, "1000000000000", "868183154558"},
		{0, 1, "100000000", "86818316"},
		{0, 1, "100002233", "86820254"},
		{1, 0, "20000", "23018"},
		{1, 0, "3000200", "3452958"},
		{1, 0, "88001800", "101282099"},
		{1, 0, "100000000000000", "115090939134446"},
	}
	poolRedis := `{
		"address": "0xb90b9b1f91a01ea22a182cd84c1e22222e39b415",
		"reserveUsd": 834336.0036396985,
		"amplifiedTvl": 834336.0036396985,
		"exchange": "curve",
		"type": "curve-plain-oracle",
		"timestamp": 1705393864,
		"reserves": ["156463394192707746175", "150781038654005989858", "316970452569291468507"],
		"tokens": [
			{ "address": "0x4200000000000000000000000000000000000006", "weight": 1, "swappable": true },
			{ "address": "0x1f32b1c2345538c0c6f582fcb022739c4a194ebb", "weight": 1, "swappable": true }
		],
		"extra": "{\"rates\":[1000000000000000000,1153777372655731291],\"initialA\":\"5000\",\"futureA\":\"5000\",\"initialATime\":0,\"futureATime\":0,\"swapFee\":\"4000000\",\"adminFee\":\"5000000000\"}",
		"staticExtra": "{\"lpToken\":\"0xefde221f306152971d8e9f181bfe998447975810\",\"aPrecision\":\"100\",\"precisionMultipliers\":[\"1\",\"1\"],\"oracle\":\"0xe59EBa0D492cA53C6f46015EEa00517F2707dc77\"}"
	}
	`
	var poolEntity entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEntity)
	require.Nil(t, err)
	p, err := NewPoolSimulator(poolEntity)
	require.Nil(t, err)

	v, dCached, err := p.GetVirtualPrice()
	require.Nil(t, err)
	assert.Equal(t, bignumber.NewBig10("1042437950645007280"), v)

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
