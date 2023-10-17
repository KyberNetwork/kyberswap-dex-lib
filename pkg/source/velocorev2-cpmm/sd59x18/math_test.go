package sd59x18

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMsb(t *testing.T) {
	x := big.NewInt(1696)
	msb := msb(x)

	bFloat := big.NewInt(int64(5e17))
	t.Error(bFloat.String())

	assert.Equal(t, 10, int(msb.Int64()))
}
