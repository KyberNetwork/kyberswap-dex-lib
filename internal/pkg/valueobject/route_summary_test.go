package valueobject

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/router-service/pkg/crypto"
)

// goos: darwin
// goarch: arm64
// pkg: github.com/KyberNetwork/router-service/internal/pkg/valueobject
// cpu: Apple M1 Pro
// BenchmarkSHA256-8          10000            367431 ns/op           80714 B/op      10016 allocs/op
// if we use json, benchmarks result as below
// BenchmarkSHA256-8           8376           2864960 ns/op         1723102 B/op      12597 allocs/op
func BenchmarkRouteSummaryChecksum(b *testing.B) {
	routeSummary := RouteSummary{
		TokenIn:      "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		AmountIn:     big.NewInt(50000000),
		AmountInUSD:  0.000000036766171242625,
		TokenOut:     "0xe9e7cea3dedca5984780bafc599bd69add087d56",
		AmountOut:    big.NewInt(32423144783),
		AmountOutUSD: 0.00000003754795749851616,
		Gas:          125000,
		GasPrice:     big.NewFloat(1000000000),
		GasUSD:       0.0919154281065625,
		ExtraFee: ExtraFee{
			FeeAmount:   big.NewInt(1000),
			ChargeFeeBy: "",
			IsInBps:     true,
			FeeReceiver: "",
		},
		Route: [][]Swap{
			{
				{
					Pool:              "0xce4e5ffb961aee2eb644186b9dcc0d804452454a",
					TokenIn:           "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
					TokenOut:          "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
					LimitReturnAmount: big.NewInt(0),
					SwapAmount:        big.NewInt(50000000),
					AmountOut:         big.NewInt(8796533),
					Exchange:          "iziswap",
					PoolLength:        2,
					PoolType:          "iziswap",
				},
				{
					Pool:              "0x8b059170f5d4d85d13ae3cabd9b03abfbb2de829",
					TokenIn:           "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
					TokenOut:          "0x55d398326f99059ff775485246999027b3197955",
					LimitReturnAmount: big.NewInt(0),
					SwapAmount:        big.NewInt(8796533),
					AmountOut:         big.NewInt(32461003255),
					Exchange:          "iziswap",
					PoolLength:        2,
					PoolType:          "iziswap",
				},
			},
		},
	}
	randomeSalt := "randomSalt"
	for i := 0; i < b.N; i++ {
		routeSummary.Route = append(routeSummary.Route, []Swap{
			{
				Pool:              "0x3a667100753cfb7538208af98cb472f65f10da87",
				TokenIn:           "0x55d398326f99059ff775485246999027b3197955",
				TokenOut:          "0xe9e7cea3dedca5984780bafc599bd69add087d56",
				LimitReturnAmount: big.NewInt(0),
				SwapAmount:        big.NewInt(32461003255),
				AmountOut:         big.NewInt(32423144783),
				Exchange:          "nomiswap-stable",
				PoolLength:        2,
				PoolType:          "nomiswap-stable",
			},
		})
		crypto.NewChecksum(routeSummary, randomeSalt)
	}
}
