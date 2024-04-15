package valueobject

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompare(t *testing.T) {
	var a, b *TokenAmount

	assert.Equal(t, 0, a.Compare(b, false))
	assert.Equal(t, 0, a.Compare(b, false))

	a = &TokenAmount{}
	assert.Equal(t, 1, a.Compare(b, true))
	assert.Equal(t, 1, a.Compare(b, false))

	a = nil
	b = &TokenAmount{}
	assert.Equal(t, -1, a.Compare(b, true))
	assert.Equal(t, -1, a.Compare(b, false))

	a = &TokenAmount{Amount: big.NewInt(100), AmountUsd: 101}
	b = &TokenAmount{Amount: big.NewInt(99), AmountUsd: 102}
	assert.Equal(t, -1, a.Compare(b, true), "gasInclude=true -> priority amountUsd")
	assert.Equal(t, 1, a.Compare(b, false), "gasInclude=false -> amount only")

	//
	a = &TokenAmount{Amount: big.NewInt(100), AmountUsd: 101}
	b = &TokenAmount{Amount: big.NewInt(99), AmountUsd: 101}
	assert.Equal(t, 1, a.Compare(b, true), "amountUsd same -> use amount")
	assert.Equal(t, 1, a.Compare(b, false), "gasInclude=false -> amount only")

	a = &TokenAmount{Amount: big.NewInt(100), AmountAfterGas: big.NewInt(101)}
	b = &TokenAmount{Amount: big.NewInt(99), AmountAfterGas: big.NewInt(102)}
	assert.Equal(t, -1, a.Compare(b, true), "gasInclude=true -> priority amountAfterGas")
	assert.Equal(t, 1, a.Compare(b, false), "gasInclude=false -> amount only")

	a = &TokenAmount{Amount: big.NewInt(100), AmountAfterGas: big.NewInt(101)}
	b = &TokenAmount{Amount: big.NewInt(99), AmountAfterGas: big.NewInt(101)}
	assert.Equal(t, 1, a.Compare(b, true), "amountAfterGas same -> use amount")
	assert.Equal(t, 1, a.Compare(b, false), "gasInclude=false -> amount only")

	a = &TokenAmount{Amount: big.NewInt(100), AmountAfterGas: big.NewInt(102), AmountUsd: 101}
	b = &TokenAmount{Amount: big.NewInt(99), AmountAfterGas: big.NewInt(101), AmountUsd: 102}
	assert.Equal(t, 1, a.Compare(b, true), "priority amountAfterGas before amountUsd")
	assert.Equal(t, 1, a.Compare(b, false), "gasInclude=false -> amount only")

	a = &TokenAmount{Amount: big.NewInt(100), AmountAfterGas: big.NewInt(102), AmountUsd: 101}
	b = &TokenAmount{Amount: big.NewInt(99), AmountAfterGas: big.NewInt(102), AmountUsd: 102}
	assert.Equal(t, -1, a.Compare(b, true), "amountAfterGas same -> user amountUsd")
	assert.Equal(t, 1, a.Compare(b, false), "gasInclude=false -> amount only")
}
