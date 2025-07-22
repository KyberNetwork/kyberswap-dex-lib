package honey

import (
	"testing"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func s(s string) *uint256.Int {
	res, _ := uint256.FromDecimal(s)
	return res
}
func TestGetAmountOut(t *testing.T) {
	// https://dashboard.tenderly.co/tien7668/project/simulator/8d6e4d71-ac1a-4ae3-a42d-d618a008d37d/debugger?trace=0.7.1.3.2.1.5.4
	p := &PoolSimulator{
		redeemRates:            []*uint256.Int{s("999500000000000000"), s("999500000000000000")},
		polFeeCollectorFeeRate: u256.BONE,
	}

	shares, _, polFeeCollectorFeeShares := p.getSharesRedeemedFromHoney(s("503373735599534552165958"), 1)
	assert.Equal(t, shares, s("503122048731734784889875"))
	assert.Equal(t, polFeeCollectorFeeShares, s("251686867799767276083"))
}
