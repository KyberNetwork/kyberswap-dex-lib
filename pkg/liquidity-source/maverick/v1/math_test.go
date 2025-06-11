package maverickv1

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestSwapAForBWithoutExactOut(t *testing.T) {
	t.Parallel()
	var bins = map[uint32]Bin{
		1: {
			ReserveA:  bignumber.NewUint256("497483862887020288"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      2,
			LowerTick: -8,
		},
		2: {
			ReserveA:  bignumber.NewUint256("497483862887020288"),
			ReserveB:  bignumber.NewUint256("601955474093294592000000000000"),
			Kind:      2,
			LowerTick: 1,
		},
		3: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("601955474093294592000000000000"),
			Kind:      2,
			LowerTick: 4,
		},
		4: {
			ReserveA:  bignumber.NewUint256("204096294304391520"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      0,
			LowerTick: -7,
		},
		5: {
			ReserveA:  bignumber.NewUint256("988635599394593504"),
			ReserveB:  bignumber.NewUint256("1196249075267458226326064162896"),
			Kind:      0,
			LowerTick: 1,
		},
		6: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("246956516108313792000000000000"),
			Kind:      0,
			LowerTick: 6,
		},
		7: {
			ReserveA:  bignumber.NewUint256("784539305090201984"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      0,
			LowerTick: -1,
		},
		8: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("949292559159144576000000000000"),
			Kind:      0,
			LowerTick: 7,
		},
		9: {
			ReserveA:  bignumber.NewUint256("242248889019272896"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      1,
			LowerTick: -9,
		},
		10: {
			ReserveA:  bignumber.NewUint256("340606509825846240"),
			ReserveB:  bignumber.NewUint256("412133876889273980196333938548"),
			Kind:      1,
			LowerTick: 1,
		},
		11: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("293121155713320256000000000000"),
			Kind:      1,
			LowerTick: 9,
		},
		12: {
			ReserveA:  bignumber.NewUint256("401225937191387328"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      3,
			LowerTick: -3,
		},
		13: {
			ReserveA:  bignumber.NewUint256("401225937191387316"),
			ReserveB:  bignumber.NewUint256("485483384001578688000000000000"),
			Kind:      3,
			LowerTick: 1,
		},
		14: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("485483384001578688000000000000"),
			Kind:      3,
			LowerTick: 4,
		},
		15: {
			ReserveA:  bignumber.NewUint256("98357620806573344"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      1,
			LowerTick: -8,
		},
		16: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("119012721175953760000000000000"),
			Kind:      1,
			LowerTick: 7,
		},
	}
	var binPositions = map[int32]map[uint8]uint32{
		1: {
			0: 5,
			1: 10,
			2: 2,
			3: 13,
		},
		4: {
			2: 3,
			3: 14,
		},
		6: {
			0: 6,
		},
		7: {
			0: 8,
			1: 16,
		},
		9: {
			1: 11,
		},
		-8: {
			1: 15,
			2: 1,
		},
		-7: {
			0: 4,
		},
		-1: {
			0: 7,
		},
		-9: {
			1: 9,
		},
		-3: {
			3: 12,
		},
	}

	var binMap = map[int16]*uint256.Int{
		0:  uint256.MustFromDecimal("138261823728"),
		-1: uint256.MustFromDecimal("7463162598112715418867754100145796611164620634624434827815830738677402697728"),
	}

	var state = &MaverickPoolState{
		Bins:             bins,
		TickSpacing:      953,
		Fee:              uint256.NewInt(uint64((0.3 / 100) * 1e18)),
		ActiveTick:       1,
		ProtocolFeeRatio: uint256.NewInt(0),
		BinPositions:     binPositions,
		BinMap:           binMap,
		minBinMapIndex:   -1,
		maxBinMapIndex:   0,
	}

	var amountIn = bignumber.NewUint256("1850163333337788672")
	_, amountOut, _, err := swap(state, amountIn, true, false, false)

	assert.Nil(t, err)
	assert.Equal(t, "1676945827577881677", amountOut.String())
}

