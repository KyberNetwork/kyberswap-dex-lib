package uniswapv4

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func TestDirectionalSwapFee(t *testing.T) {
	lp := FeeAmount(3000)
	pf0, pf1 := FeeAmount(500), FeeAmount(1000)
	asym := &entity.Pool{
		SwapFee: float64(CalculateSwapFee(pf0, lp)),
		Extra:   `{"pF":` + strconv.FormatUint(uint64(lp)<<24|uint64(packProtocolFee(uint32(pf0), uint32(pf1))), 10) + `}`,
	}
	assert.Equal(t, CalculateSwapFee(pf0, lp), DirectionalSwapFee(asym, true))
	assert.Equal(t, CalculateSwapFee(pf1, lp), DirectionalSwapFee(asym, false))

	off := &entity.Pool{SwapFee: 3000, Extra: `{"liquidity":"1"}`}
	assert.Equal(t, FeeAmount(3000), DirectionalSwapFee(off, true))
	assert.Equal(t, FeeAmount(3000), DirectionalSwapFee(off, false))
}
