package dodo

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func decStr(amt int64) string {
	return new(big.Int).Mul(big.NewInt(amt), bignumber.TenPowInt(18)).String()
}

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	var pools []*PoolSimulator
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			SwapFee:  0.001 + 0.002,
			Tokens:   []*entity.PoolToken{{Address: "BASE", Decimals: 18}, {Address: "QUOTE", Decimals: 18}},
			Extra: fmt.Sprintf("{\"reserves\": [%v, %v], \"targetReserves\": [%v, %v],\"i\": %v,\"k\": %v,\"rStatus\": %v,\"mtFeeRate\": \"%v\",\"lpFeeRate\": \"%v\" }",
				decStr(10), decStr(1000),
				decStr(10), decStr(1000),
				decStr(100),          // i=100
				"100000000000000000", // k=0.1
				0,
				"0.001",
				"0.002",
			),
			StaticExtra: fmt.Sprintf("{\"tokens\": [\"%v\",\"%v\"], \"type\": \"%v\", \"dodoV1SellHelper\": \"%v\"}",
				"BASE", "QUOTE", "DPP", ""),
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			SwapFee:  0.001 + 0.002,
			Tokens:   []*entity.PoolToken{{Address: "BASE", Decimals: 18}, {Address: "QUOTE", Decimals: 18}},
			Extra: fmt.Sprintf("{\"reserves\": [%v, %v], \"targetReserves\": [%v, %v],\"i\": %v,\"k\": %v,\"rStatus\": %v,\"mtFeeRate\": \"%v\",\"lpFeeRate\": \"%v\" }",
				decStr(10), decStr(1000),
				decStr(10), decStr(1000),
				decStr(100),          // i=100
				"100000000000000000", // k=0.1
				0,
				"0.001",
				"0.002",
			),
			StaticExtra: fmt.Sprintf("{\"tokens\": [\"%v\",\"%v\"], \"type\": \"%v\", \"dodoV1SellHelper\": \"%v\"}",
				"BASE", "QUOTE", "DPP", ""),
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	return pools
}