func TestSwapAForBExactOut(t *testing.T) {
	t.Parallel()
	var bins = map[uint32]Bin{
		1: {
			ReserveA:  bignumber.NewUint256("497483862887020288"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      2,
			LowerTick: -8,
		},
		2: {
			ReserveA:  bignumber.NewUint256("497483862887020288"),
			ReserveB:  bignumber.NewUint256("601955474093294592000000000000"),
			Kind:      2,
			LowerTick: 1,
		},
		3: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("601955474093294592000000000000"),
			Kind:      2,
			LowerTick: 4,
		},
		4: {
			ReserveA:  bignumber.NewUint256("204096294304391520"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      0,
			LowerTick: -7,
		},
		5: {
			ReserveA:  bignumber.NewUint256("988635599394593504"),
			ReserveB:  bignumber.NewUint256("1196249075267458226326064162896"),
			Kind:      0,
			LowerTick: 1,
		},
		6: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("246956516108313792000000000000"),
			Kind:      0,
			LowerTick: 6,
		},
		7: {
			ReserveA:  bignumber.NewUint256("784539305090201984"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      0,
			LowerTick: -1,
		},
		8: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("949292559159144576000000000000"),
			Kind:      0,
			LowerTick: 7,
		},
		9: {
			ReserveA:  bignumber.NewUint256("242248889019272896"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      1,
			LowerTick: -9,
		},
		10: {
			ReserveA:  bignumber.NewUint256("340606509825846240"),
			ReserveB:  bignumber.NewUint256("412133876889273980196333938548"),
			Kind:      1,
			LowerTick: 1,
		},
		11: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("293121155713320256000000000000"),
			Kind:      1,
			LowerTick: 9,
		},
		12: {
			ReserveA:  bignumber.NewUint256("401225937191387328"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      3,
			LowerTick: -3,
		},
		13: {
			ReserveA:  bignumber.NewUint256("401225937191387316"),
			ReserveB:  bignumber.NewUint256("485483384001578688000000000000"),
			Kind:      3,
			LowerTick: 1,
		},
		14: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("485483384001578688000000000000"),
			Kind:      3,
			LowerTick: 4,
		},
		15: {
			ReserveA:  bignumber.NewUint256("98357620806573344"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      1,
			LowerTick: -8,
		},
		16: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("119012721175953760000000000000"),
			Kind:      1,
			LowerTick: 7,
		},
	}
	var binPositions = map[int32]map[uint8]uint32{
		1: {
			0: 5,
			1: 10,
			2: 2,
			3: 13,
		},
		4: {
			2: 3,
			3: 14,
		},
		6: {
			0: 6,
		},
		7: {
			0: 8,
			1: 16,
		},
		9: {
			1: 11,
		},
		-8: {
			1: 15,
			2: 1,
		},
		-7: {
			0: 4,
		},
		-1: {
			0: 7,
		},
		-9: {
			1: 9,
		},
		-3: {
			3: 12,
		},
	}

	var binMap = map[int16]*uint256.Int{
		0:  uint256.MustFromDecimal("138261823728"),
		-1: uint256.MustFromDecimal("7463162598112715418867754100145796611164620634624434827815830738677402697728"),
	}

	var state = &MaverickPoolState{
		Bins:             bins,
		TickSpacing:      953,
		Fee:              uint256.NewInt(uint64((0.3 / 100) * 1e18)),
		ActiveTick:       1,
		ProtocolFeeRatio: uint256.NewInt(0),
		BinPositions:     binPositions,
		BinMap:           binMap,
		minBinMapIndex:   -1,
		maxBinMapIndex:   0,
	}

	var amountOut = bignumber.NewUint256("2963297000000000000")
	amountIn, _, _, err := swap(state, amountOut, true, true, false)

	assert.Nil(t, err)
	assert.Equal(t, "3269386145352608663", amountIn.String())
}

