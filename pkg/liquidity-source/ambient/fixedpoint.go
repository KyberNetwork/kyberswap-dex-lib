package ambient

import "math/big"

var (
	Q48  = new(big.Int).Lsh(big.NewInt(1), 48)
	Q64F = new(big.Int).Lsh(big.NewInt(1), 64)
	Q128 = new(big.Int).Lsh(big.NewInt(1), 128)

	mask128 = new(big.Int).Sub(Q128, big.NewInt(1))
	mask64  = new(big.Int).SetUint64(^uint64(0))
)

func MulQ64(x, y *big.Int) *big.Int {
	result := new(big.Int).Mul(x, y)
	return result.Rsh(result, 64)
}

func DivQ64(x, y *big.Int) *big.Int {
	num := new(big.Int).Lsh(x, 64)
	return num.Div(num, y)
}

func MulQ48(x *big.Int, y uint64) *big.Int {
	result := new(big.Int).Mul(x, new(big.Int).SetUint64(y))
	return result.Rsh(result, 48)
}

func RecipQ64(x *big.Int) *big.Int {
	return new(big.Int).Div(Q128, x)
}
