package balancerv1

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	return []*PoolSimulator{
		{
			records: map[string]Record{
				"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
					Bound:   true,
					Balance: number.NewUint256("181453339134494385762"),
					Denorm:  number.NewUint256("25000000000000000000"),
				},
				"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": {
					Bound:   true,
					Balance: number.NewUint256("982184296"),
					Denorm:  number.NewUint256("25000000000000000000"),
				},
			},
			publicSwap: true,
			swapFee:    number.NewUint256("4000000000000000"),
			totalAmountsIn: map[string]*uint256.Int{
				"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": uint256.NewInt(0),
				"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": uint256.NewInt(0),
			},
			maxTotalAmountsIn: map[string]*uint256.Int{
				"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
				"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
			},
		},
	}
}