func TestSwapBForAExactOut(t *testing.T) {
	t.Parallel()
	var bins = map[uint32]Bin{
		1: {
			ReserveA:  bignumber.NewUint256("497483862887020288"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      2,
			LowerTick: -8,
		},
		2: {
			ReserveA:  bignumber.NewUint256("497483862887020288"),
			ReserveB:  bignumber.NewUint256("601955474093294592000000000000"),
			Kind:      2,
			LowerTick: 1,
		},
		3: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("601955474093294592000000000000"),
			Kind:      2,
			LowerTick: 4,
		},
		4: {
			ReserveA:  bignumber.NewUint256("204096294304391520"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      0,
			LowerTick: -7,
		},
		5: {
			ReserveA:  bignumber.NewUint256("988635599394593504"),
			ReserveB:  bignumber.NewUint256("1196249075267458226326064162896"),
			Kind:      0,
			LowerTick: 1,
		},
		6: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("246956516108313792000000000000"),
			Kind:      0,
			LowerTick: 6,
		},
		7: {
			ReserveA:  bignumber.NewUint256("784539305090201984"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      0,
			LowerTick: -1,
		},
		8: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("949292559159144576000000000000"),
			Kind:      0,
			LowerTick: 7,
		},
		9: {
			ReserveA:  bignumber.NewUint256("242248889019272896"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      1,
			LowerTick: -9,
		},
		10: {
			ReserveA:  bignumber.NewUint256("340606509825846240"),
			ReserveB:  bignumber.NewUint256("412133876889273980196333938548"),
			Kind:      1,
			LowerTick: 1,
		},
		11: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("293121155713320256000000000000"),
			Kind:      1,
			LowerTick: 9,
		},
		12: {
			ReserveA:  bignumber.NewUint256("401225937191387328"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      3,
			LowerTick: -3,
		},
		13: {
			ReserveA:  bignumber.NewUint256("401225937191387316"),
			ReserveB:  bignumber.NewUint256("485483384001578688000000000000"),
			Kind:      3,
			LowerTick: 1,
		},
		14: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("485483384001578688000000000000"),
			Kind:      3,
			LowerTick: 4,
		},
		15: {
			ReserveA:  bignumber.NewUint256("98357620806573344"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      1,
			LowerTick: -8,
		},
		16: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("119012721175953760000000000000"),
			Kind:      1,
			LowerTick: 7,
		},
	}
	var binPositions = map[int32]map[uint8]uint32{
		1: {
			0: 5,
			1: 10,
			2: 2,
			3: 13,
		},
		4: {
			2: 3,
			3: 14,
		},
		6: {
			0: 6,
		},
		7: {
			0: 8,
			1: 16,
		},
		9: {
			1: 11,
		},
		-8: {
			1: 15,
			2: 1,
		},
		-7: {
			0: 4,
		},
		-1: {
			0: 7,
		},
		-9: {
			1: 9,
		},
		-3: {
			3: 12,
		},
	}

	var binMap = map[int16]*uint256.Int{
		0:  uint256.MustFromDecimal("138261823728"),
		-1: uint256.MustFromDecimal("7463162598112715418867754100145796611164620634624434827815830738677402697728"),
	}

	var state = &MaverickPoolState{
		Bins:             bins,
		TickSpacing:      953,
		Fee:              uint256.NewInt(uint64((0.3 / 100) * 1e18)),
		ActiveTick:       1,
		ProtocolFeeRatio: uint256.NewInt(0),
		BinPositions:     binPositions,
		BinMap:           binMap,
		minBinMapIndex:   -1,
		maxBinMapIndex:   0,
	}

	var amountOut = bignumber.NewUint256("1894736241169897472")
	amountIn, _, _, err := swap(state, amountOut, false, true, false)

	assert.Nil(t, err)
	assert.Equal(t, "1727696322824790157", amountIn.String())
}

