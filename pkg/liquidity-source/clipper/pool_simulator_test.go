package clipper

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var poolEntityStr = "{\"address\":\"0x655edce464cc797526600a462a8154650eee4b77\",\"reserveUsd\":3099576.562241563,\"amplifiedTvl\":3099576.562241563,\"exchange\":\"clipper\",\"type\":\"clipper\",\"timestamp\":1729014768,\"reserves\":[\"491115278550168767440992\",\"597835189535037939399\",\"650931997785\",\"410635515666\"],\"tokens\":[{\"address\":\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"name\":\"Dai Stablecoin\",\"symbol\":\"DAI\",\"decimals\":18},{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"name\":\"Wrapped Ether\",\"symbol\":\"WETH\",\"decimals\":18},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"name\":\"USD Coin\",\"symbol\":\"USDC\",\"decimals\":6},{\"address\":\"0xdac17f958d2ee523a2206206994597c13d831ec7\",\"name\":\"Tether USD\",\"symbol\":\"USDT\",\"decimals\":6}],\"extra\":\"{\\\"SwapsEnabled\\\":true,\\\"K\\\":0.02,\\\"TimeInSeconds\\\":60,\\\"Assets\\\":[{\\\"Address\\\":\\\"0x6b175474e89094c44da98b954eedeac495271d0f\\\",\\\"Symbol\\\":\\\"DAI\\\",\\\"Decimals\\\":18,\\\"PriceInUSD\\\":1,\\\"Quantity\\\":491115278550168767440992,\\\"ListingWeight\\\":250},{\\\"Address\\\":\\\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\\\",\\\"Symbol\\\":\\\"ETH\\\",\\\"Decimals\\\":18,\\\"PriceInUSD\\\":2587.488,\\\"Quantity\\\":597835189535037939399,\\\"ListingWeight\\\":79},{\\\"Address\\\":\\\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\\\",\\\"Symbol\\\":\\\"USDC\\\",\\\"Decimals\\\":6,\\\"PriceInUSD\\\":1,\\\"Quantity\\\":650931997785,\\\"ListingWeight\\\":188},{\\\"Address\\\":\\\"0xdac17f958d2ee523a2206206994597c13d831ec7\\\",\\\"Symbol\\\":\\\"USDT\\\",\\\"Decimals\\\":6,\\\"PriceInUSD\\\":1,\\\"Quantity\\\":410635515666,\\\"ListingWeight\\\":305}],\\\"Pairs\\\":[{\\\"Assets\\\":[\\\"ETH\\\",\\\"USDC\\\"],\\\"FeeInBasisPoints\\\":4},{\\\"Assets\\\":[\\\"ETH\\\",\\\"USDT\\\"],\\\"FeeInBasisPoints\\\":4},{\\\"Assets\\\":[\\\"ETH\\\",\\\"DAI\\\"],\\\"FeeInBasisPoints\\\":4},{\\\"Assets\\\":[\\\"USDC\\\",\\\"USDT\\\"],\\\"FeeInBasisPoints\\\":1},{\\\"Assets\\\":[\\\"USDC\\\",\\\"DAI\\\"],\\\"FeeInBasisPoints\\\":1},{\\\"Assets\\\":[\\\"USDT\\\",\\\"DAI\\\"],\\\"FeeInBasisPoints\\\":0}]}\"}"

func TestPoolSimulator(t *testing.T) {
	var poolEntity entity.Pool
	err := json.Unmarshal([]byte(poolEntityStr), &poolEntity)
	assert.NoError(t, err)

	poolSimulator, err := NewPoolSimulator(poolEntity)
	assert.NoError(t, err)

	// Swap 1 ETH to USDC
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(1e18),
		},
		TokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	}
	res, err := poolSimulator.CalcAmountOut(params)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(2586379125), res.TokenAmountOut.Amount)
}
