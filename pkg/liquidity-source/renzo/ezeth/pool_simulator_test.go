package ezeth

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func initPoolSim() *PoolSimulator {
	data := `{"address":"0x74a09653a083691711cf8215a6ab074bb4e99ef5","exchange":"renzo-ezeth","type":"renzo-ezeth","timestamp":1756284706,"reserves":["10000000000000000000","10000000000000000000","10000000000000000000"],"tokens":[{"address":"0xbf5495efe5db9ce00f80364c8b423567e58d2110","symbol":"ezETH","decimals":18,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},{"address":"0xae7ab96520de3a18e5e111b5eaab095312d7fe84","symbol":"stETH","decimals":18,"swappable":true}],"extra":"{\"paused\":false,\"strategyManagerPaused\":false,\"collateralTokenIndex\":{\"0xae7ab96520de3a18e5e111b5eaab095312d7fe84\":0},\"operatorDelegatorTokenTvls\":[[1,23726618205752000000000],[136369169608104227457,0],[49726593155322714812702,44937798600595000000000],[28908229159363013180973,53516411391603000000000],[10845829474726579684173,54449173699067000000000]],\"operatorDelegatorTvls\":[23726618205752000000001,136369169608104227457,94664391755917714812702,82424640550966013180973,65295003173793579684173],\"totalTvl\":310814699517703687265608,\"operatorDelegatorAllocations\":[1,1,3332,3333,3333],\"tokenStrategyMapping\":[{\"0xae7ab96520de3a18e5e111b5eaab095312d7fe84\":false},{\"0xae7ab96520de3a18e5e111b5eaab095312d7fe84\":false},{\"0xae7ab96520de3a18e5e111b5eaab095312d7fe84\":false},{\"0xae7ab96520de3a18e5e111b5eaab095312d7fe84\":false},{\"0xae7ab96520de3a18e5e111b5eaab095312d7fe84\":false}],\"totalSupply\":293714268825320553419259,\"tokenOracleLookup\":{\"0xae7ab96520de3a18e5e111b5eaab095312d7fe84\":{\"answer\":1000000000000000000,\"updatedAt\":1999999999}},\"collateralTokenTvlLimits\":{\"0xae7ab96520de3a18e5e111b5eaab095312d7fe84\":320000000000000000000000}}","blockNumber":23231381}`

	var pool entity.Pool
	err := json.Unmarshal([]byte(data), &pool)
	if err != nil {
		panic(err)
	}

	poolSimulator, err := NewPoolSimulator(pool)
	if err != nil {
		panic(err)
	}

	return poolSimulator
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	t.Run("[DepositETH] it should return correct amountOut", func(t *testing.T) {
		poolSimulator := initPoolSim()

		params := poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			TokenOut: "0xbf5495efe5db9ce00f80364c8b423567e58d2110",
		}

		result, err := poolSimulator.CalcAmountOut(params)

		require.NoError(t, err)
		assert.Equal(t, bignumber.NewBig("944981911348085675"), result.TokenAmountOut.Amount)
	})

	t.Run("[Deposit] it should return correct amountOut", func(t *testing.T) {
		poolSimulator := initPoolSim()

		params := poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0xae7ab96520de3a18e5e111b5eaab095312d7fe84",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			TokenOut: "0xbf5495efe5db9ce00f80364c8b423567e58d2110",
		}

		result, err := poolSimulator.CalcAmountOut(params)

		require.NoError(t, err)
		assert.Equal(t, bignumber.NewBig("944981911348085675"), result.TokenAmountOut.Amount)
	})
}
