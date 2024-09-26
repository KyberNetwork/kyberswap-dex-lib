package dexT1

import (
	"math/big"
	"testing"
)

var colReservesOne = CollateralReserves{
	Token0RealReserves:      big.NewInt(20000000006000000),
	Token1RealReserves:      big.NewInt(20000000000500000),
	Token0ImaginaryReserves: big.NewInt(389736659726997981),
	Token1ImaginaryReserves: big.NewInt(389736659619871949),
}

var colReservesEmpty = CollateralReserves{
	Token0RealReserves:      big.NewInt(0),
	Token1RealReserves:      big.NewInt(0),
	Token0ImaginaryReserves: big.NewInt(0),
	Token1ImaginaryReserves: big.NewInt(0),
}

var debtReservesEmpty = DebtReserves{
	Token0RealReserves:      big.NewInt(0),
	Token1RealReserves:      big.NewInt(0),
	Token0ImaginaryReserves: big.NewInt(0),
	Token1ImaginaryReserves: big.NewInt(0),
}

var debtReservesOne = DebtReserves{
	Token0RealReserves:      big.NewInt(9486832995556050),
	Token1RealReserves:      big.NewInt(9486832993079885),
	Token0ImaginaryReserves: big.NewInt(184868330099560759),
	Token1ImaginaryReserves: big.NewInt(184868330048879109),
}

func assertSwapInResult(t *testing.T, expected bool, amountIn *big.Int, colReserves CollateralReserves, debtReserves DebtReserves, expectedAmountIn string, expectedAmountOut string) {
	inAmt, outAmt := swapIn(expected, amountIn, colReserves, debtReserves)

	if inAmt.String() != expectedAmountIn {
		t.Errorf("Expected amountIn to be %s, but got %s", expectedAmountIn, inAmt.String())
	}
	if outAmt.String() != expectedAmountOut {
		t.Errorf("Expected amountOut to be %s, but got %s", expectedAmountOut, outAmt.String())
	}
}

func assertSwapOutResult(t *testing.T, expected bool, amountOut *big.Int, colReserves CollateralReserves, debtReserves DebtReserves, expectedAmountIn string, expectedAmountOut string) {
	inAmt, outAmt := swapOut(expected, amountOut, colReserves, debtReserves)

	if inAmt.String() != expectedAmountIn {
		t.Errorf("Expected amountIn to be %s, but got %s", expectedAmountIn, inAmt.String())
	}
	if outAmt.String() != expectedAmountOut {
		t.Errorf("Expected amountOut to be %s, but got %s", expectedAmountOut, outAmt.String())
	}
}

func TestSwapIn(t *testing.T) {
	t.Run("TestSwapIn", func(t *testing.T) {
		assertSwapInResult(t, true, big.NewInt(1e15), colReservesOne, debtReservesOne, "1000000000000000", "998262697204710")
		assertSwapInResult(t, true, big.NewInt(1e15), colReservesEmpty, debtReservesOne, "1000000000000000", "994619847016724")
		assertSwapInResult(t, true, big.NewInt(1e15), colReservesOne, debtReservesEmpty, "1000000000000000", "997440731289905")
		assertSwapInResult(t, false, big.NewInt(1e15), colReservesOne, debtReservesOne, "1000000000000000", "998262697752553")
		assertSwapInResult(t, false, big.NewInt(1e15), colReservesEmpty, debtReservesOne, "1000000000000000", "994619847560607")
		assertSwapInResult(t, false, big.NewInt(1e15), colReservesOne, debtReservesEmpty, "1000000000000000", "997440731837532")
	})
}

func TestSwapOut(t *testing.T) {
	t.Run("TestSwapOut", func(t *testing.T) {
		assertSwapOutResult(t, true, big.NewInt(1e15), colReservesOne, debtReservesOne, "1001743360284199", "1000000000000000")
		assertSwapOutResult(t, true, big.NewInt(1e15), colReservesEmpty, debtReservesOne, "1005438674786548", "1000000000000000")
		assertSwapOutResult(t, true, big.NewInt(1e15), colReservesOne, debtReservesEmpty, "1002572435818386", "1000000000000000")
		assertSwapOutResult(t, false, big.NewInt(1e15), colReservesOne, debtReservesOne, "1001743359733488", "1000000000000000")
		assertSwapOutResult(t, false, big.NewInt(1e15), colReservesEmpty, debtReservesOne, "1005438674233767", "1000000000000000")
		assertSwapOutResult(t, false, big.NewInt(1e15), colReservesOne, debtReservesEmpty, "1002572435266527", "1000000000000000")
	})
}

func TestSwapInOut(t *testing.T) {
	t.Run("TestSwapInOut", func(t *testing.T) {
		assertSwapInResult(t, true, big.NewInt(1e15), colReservesOne, debtReservesOne, "1000000000000000", "998262697204710")

		assertSwapOutResult(t, true, big.NewInt(998262697204710), colReservesOne, debtReservesOne, "999999999999998", "998262697204710")

		assertSwapInResult(t, false, big.NewInt(1e15), colReservesOne, debtReservesOne, "1000000000000000", "998262697752553")

		assertSwapOutResult(t, false, big.NewInt(998262697752553), colReservesOne, debtReservesOne, "999999999999998", "998262697752553")
	})
}

func TestSwapInOutDebtEmpty(t *testing.T) {
	t.Run("TestSwapInOutDebtEmpty", func(t *testing.T) {
		assertSwapInResult(t, true, big.NewInt(1e15), colReservesEmpty, debtReservesOne, "1000000000000000", "994619847016724")

		assertSwapOutResult(t, true, big.NewInt(994619847016724), colReservesEmpty, debtReservesOne, "999999999999999", "994619847016724")

		assertSwapInResult(t, false, big.NewInt(1e15), colReservesEmpty, debtReservesOne, "1000000000000000", "994619847560607")

		assertSwapOutResult(t, false, big.NewInt(994619847560607), colReservesEmpty, debtReservesOne, "999999999999999", "994619847560607")
	})

}

func TestSwapInOutColEmpty(t *testing.T) {
	t.Run("TestSwapInOutColEmpty", func(t *testing.T) {
		assertSwapInResult(t, true, big.NewInt(1e15), colReservesOne, debtReservesEmpty, "1000000000000000", "997440731289905")

		assertSwapOutResult(t, true, big.NewInt(997440731289905), colReservesOne, debtReservesEmpty, "999999999999999", "997440731289905")

		assertSwapInResult(t, false, big.NewInt(1e15), colReservesOne, debtReservesEmpty, "1000000000000000", "997440731837532")

		assertSwapOutResult(t, false, big.NewInt(997440731837532), colReservesOne, debtReservesEmpty, "999999999999999", "997440731837532")
	})
}
