package smardex

import (
	"encoding/json"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

var (
	TIMESTAMP_JAN_2020 int64 = 1577833200
	amountInT0               = parseString("1000000000000000000")
	resT0                    = parseString("13847262709278700000")
	resT1                    = parseString("119700592015995000000000")
	resFicT0                 = parseString("6441406027101710000")
	resFicT1                 = parseString("53094867866428500000000")
	priceAvT0                = parseString("1000000000000000000")
	priceAvT1                = parseString("8197837914161090000000")
	feesLP                   = big.NewInt(500)
	feesPool                 = big.NewInt(200)

	expectedResT0       = parseString("14847062709278699999")
	expectedResT1       = parseString("112484184376480628646478")
	expectedResFicT0    = parseString("8094353523617659658")
	expectedResFicT1    = parseString("51232857537391979202756")
	expectedAmountOutT0 = parseString("7216407639514371353522")
)

func parseString(value string) *big.Int {
	newValue := new(big.Int)
	newValue.SetString(value, 10)
	return newValue
}

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	var pools []*PoolSimulator
	{
		extra := SmardexPair{
			PairFee: PairFee{
				FeesLP:   feesLP,
				FeesPool: feesPool,
				FeesBase: FEES_BASE,
			},
			FictiveReserve: FictiveReserve{
				FictiveReserve0: resFicT0,
				FictiveReserve1: resFicT1,
			},
			PriceAverage: PriceAverage{
				PriceAverage0:             priceAvT0,
				PriceAverage1:             priceAvT1,
				PriceAverageLastTimestamp: big.NewInt(TIMESTAMP_JAN_2020),
			},
			FeeToAmount: FeeToAmount{
				Fees0: big.NewInt(0),
				Fees1: big.NewInt(0),
			},
		}
		extraJson, _ := json.Marshal(extra)

		token0 := entity.PoolToken{
			Address:   "token0",
			Swappable: true,
		}
		token1 := entity.PoolToken{
			Address:   "token1",
			Swappable: true,
		}

		pool := entity.Pool{
			Reserves: entity.PoolReserves{resT0.String(), resT1.String()},
			Tokens:   []*entity.PoolToken{&token0, &token1},
			Extra:    string(extraJson),
		}
		poolSimulator, err := NewPoolSimulator(pool)
		if err != nil {
			panic(err)
		}
		pools = append(pools, poolSimulator)
	}
	return pools
}
