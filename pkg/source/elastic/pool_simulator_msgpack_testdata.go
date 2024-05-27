package elastic

import (
	"math/big"
	"math/rand"

	elasticconstants "github.com/KyberNetwork/elastic-go-sdk/v2/constants"
	elasticentities "github.com/KyberNetwork/elastic-go-sdk/v2/entities"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/ethereum/go-ethereum/common"
)

func randomBigInt() *big.Int {
	words := make([]big.Word, 4)
	for i := range words {
		words[i] = big.Word(rand.Uint64())
	}
	return new(big.Int).SetBits(words)
}

func randomAddress() common.Address {
	buf := make([]byte, common.AddressLength)
	for i := range buf {
		buf[i] = byte(rand.Uint64() % 256)
	}
	return common.BytesToAddress(buf)
}

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	var pools []*PoolSimulator
	{
		elasticPool := &elasticentities.Pool{
			Token0:             entities.NewToken(uint(valueobject.ChainIDEthereum), randomAddress(), 18, "Token0", "Token0"),
			Token1:             entities.NewToken(uint(valueobject.ChainIDEthereum), randomAddress(), 18, "Token1", "Token1"),
			Fee:                elasticconstants.FeeAmount(rand.Uint64()),
			SqrtP:              randomBigInt(),
			BaseL:              randomBigInt(),
			ReinvestL:          randomBigInt(),
			CurrentTick:        rand.Int(),
			NearestCurrentTick: rand.Int(),
			Ticks: map[int]elasticentities.TickData{
				1: {
					LiquidityGross: randomBigInt(),
					LiquidityNet:   randomBigInt(),
				},
				100: {
					LiquidityGross: randomBigInt(),
					LiquidityNet:   randomBigInt(),
				},
			},
			InitializedTicks: map[int]elasticentities.LinkedListData{
				1: {
					Previous: rand.Int(),
					Next:     rand.Int(),
				},
				100: {
					Previous: rand.Int(),
					Next:     rand.Int(),
				},
			},
		}
		pool := &PoolSimulator{
			Pool: pool.Pool{
				Info: pool.PoolInfo{
					Address:    randomAddress().Hex(),
					ReserveUsd: rand.Float64(),
					SwapFee:    randomBigInt(),
					Exchange:   "uniswapv3",
					Type:       "uniswapv3",
					Tokens: []string{
						randomAddress().Hex(),
						randomAddress().Hex(),
					},
					Reserves: []*big.Int{
						randomBigInt(),
						randomBigInt(),
					},
					Checked:     true,
					BlockNumber: rand.Uint64(),
				},
			},
			elasticPool: elasticPool,
			gas: Gas{
				SwapBase:    int64(rand.Int()),
				SwapNonBase: int64(rand.Int()),
			},
			tickMin: rand.Int(),
			tickMax: rand.Int(),
		}
		pools = append(pools, pool)
	}
	return pools
}
