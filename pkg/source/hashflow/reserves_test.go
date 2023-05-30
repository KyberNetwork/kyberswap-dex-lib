package hashflow

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/stretchr/testify/assert"
)

func newFloat(s string) (res *big.Float) {
	res, _ = new(big.Float).SetString(s)
	return res
}

func Test_calcReserves(t *testing.T) {
	testCases := []struct {
		name             string
		pair             Pair
		expectedReserves entity.PoolReserves
	}{
		{
			name: "it should return correct reserves",
			pair: Pair{
				Decimals: []uint8{6, 18},
				ZeroToOnePriceLevels: []PriceLevel{
					{
						Level: newFloat("30"),
						Price: newFloat("0.000643150720244476791637167068"),
					},
					{
						Level: newFloat("128361.3813982991"),
						Price: newFloat("0.00064286527190387935445231582"),
					},
					{
						Level: newFloat("269558.9009364282"),
						Price: newFloat("0.00064283342369076909052216795"),
					},
					{
						Level: newFloat("424876.1724283701"),
						Price: newFloat("0.000642789719751000283506914279"),
					},
					{
						Level: newFloat("595725.1710695063"),
						Price: newFloat("0.000642736477419145321374194246"),
					},
					{
						Level: newFloat("783659.0695747561"),
						Price: newFloat("0.000642695165970368401629764232"),
					},
					{
						Level: newFloat("990386"),
						Price: newFloat("0.000642649898178725424886803541"),
					},
				},
				OneToZeroPriceLevels: []PriceLevel{
					{
						Level: newFloat("0.0192210971"),
						Price: newFloat("1551.372263451858543703565374016762"),
					},
					{
						Level: newFloat("33.4829891049"),
						Price: newFloat("1551.37204978992804171866737306118"),
					},
					{
						Level: newFloat("70.3142771203"),
						Price: newFloat("1550.751090858661200400092639029026"),
					},
					{
						Level: newFloat("110.8286939372"),
						Price: newFloat("1550.750838235828041433705948293209"),
					},
					{
						Level: newFloat("155.3945524358"),
						Price: newFloat("1550.750090661821332105319015681744"),
					},
					{
						Level: newFloat("204.4169967842"),
						Price: newFloat("1550.713741700079253860167227685452"),
					},
					{
						Level: newFloat("258"),
						Price: newFloat("1550.707001046811683409032411873341"),
					},
				},
			},
			expectedReserves: entity.PoolReserves{"400110324785", "636570045671226045056"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reserves := calcReserves(tc.pair)

			assert.Equal(t, tc.expectedReserves, reserves)
		})
	}
}
