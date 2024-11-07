package vooi

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
)

func sqrt(y *big.Int, guess *big.Int) *big.Int {
	var z *big.Int

	if y.Cmp(integer.Three()) > 0 {
		minusGuess := new(big.Int).Mul(guess, big.NewInt(-1))

		if guess.Cmp(integer.Zero()) > 0 && guess.Cmp(y) <= 0 {
			z = new(big.Int).Set(guess)
		} else if guess.Cmp(integer.Zero()) < 0 && minusGuess.Cmp(y) <= 0 {
			z = minusGuess
		} else {
			z = new(big.Int).Set(y)
		}

		x := new(big.Int).Div(
			new(big.Int).Add(
				new(big.Int).Div(y, z),
				z,
			),
			integer.Two(),
		)

		for ok := true; ok; ok = x.Cmp(z) != 0 {
			z = new(big.Int).Set(x)
			x = new(big.Int).Div(new(big.Int).Add(new(big.Int).Div(y, x), x), integer.Two())
		}

	} else if y.Cmp(integer.Zero()) != 0 {
		z = integer.One()
	}

	return z
}
