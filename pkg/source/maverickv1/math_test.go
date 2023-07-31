package maverickv1_test

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/elastic"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/maverickv1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestSwapAForBWithoutExactOut(t *testing.T) {
	var bins = map[string]maverickv1.Bin{
		"1": {
			ReserveA:  bignumber.NewBig10("497483862887020288"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("2"),
			LowerTick: big.NewInt(-8),
			MergeID:   bignumber.NewBig10("0"),
		},
		"2": {
			ReserveA:  bignumber.NewBig10("497483862887020288"),
			ReserveB:  bignumber.NewBig10("601955474093294592000000000000"),
			Kind:      bignumber.NewBig10("2"),
			LowerTick: bignumber.NewBig10("1"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"3": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("601955474093294592000000000000"),
			Kind:      bignumber.NewBig10("2"),
			LowerTick: bignumber.NewBig10("4"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"4": {
			ReserveA:  bignumber.NewBig10("204096294304391520"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: big.NewInt(-7),
			MergeID:   bignumber.NewBig10("0"),
		},
		"5": {
			ReserveA:  bignumber.NewBig10("988635599394593504"),
			ReserveB:  bignumber.NewBig10("1196249075267458226326064162896"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: bignumber.NewBig10("1"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"6": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("246956516108313792000000000000"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: bignumber.NewBig10("6"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"7": {
			ReserveA:  bignumber.NewBig10("784539305090201984"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: big.NewInt(-1),
			MergeID:   bignumber.NewBig10("0"),
		},
		"8": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("949292559159144576000000000000"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: bignumber.NewBig10("7"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"9": {
			ReserveA:  bignumber.NewBig10("242248889019272896"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: big.NewInt(-9),
			MergeID:   bignumber.NewBig10("0"),
		},
		"10": {
			ReserveA:  bignumber.NewBig10("340606509825846240"),
			ReserveB:  bignumber.NewBig10("412133876889273980196333938548"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: bignumber.NewBig10("1"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"11": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("293121155713320256000000000000"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: bignumber.NewBig10("9"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"12": {
			ReserveA:  bignumber.NewBig10("401225937191387328"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("3"),
			LowerTick: big.NewInt(-3),
			MergeID:   bignumber.NewBig10("0"),
		},
		"13": {
			ReserveA:  bignumber.NewBig10("401225937191387316"),
			ReserveB:  bignumber.NewBig10("485483384001578688000000000000"),
			Kind:      bignumber.NewBig10("3"),
			LowerTick: bignumber.NewBig10("1"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"14": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("485483384001578688000000000000"),
			Kind:      bignumber.NewBig10("3"),
			LowerTick: bignumber.NewBig10("4"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"15": {
			ReserveA:  bignumber.NewBig10("98357620806573344"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: big.NewInt(-8),
			MergeID:   bignumber.NewBig10("0"),
		},
		"16": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("119012721175953760000000000000"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: bignumber.NewBig10("7"),
			MergeID:   bignumber.NewBig10("0"),
		},
	}
	var binPositions = map[string]map[string]*big.Int{
		"1": {
			"0": bignumber.NewBig10("5"),
			"1": bignumber.NewBig10("10"),
			"2": bignumber.NewBig10("2"),
			"3": bignumber.NewBig10("13"),
		},
		"4": {
			"2": bignumber.NewBig10("3"),
			"3": bignumber.NewBig10("14"),
		},
		"6": {
			"0": bignumber.NewBig10("6"),
		},
		"7": {
			"0": bignumber.NewBig10("8"),
			"1": bignumber.NewBig10("16"),
		},
		"9": {
			"1": bignumber.NewBig10("11"),
		},
		"-8": {
			"1": bignumber.NewBig10("15"),
			"2": bignumber.NewBig10("1"),
		},
		"-7": {
			"0": bignumber.NewBig10("4"),
		},
		"-1": {
			"0": bignumber.NewBig10("7"),
		},
		"-9": {
			"1": bignumber.NewBig10("9"),
		},
		"-3": {
			"3": bignumber.NewBig10("12"),
		},
	}

	var binMap = map[string]*big.Int{
		"0":  bignumber.NewBig10("138261823728"),
		"-1": bignumber.NewBig10("7463162598112715418867754100145796611164620634624434827815830738677402697728"),
	}

	var state = &maverickv1.MaverickPoolState{
		Bins:             bins,
		TickSpacing:      big.NewInt(953),
		Fee:              big.NewInt(int64((0.3 / 100) * 1e18)),
		ActiveTick:       big.NewInt(1),
		BinCounter:       big.NewInt(16),
		ProtocolFeeRatio: big.NewInt(0),
		BinPositions:     binPositions,
		BinMap:           binMap,
	}

	var amountIn = elastic.NewBig10("1850163333337788672")
	_, amountOut, err := maverickv1.GetAmountOut(state, amountIn, true, false, false)

	assert.Nil(t, err)
	assert.Equal(t, "1676945827577881677", amountOut.String())
}

func TestSwapAForBExactOut(t *testing.T) {
	var bins = map[string]maverickv1.Bin{
		"1": {
			ReserveA:  bignumber.NewBig10("497483862887020288"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("2"),
			LowerTick: big.NewInt(-8),
			MergeID:   bignumber.NewBig10("0"),
		},
		"2": {
			ReserveA:  bignumber.NewBig10("497483862887020288"),
			ReserveB:  bignumber.NewBig10("601955474093294592000000000000"),
			Kind:      bignumber.NewBig10("2"),
			LowerTick: bignumber.NewBig10("1"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"3": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("601955474093294592000000000000"),
			Kind:      bignumber.NewBig10("2"),
			LowerTick: bignumber.NewBig10("4"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"4": {
			ReserveA:  bignumber.NewBig10("204096294304391520"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: big.NewInt(-7),
			MergeID:   bignumber.NewBig10("0"),
		},
		"5": {
			ReserveA:  bignumber.NewBig10("988635599394593504"),
			ReserveB:  bignumber.NewBig10("1196249075267458226326064162896"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: bignumber.NewBig10("1"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"6": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("246956516108313792000000000000"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: bignumber.NewBig10("6"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"7": {
			ReserveA:  bignumber.NewBig10("784539305090201984"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: big.NewInt(-1),
			MergeID:   bignumber.NewBig10("0"),
		},
		"8": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("949292559159144576000000000000"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: bignumber.NewBig10("7"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"9": {
			ReserveA:  bignumber.NewBig10("242248889019272896"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: big.NewInt(-9),
			MergeID:   bignumber.NewBig10("0"),
		},
		"10": {
			ReserveA:  bignumber.NewBig10("340606509825846240"),
			ReserveB:  bignumber.NewBig10("412133876889273980196333938548"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: bignumber.NewBig10("1"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"11": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("293121155713320256000000000000"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: bignumber.NewBig10("9"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"12": {
			ReserveA:  bignumber.NewBig10("401225937191387328"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("3"),
			LowerTick: big.NewInt(-3),
			MergeID:   bignumber.NewBig10("0"),
		},
		"13": {
			ReserveA:  bignumber.NewBig10("401225937191387316"),
			ReserveB:  bignumber.NewBig10("485483384001578688000000000000"),
			Kind:      bignumber.NewBig10("3"),
			LowerTick: bignumber.NewBig10("1"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"14": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("485483384001578688000000000000"),
			Kind:      bignumber.NewBig10("3"),
			LowerTick: bignumber.NewBig10("4"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"15": {
			ReserveA:  bignumber.NewBig10("98357620806573344"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: big.NewInt(-8),
			MergeID:   bignumber.NewBig10("0"),
		},
		"16": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("119012721175953760000000000000"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: bignumber.NewBig10("7"),
			MergeID:   bignumber.NewBig10("0"),
		},
	}
	var binPositions = map[string]map[string]*big.Int{
		"1": {
			"0": bignumber.NewBig10("5"),
			"1": bignumber.NewBig10("10"),
			"2": bignumber.NewBig10("2"),
			"3": bignumber.NewBig10("13"),
		},
		"4": {
			"2": bignumber.NewBig10("3"),
			"3": bignumber.NewBig10("14"),
		},
		"6": {
			"0": bignumber.NewBig10("6"),
		},
		"7": {
			"0": bignumber.NewBig10("8"),
			"1": bignumber.NewBig10("16"),
		},
		"9": {
			"1": bignumber.NewBig10("11"),
		},
		"-8": {
			"1": bignumber.NewBig10("15"),
			"2": bignumber.NewBig10("1"),
		},
		"-7": {
			"0": bignumber.NewBig10("4"),
		},
		"-1": {
			"0": bignumber.NewBig10("7"),
		},
		"-9": {
			"1": bignumber.NewBig10("9"),
		},
		"-3": {
			"3": bignumber.NewBig10("12"),
		},
	}

	var binMap = map[string]*big.Int{
		"0":  bignumber.NewBig10("138261823728"),
		"-1": bignumber.NewBig10("7463162598112715418867754100145796611164620634624434827815830738677402697728"),
	}

	var state = &maverickv1.MaverickPoolState{
		Bins:             bins,
		TickSpacing:      big.NewInt(953),
		Fee:              big.NewInt(int64((0.3 / 100) * 1e18)),
		ActiveTick:       big.NewInt(1),
		BinCounter:       big.NewInt(16),
		ProtocolFeeRatio: big.NewInt(0),
		BinPositions:     binPositions,
		BinMap:           binMap,
	}

	var amountIn = elastic.NewBig10("2963297000000000000")
	_, amountOut, err := maverickv1.GetAmountOut(state, amountIn, true, true, false)

	assert.Nil(t, err)
	assert.Equal(t, "2963297000000000000", amountOut.String())

	//var amountIn = elastic.NewBig10("1676945827577881677")
	//amountInResult, amountOut, err := maverick.GetAmountOut(state, amountIn, true, true, false)
	//assert.Nil(t, err)
	//assert.Equal(t, "1850163333337788672", amountInResult.String())

}

func TestSwapBForAExactOut(t *testing.T) {
	var bins = map[string]maverickv1.Bin{
		"1": {
			ReserveA:  bignumber.NewBig10("497483862887020288"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("2"),
			LowerTick: big.NewInt(-8),
			MergeID:   bignumber.NewBig10("0"),
		},
		"2": {
			ReserveA:  bignumber.NewBig10("497483862887020288"),
			ReserveB:  bignumber.NewBig10("601955474093294592000000000000"),
			Kind:      bignumber.NewBig10("2"),
			LowerTick: bignumber.NewBig10("1"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"3": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("601955474093294592000000000000"),
			Kind:      bignumber.NewBig10("2"),
			LowerTick: bignumber.NewBig10("4"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"4": {
			ReserveA:  bignumber.NewBig10("204096294304391520"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: big.NewInt(-7),
			MergeID:   bignumber.NewBig10("0"),
		},
		"5": {
			ReserveA:  bignumber.NewBig10("988635599394593504"),
			ReserveB:  bignumber.NewBig10("1196249075267458226326064162896"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: bignumber.NewBig10("1"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"6": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("246956516108313792000000000000"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: bignumber.NewBig10("6"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"7": {
			ReserveA:  bignumber.NewBig10("784539305090201984"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: big.NewInt(-1),
			MergeID:   bignumber.NewBig10("0"),
		},
		"8": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("949292559159144576000000000000"),
			Kind:      bignumber.NewBig10("0"),
			LowerTick: bignumber.NewBig10("7"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"9": {
			ReserveA:  bignumber.NewBig10("242248889019272896"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: big.NewInt(-9),
			MergeID:   bignumber.NewBig10("0"),
		},
		"10": {
			ReserveA:  bignumber.NewBig10("340606509825846240"),
			ReserveB:  bignumber.NewBig10("412133876889273980196333938548"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: bignumber.NewBig10("1"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"11": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("293121155713320256000000000000"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: bignumber.NewBig10("9"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"12": {
			ReserveA:  bignumber.NewBig10("401225937191387328"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("3"),
			LowerTick: big.NewInt(-3),
			MergeID:   bignumber.NewBig10("0"),
		},
		"13": {
			ReserveA:  bignumber.NewBig10("401225937191387316"),
			ReserveB:  bignumber.NewBig10("485483384001578688000000000000"),
			Kind:      bignumber.NewBig10("3"),
			LowerTick: bignumber.NewBig10("1"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"14": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("485483384001578688000000000000"),
			Kind:      bignumber.NewBig10("3"),
			LowerTick: bignumber.NewBig10("4"),
			MergeID:   bignumber.NewBig10("0"),
		},
		"15": {
			ReserveA:  bignumber.NewBig10("98357620806573344"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: big.NewInt(-8),
			MergeID:   bignumber.NewBig10("0"),
		},
		"16": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("119012721175953760000000000000"),
			Kind:      bignumber.NewBig10("1"),
			LowerTick: bignumber.NewBig10("7"),
			MergeID:   bignumber.NewBig10("0"),
		},
	}
	var binPositions = map[string]map[string]*big.Int{
		"1": {
			"0": bignumber.NewBig10("5"),
			"1": bignumber.NewBig10("10"),
			"2": bignumber.NewBig10("2"),
			"3": bignumber.NewBig10("13"),
		},
		"4": {
			"2": bignumber.NewBig10("3"),
			"3": bignumber.NewBig10("14"),
		},
		"6": {
			"0": bignumber.NewBig10("6"),
		},
		"7": {
			"0": bignumber.NewBig10("8"),
			"1": bignumber.NewBig10("16"),
		},
		"9": {
			"1": bignumber.NewBig10("11"),
		},
		"-8": {
			"1": bignumber.NewBig10("15"),
			"2": bignumber.NewBig10("1"),
		},
		"-7": {
			"0": bignumber.NewBig10("4"),
		},
		"-1": {
			"0": bignumber.NewBig10("7"),
		},
		"-9": {
			"1": bignumber.NewBig10("9"),
		},
		"-3": {
			"3": bignumber.NewBig10("12"),
		},
	}

	var binMap = map[string]*big.Int{
		"0":  bignumber.NewBig10("138261823728"),
		"-1": bignumber.NewBig10("7463162598112715418867754100145796611164620634624434827815830738677402697728"),
	}

	var state = &maverickv1.MaverickPoolState{
		Bins:             bins,
		TickSpacing:      big.NewInt(953),
		Fee:              big.NewInt(int64((0.3 / 100) * 1e18)),
		ActiveTick:       big.NewInt(1),
		BinCounter:       big.NewInt(16),
		ProtocolFeeRatio: big.NewInt(0),
		BinPositions:     binPositions,
		BinMap:           binMap,
	}

	var amountIn = elastic.NewBig10("1894736241169897472")
	_, amountOut, err := maverickv1.GetAmountOut(state, amountIn, false, true, false)

	assert.Nil(t, err)
	assert.Equal(t, "1894736241169897472", amountOut.String())
}

func TestSwapBForAWithoutExactOut(t *testing.T) {
	bins := map[string]maverickv1.Bin{
		"1": {
			ReserveA:  bignumber.NewBig10("36455272596522751"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      big.NewInt(2),
			LowerTick: big.NewInt(-8),
			MergeID:   big.NewInt(0),
		},
		"2": {
			ReserveA:  bignumber.NewBig10("1597760289074763328"),
			ReserveB:  bignumber.NewBig10("200494651188877308086402554219"),
			Kind:      big.NewInt(2),
			LowerTick: big.NewInt(1),
			MergeID:   big.NewInt(0),
		},
		"3": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("601955474093294592000000000000"),
			Kind:      big.NewInt(2),
			LowerTick: big.NewInt(4),
			MergeID:   big.NewInt(0),
		},
		"4": {
			ReserveA:  bignumber.NewBig10("152321218072163223"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      big.NewInt(0),
			LowerTick: big.NewInt(-7),
			MergeID:   big.NewInt(0),
		},
		"5": {
			ReserveA:  bignumber.NewBig10("8441278100329328224"),
			ReserveB:  bignumber.NewBig10("1059252204405390821020543785987"),
			Kind:      big.NewInt(0),
			LowerTick: big.NewInt(1),
			MergeID:   big.NewInt(0),
		},
		"6": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("92625772049616308793229705215"),
			Kind:      big.NewInt(0),
			LowerTick: big.NewInt(6),
			MergeID:   big.NewInt(0),
		},
		"7": {
			ReserveA:  bignumber.NewBig10("784539305090201984"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      big.NewInt(0),
			LowerTick: big.NewInt(-1),
			MergeID:   big.NewInt(0),
		},
		"8": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("949292559159144576000000000000"),
			Kind:      big.NewInt(0),
			LowerTick: big.NewInt(7),
			MergeID:   big.NewInt(0),
		},
		"9": {
			ReserveA:  bignumber.NewBig10("226486283758623329"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      big.NewInt(1),
			LowerTick: big.NewInt(-9),
			MergeID:   big.NewInt(0),
		},
		"10": {
			ReserveA:  bignumber.NewBig10("2022767072556136430"),
			ReserveB:  bignumber.NewBig10("253826547963173300232508721246"),
			Kind:      big.NewInt(1),
			LowerTick: big.NewInt(1),
			MergeID:   big.NewInt(0),
		},
		"11": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("293121155713320256000000000000"),
			Kind:      big.NewInt(1),
			LowerTick: big.NewInt(9),
			MergeID:   big.NewInt(0),
		},
		"12": {
			ReserveA:  bignumber.NewBig10("401225937191387328"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      big.NewInt(3),
			LowerTick: big.NewInt(-3),
			MergeID:   big.NewInt(0),
		},
		"13": {
			ReserveA:  bignumber.NewBig10("6922143575397388934"),
			ReserveB:  bignumber.NewBig10("868623892531657677331389843032"),
			Kind:      big.NewInt(3),
			LowerTick: big.NewInt(1),
			MergeID:   big.NewInt(0),
		},
		"14": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("485483384001578688000000000000"),
			Kind:      big.NewInt(3),
			LowerTick: big.NewInt(4),
			MergeID:   big.NewInt(0),
		},
		"15": {
			ReserveA:  bignumber.NewBig10("98357620806573344"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      big.NewInt(1),
			LowerTick: big.NewInt(-8),
			MergeID:   big.NewInt(0),
		},
		"16": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("119012721175953760000000000000"),
			Kind:      big.NewInt(1),
			LowerTick: big.NewInt(7),
			MergeID:   big.NewInt(0),
		},
		"17": {
			ReserveA:  bignumber.NewBig10("579597339107942400"),
			ReserveB:  bignumber.NewBig10("0"),
			Kind:      big.NewInt(3),
			LowerTick: big.NewInt(-7),
			MergeID:   big.NewInt(0),
		},
		"18": {
			ReserveA:  bignumber.NewBig10("0"),
			ReserveB:  bignumber.NewBig10("701312780320610432000000000000"),
			Kind:      big.NewInt(3),
			LowerTick: big.NewInt(5),
			MergeID:   big.NewInt(0),
		},
	}

	binPositions := map[string]map[string]*big.Int{
		"1": {
			"0": big.NewInt(5),
			"1": big.NewInt(10),
			"2": big.NewInt(2),
			"3": big.NewInt(13),
		},
		"4": {
			"2": big.NewInt(3),
			"3": big.NewInt(14),
		},
		"5": {
			"3": big.NewInt(18),
		},
		"6": {
			"0": big.NewInt(6),
		},
		"7": {
			"0": big.NewInt(8),
			"1": big.NewInt(16),
		},
		"9": {
			"1": big.NewInt(11),
		},
		"-8": {
			"1": big.NewInt(15),
			"2": big.NewInt(1),
		},
		"-7": {
			"0": big.NewInt(4),
			"3": big.NewInt(17),
		},
		"-1": {
			"0": big.NewInt(7),
		},
		"-9": {
			"1": big.NewInt(9),
		},
		"-3": {
			"3": big.NewInt(12),
		},
	}

	binMap := map[string]*big.Int{
		"0":  bignumber.NewBig10("138270212336"),
		"-1": bignumber.NewBig10("7463166048985888814149647817523727749677346860178920913009108319939514597376"),
	}

	var state = &maverickv1.MaverickPoolState{
		Bins:             bins,
		TickSpacing:      big.NewInt(953),
		Fee:              big.NewInt(int64((0.3 / 100) * 1e18)),
		ActiveTick:       big.NewInt(1),
		BinCounter:       big.NewInt(18),
		ProtocolFeeRatio: big.NewInt(0),
		BinPositions:     binPositions,
		BinMap:           binMap,
	}

	var amountIn = elastic.NewBig10("4221332000000000000")
	_, amountOut, err := maverickv1.GetAmountOut(state, amountIn, false, false, false)

	assert.Nil(t, err)
	assert.Equal(t, "4629465618898435945", amountOut.String())
	//assert.Equal(t, "1676945", new(big.Int).Div(amountOut, bignumber.TenPowInt(12)).String())
}