func TestSwapBForAWithoutExactOut(t *testing.T) {
	t.Parallel()
	bins := map[uint32]Bin{
		1: {
			ReserveA:  bignumber.NewUint256("36455272596522751"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      2,
			LowerTick: -8,
		},
		2: {
			ReserveA:  bignumber.NewUint256("1597760289074763328"),
			ReserveB:  bignumber.NewUint256("200494651188877308086402554219"),
			Kind:      2,
			LowerTick: 1,
		},
		3: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("601955474093294592000000000000"),
			Kind:      2,
			LowerTick: 4,
		},
		4: {
			ReserveA:  bignumber.NewUint256("152321218072163223"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      0,
			LowerTick: -7,
		},
		5: {
			ReserveA:  bignumber.NewUint256("8441278100329328224"),
			ReserveB:  bignumber.NewUint256("1059252204405390821020543785987"),
			Kind:      0,
			LowerTick: 1,
		},
		6: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("92625772049616308793229705215"),
			Kind:      0,
			LowerTick: 6,
		},
		7: {
			ReserveA:  bignumber.NewUint256("784539305090201984"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      0,
			LowerTick: -1,
		},
		8: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("949292559159144576000000000000"),
			Kind:      0,
			LowerTick: 7,
		},
		9: {
			ReserveA:  bignumber.NewUint256("226486283758623329"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      1,
			LowerTick: -9,
		},
		10: {
			ReserveA:  bignumber.NewUint256("2022767072556136430"),
			ReserveB:  bignumber.NewUint256("253826547963173300232508721246"),
			Kind:      1,
			LowerTick: 1,
		},
		11: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("293121155713320256000000000000"),
			Kind:      1,
			LowerTick: 9,
		},
		12: {
			ReserveA:  bignumber.NewUint256("401225937191387328"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      3,
			LowerTick: -3,
		},
		13: {
			ReserveA:  bignumber.NewUint256("6922143575397388934"),
			ReserveB:  bignumber.NewUint256("868623892531657677331389843032"),
			Kind:      3,
			LowerTick: 1,
		},
		14: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("485483384001578688000000000000"),
			Kind:      3,
			LowerTick: 4,
		},
		15: {
			ReserveA:  bignumber.NewUint256("98357620806573344"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      1,
			LowerTick: -8,
		},
		16: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("119012721175953760000000000000"),
			Kind:      1,
			LowerTick: 7,
		},
		17: {
			ReserveA:  bignumber.NewUint256("579597339107942400"),
			ReserveB:  bignumber.NewUint256("0"),
			Kind:      3,
			LowerTick: -7,
		},
		18: {
			ReserveA:  bignumber.NewUint256("0"),
			ReserveB:  bignumber.NewUint256("701312780320610432000000000000"),
			Kind:      3,
			LowerTick: 5,
		},
	}

	binPositions := map[int32]map[uint8]uint32{
		1: {
			0: 5,
			1: 10,
			2: 2,
			3: 13,
		},
		4: {
			2: 3,
			3: 14,
		},
		5: {
			3: 18,
		},
		6: {
			0: 6,
		},
		7: {
			0: 8,
			1: 16,
		},
		9: {
			1: 11,
		},
		-8: {
			1: 15,
			2: 1,
		},
		-7: {
			0: 4,
			3: 17,
		},
		-1: {
			0: 7,
		},
		-9: {
			1: 9,
		},
		-3: {
			3: 12,
		},
	}

	binMap := map[int16]*uint256.Int{
		0:  uint256.MustFromDecimal("138261823728"),
		-1: uint256.MustFromDecimal("7463162598112715418867754100145796611164620634624434827815830738677402697728"),
	}

	var state = &MaverickPoolState{
		Bins:             bins,
		TickSpacing:      953,
		Fee:              uint256.NewInt(uint64((0.3 / 100) * 1e18)),
		ActiveTick:       1,
		ProtocolFeeRatio: uint256.NewInt(0),
		BinPositions:     binPositions,
		BinMap:           binMap,
		minBinMapIndex:   -1,
		maxBinMapIndex:   0,
	}

	var amountIn = bignumber.NewUint256("4221332000000000000")
	_, amountOut, _, err := swap(state, amountIn, false, false, false)

	assert.Nil(t, err)
	assert.Equal(t, "4629465618898435945", amountOut.String())
}

func Test_getKindsAtTick(t *testing.T) {
	t.Parallel()
	type args struct {
		binMap map[int16]*uint256.Int
		tick   int32
	}
	tests := []struct {
		name string
		args args
		want Active
	}{
		{
			"happy",
			args{
				map[int16]*uint256.Int{
					1: uint256.MustFromHex("0x1123456789abcdef"),
				},
				65,
			},
			Active{
				Word: 0xe,
				Tick: 65,
			},
		},
		{
			"no existing submap",
			args{
				map[int16]*uint256.Int{
					1: uint256.MustFromHex("0x1123456789abcdef"),
				},
				63,
			},
			Active{
				Word: 0x0,
				Tick: 63,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, getKindsAtTick(tt.args.binMap, tt.args.tick), "getKindsAtTick(%v, %v)",
				tt.args.binMap, tt.args.tick)
		})
	}
}
