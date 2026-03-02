package axima

import (
	"testing"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

func TestPoolSimulatorWithAmountInSmallerThanFirstBin(t *testing.T) {
	poolData := "{\"address\":\"0xd991b723ea3d6cbeb04a0bfd9fbfd2d9da60cf64\",\"exchange\":\"axima\",\"type\":\"axima\",\"timestamp\":1772447695,\"reserves\":[\"7777799523750376237\",\"16888764682\"],\"tokens\":[{\"address\":\"0x4200000000000000000000000000000000000006\",\"symbol\":\"WETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"qA\\\":true,\\\"maxAge\\\":30,\\\"asks\\\":[{\\\"bi\\\":0,\\\"r\\\":0.0005124556418238227,\\\"cv\\\":323224270789556102},{\\\"bi\\\":1,\\\"r\\\":0.0005124044065067038,\\\"cv\\\":1351899229643009824},{\\\"bi\\\":2,\\\"r\\\":0.000512353181433575,\\\"cv\\\":2380755875998654861},{\\\"bi\\\":3,\\\"r\\\":0.0005123019666013645,\\\"cv\\\":3409890738877127399},{\\\"bi\\\":4,\\\"r\\\":0.0005122507620070017,\\\"cv\\\":4438109209944069518},{\\\"bi\\\":5,\\\"r\\\":0.0005121995676474167,\\\"cv\\\":5466212323142871347},{\\\"bi\\\":6,\\\"r\\\":0.0005121483835195415,\\\"cv\\\":6494568740853678506},{\\\"bi\\\":7,\\\"r\\\":0.0005120972096203088,\\\"cv\\\":7523438205822099227}],\\\"bids\\\":[{\\\"bi\\\":0,\\\"r\\\":1951.056711737211,\\\"cv\\\":384147975},{\\\"bi\\\":-1,\\\"r\\\":1950.8616060660372,\\\"cv\\\":2389857066},{\\\"bi\\\":-2,\\\"r\\\":1950.6665003948633,\\\"cv\\\":4394057845},{\\\"bi\\\":-3,\\\"r\\\":1950.4713947236896,\\\"cv\\\":6394774198},{\\\"bi\\\":-4,\\\"r\\\":1950.276289052516,\\\"cv\\\":8394221830},{\\\"bi\\\":-5,\\\"r\\\":1950.0811833813423,\\\"cv\\\":10394409415},{\\\"bi\\\":-6,\\\"r\\\":1949.8860777101686,\\\"cv\\\":12395120097},{\\\"bi\\\":-7,\\\"r\\\":1949.6909720389947,\\\"cv\\\":14396499133},{\\\"bi\\\":-8,\\\"r\\\":1949.495866367821,\\\"cv\\\":16397469480}]}\",\"staticExtra\":\"{\\\"pair\\\":\\\"WethUsdc_v13\\\"}\"}"

	var entityPool entity.Pool
	err := json.Unmarshal([]byte(poolData), &entityPool)
	assert.NoError(t, err, "failed to unmarshal pool data")

	poolSimulator, err := NewPoolSimulator(entityPool)
	assert.NoError(t, err, "failed to create pool simulator")
	poolSimulator.poolTimestamp = time.Now().Unix()

	tokenIn := poolSimulator.Pool.Info.Tokens[0]
	tokenOut := poolSimulator.Pool.Info.Tokens[1]
	amountIn := bignumber.NewBig10("50000000000000000")

	result, err := poolSimulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		TokenOut: tokenOut,
	})

	assert.NoError(t, err, "failed to calculate amount out")

	assert.Equal(t, "97552835", result.TokenAmountOut.Amount.String(), "unexpected amount out")
}
