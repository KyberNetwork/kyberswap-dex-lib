package traderjoecommon

import "math/big"

func CalculateLiquidity(priceX128, binReserveX, binReserveY *big.Int) *big.Int {
	// https://docs.traderjoexyz.com/concepts/bin-liquidity#introduction
	liquidity := new(big.Int).Mul(priceX128, binReserveX)
	liquidity.Rsh(liquidity, 128)
	liquidity.Add(liquidity, binReserveY)
	return liquidity
}
