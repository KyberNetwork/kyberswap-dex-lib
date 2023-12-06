package sd59x18

import "math/big"

func ConvertSD59x18(x *big.Int) (*SD59x18, error) {
	if x.Cmp(new(big.Int).Quo(uMIN_SD59x18, uUNIT)) < 0 {
		return nil, Err_PRBMath_SD59x18_Convert_Underflow
	}

	if x.Cmp(new(big.Int).Quo(uMAX_SD59x18, uUNIT)) > 0 {
		return nil, Err_PRBMath_SD59x18_Convert_Overflow
	}

	value := new(big.Int).Mul(x, uUNIT)

	return &SD59x18{value}, nil

}

func ConvertBI(x *SD59x18) *big.Int {
	return new(big.Int).Quo(x.value, uUNIT)
}
