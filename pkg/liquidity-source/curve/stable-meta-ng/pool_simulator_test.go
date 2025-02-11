package stablemetang

import (
	"fmt"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	stableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	basePools := []string{
		// base pool is NG https://etherscan.io/address/0x383e6b4437b59fff47b619cba855ca29342a8559
		"{\"address\":\"0x383e6b4437b59fff47b619cba855ca29342a8559\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1710325214,\"reserves\":[\"20645714947000\",\"16619279610257\",\"37260809758180318203561662\"],\"tokens\":[{\"address\":\"0x6c3ea9036406852006290770bedfcaba0e23a0e8\",\"symbol\":\"PYUSD\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"15000\\\",\\\"FutureA\\\":\\\"15000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"50000000000\\\",\\\"IsNativeCoins\\\":[false,false]}\",\"blockNumber\":19425514}",

		// base pool is plain https://etherscan.io/address/0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7
		"{\"address\":\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1710405854,\"reserves\":[\"74882317978601283428112533\",\"76066551886323\",\"32115318520985\",\"177637651221630809031052488\"],\"tokens\":[{\"address\":\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"symbol\":\"DAI\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xdac17f958d2ee523a2206206994597c13d831ec7\",\"symbol\":\"USDT\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"5000\\\",\\\"FutureA\\\":\\\"2000\\\",\\\"InitialATime\\\":1653559305,\\\"FutureATime\\\":1654158027,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"1\\\",\\\"LpToken\\\":\\\"0x6c3F90f043a72FA612cbac8115EE7e52BDe6E490\\\",\\\"IsNativeCoin\\\":[false,false,false]}\",\"blockNumber\":19432140}",
	}

	pools := []string{
		// https://etherscan.io/address/0x9e10f9fb6f0d32b350cee2618662243d4f24c64a
		"{\"address\":\"0x9e10f9fb6f0d32b350cee2618662243d4f24c64a\",\"exchange\":\"curve-stable-meta-ng\",\"type\":\"curve-stable-meta-ng\",\"timestamp\":1710325225,\"reserves\":[\"1400402037639032709376918\",\"389831262966377525851519\",\"1786431867672163347040320\"],\"tokens\":[{\"address\":\"0x4591dbff62656e7859afe5e45f6f47d3669fbb28\",\"symbol\":\"mkUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x383e6b4437b59fff47b619cba855ca29342a8559\",\"symbol\":\"PYUSDUSDC\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"15000\\\",\\\"FutureA\\\":\\\"15000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1000073197173325044\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\",\\\"IsNativeCoins\\\":[false,false],\\\"BasePool\\\":\\\"0x383e6b4437b59fff47b619cba855ca29342a8559\\\"}\",\"blockNumber\":19425514}",

		// https://etherscan.io/address/0x2482dfb5a65d901d137742ab1095f26374509352
		"{\"address\":\"0x2482dfb5a65d901d137742ab1095f26374509352\",\"exchange\":\"curve-stable-meta-ng\",\"type\":\"curve-stable-meta-ng\",\"timestamp\":1710405853,\"reserves\":[\"4556837199510378636842480\",\"113547535917173130561003\",\"4650797641270672114959944\"],\"tokens\":[{\"address\":\"0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54\",\"symbol\":\"PUSd\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x6c3f90f043a72fa612cbac8115ee7e52bde6e490\",\"symbol\":\"3Crv\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"100000\\\",\\\"FutureA\\\":\\\"100000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1030506792713195533\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\",\\\"IsNativeCoins\\\":[false,false],\\\"BasePool\\\":\\\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\\\"}\",\"blockNumber\":19432140}",
	}

	testcases := []struct {
		poolIdx           int
		in                string
		inAmount          string
		out               string
		expectedOutAmount string
	}{
		// meta pool swap
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "49183840532051551"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "491838395529404280"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5000000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "4918382977155694248"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "49183731951271033406"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "50000000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50781231606587338678"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "5000000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5078133338462932470"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "500000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "507813435641987387"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "50000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50781344581152143"},

		// meta -> base
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "49219"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "492193"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5000000000000000000", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "4921930"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000000", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "49219208"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000001", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "492182271"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "49146"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "491467"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5000000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "4914673"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "49146640"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000001", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "491456594"},

		// base -> meta
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "5000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5074020539796174052"},
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "50000001", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50740089393727228165"},
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "500000012", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "507390727614246026441"},
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "5000000123", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5072902983968984217079"},
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "50000001234", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50639540695688204241877"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "5000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5081403477635882869"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "50000001", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50813917368257894387"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "500000012", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "508128962320265975010"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "5000000123", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5080281497249917498225"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "50000001234", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50712988541587113680160"},

		// meta pool swap
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "50000000000000000", "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "40553074115672826"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "500000000000000000", "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "405530528618569204"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5000000000000000000", "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "4055284031866541673"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "50000000000000000000", "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "40550714375305973410"},
		{1, "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "50000000000000000000", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "61553361367675214388"},
		{1, "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "5000000000000000000", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "6155733280148991567"},
		{1, "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "500000000000000000", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "615577301147228989"},
		{1, "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "50000000000000000", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "61557769849618186"},

		// meta -> base
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "50000000000000000", "0x6b175474e89094c44da98b954eedeac495271d0f", "41794852153815678"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "500000000000000000", "0x6b175474e89094c44da98b954eedeac495271d0f", "417948302491463989"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5000000000000000000", "0x6b175474e89094c44da98b954eedeac495271d0f", "4179461119726707551"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "50000000000000000000", "0x6b175474e89094c44da98b954eedeac495271d0f", "41792420151368915339"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "500000000000000000001", "0x6b175474e89094c44da98b954eedeac495271d0f", "417704577277418691838"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "50000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "41795"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "500000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "417951"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5000000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "4179496"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "50000000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "41792769"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "500000000000000000001", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "417708068"},

		// base -> meta
		{1, "0x6b175474e89094c44da98b954eedeac495271d0f", "5000000", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5972485"},
		{1, "0x6b175474e89094c44da98b954eedeac495271d0f", "50000001", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "59724844"},
		{1, "0x6b175474e89094c44da98b954eedeac495271d0f", "500000012", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "597248450"},
		{1, "0x6b175474e89094c44da98b954eedeac495271d0f", "5000000123", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5972484507"},
		{1, "0x6b175474e89094c44da98b954eedeac495271d0f", "50000001234", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "59724845089"},
		{1, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "5000000", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5972268090131088763"},
		{1, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "50000001", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "59718928269794143059"},
		{1, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "500000012", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "596817002694550661705"},
		{1, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "5000000123", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5932446939278617406268"},
		{1, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "50000001234", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "56807326187619465404522"},
	}

	baseSimsByAddress := make(map[string]pool.IPoolSimulator, len(basePools))
	for _, basePool := range basePools {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(basePool), &poolEntity)
		require.Nil(t, err)

		if poolEntity.Exchange == stableng.DexType {
			p, err := stableng.NewPoolSimulator(poolEntity)
			require.Nil(t, err)
			baseSimsByAddress[poolEntity.Address] = p
		} else if poolEntity.Exchange == plain.DexType {
			p, err := plain.NewPoolSimulator(poolEntity)
			require.Nil(t, err)
			baseSimsByAddress[poolEntity.Address] = p
		}
	}

	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(poolRedis), &poolEntity)
		require.Nil(t, err)
		var e StaticExtra
		err = json.Unmarshal([]byte(poolEntity.StaticExtra), &e)
		require.Nil(t, err)

		p, err := NewPoolSimulator(poolEntity, baseSimsByAddress)
		require.Nil(t, err)

		// 1st meta token can be swapped to anything
		assert.Equal(t, append([]string{p.Info.Tokens[1]}, p.GetBasePoolTokens()...), p.CanSwapTo(p.Info.Tokens[0]))

		// last meta token can't be swapped to anything other than the 1st one
		assert.Equal(t, []string{p.Info.Tokens[0]}, p.CanSwapTo(p.Info.Tokens[1]))

		// base token can be swapped to anything other than the last meta token and itself
		for _, baseToken := range p.GetBasePoolTokens() {
			assert.Equal(t, []string{p.Info.Tokens[0]}, p.CanSwapTo(baseToken))
		}

		return p
	})

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			p := sims[tc.poolIdx]
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}

func TestUpdateBalance(t *testing.T) {
	basePools := []string{
		// base pool is NG https://etherscan.io/address/0x383e6b4437b59fff47b619cba855ca29342a8559
		"{\"address\":\"0x383e6b4437b59fff47b619cba855ca29342a8559\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1710382680,\"reserves\":[\"21024903652839\",\"16240730126117\",\"37260809758180318203561662\"],\"tokens\":[{\"address\":\"0x6c3ea9036406852006290770bedfcaba0e23a0e8\",\"symbol\":\"PYUSD\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"15000\\\",\\\"FutureA\\\":\\\"15000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"50000000000\\\",\\\"IsNativeCoins\\\":[false,false]}\",\"blockNumber\":19430235}",

		// base pool is plain https://etherscan.io/address/0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7
		"{\"address\":\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1710405854,\"reserves\":[\"74882317978601283428112533\",\"76066551886323\",\"32115318520985\",\"177637651221630809031052488\"],\"tokens\":[{\"address\":\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"symbol\":\"DAI\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"symbol\":\"USDC\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xdac17f958d2ee523a2206206994597c13d831ec7\",\"symbol\":\"USDT\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"5000\\\",\\\"FutureA\\\":\\\"2000\\\",\\\"InitialATime\\\":1653559305,\\\"FutureATime\\\":1654158027,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"1\\\",\\\"LpToken\\\":\\\"0x6c3F90f043a72FA612cbac8115EE7e52BDe6E490\\\",\\\"IsNativeCoin\\\":[false,false,false]}\",\"blockNumber\":19432140}",
	}

	pools := []string{
		// https://etherscan.io/address/0x9e10f9fb6f0d32b350cee2618662243d4f24c64a
		"{\"address\":\"0x9e10f9fb6f0d32b350cee2618662243d4f24c64a\",\"exchange\":\"curve-stable-meta-ng\",\"type\":\"curve-stable-meta-ng\",\"timestamp\":1710382680,\"reserves\":[\"1400402037639032709376918\",\"389831262966377525851519\",\"1786431867672163347040320\"],\"tokens\":[{\"address\":\"0x4591dbff62656e7859afe5e45f6f47d3669fbb28\",\"symbol\":\"mkUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x383e6b4437b59fff47b619cba855ca29342a8559\",\"symbol\":\"PYUSDUSDC\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"15000\\\",\\\"FutureA\\\":\\\"15000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1000073979307112987\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\",\\\"IsNativeCoins\\\":[false,false],\\\"BasePool\\\":\\\"0x383e6b4437b59fff47b619cba855ca29342a8559\\\"}\",\"blockNumber\":19430235}",

		// https://etherscan.io/address/0x2482dfb5a65d901d137742ab1095f26374509352
		"{\"address\":\"0x2482dfb5a65d901d137742ab1095f26374509352\",\"exchange\":\"curve-stable-meta-ng\",\"type\":\"curve-stable-meta-ng\",\"timestamp\":1710405853,\"reserves\":[\"4556837199510378636842480\",\"113547535917173130561003\",\"4650797641270672114959944\"],\"tokens\":[{\"address\":\"0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54\",\"symbol\":\"PUSd\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x6c3f90f043a72fa612cbac8115ee7e52bde6e490\",\"symbol\":\"3Crv\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"100000\\\",\\\"FutureA\\\":\\\"100000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1030506792713195533\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\",\\\"IsNativeCoins\\\":[false,false],\\\"BasePool\\\":\\\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\\\"}\",\"blockNumber\":19432140}",
	}

	testcases := []struct {
		poolIdx          int
		in               string
		inAmount         string
		out              string
		errorOrAmountOut interface{}
	}{
		// meta swap
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "49183803117034717"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "491838019214800179"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5000000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "4918378996577723203"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000000", "0x383e6b4437b59fff47b619cba855ca29342a8559", "49183670407928875253"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "50000000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50781517537183117580"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "5000000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5078139311329564610"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "500000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "507813806736460752"},
		{0, "0x383e6b4437b59fff47b619cba855ca29342a8559", "50000000000000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50781379429698345"},

		// meta -> base
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "49225"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "492257"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5000000000000000000", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "4922570"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000000", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "49225589"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000001", "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "492243896"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "49136"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "491362"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5000000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "4913624"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50000000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "49136126"},
		{0, "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "500000000000000000001", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "491349219"},

		// base -> meta
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "5000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5073865293005121823"},
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "50000001", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50738523479831778367"},
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "500000012", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "507372701068203994424"},
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "5000000123", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5072491923750267051609"},
		{0, "0x6c3ea9036406852006290770bedfcaba0e23a0e8", "50000001234", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50616343745703407410794"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "5000000", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5061857447205291553"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "50000001", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50618476537958348229"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "500000012", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "506176186968441504177"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "5000000123", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "5060914974457911866316"},
		{0, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "50000001234", "0x4591dbff62656e7859afe5e45f6f47d3669fbb28", "50533364232772870927842"},

		// meta swap
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "50000000000000000", "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "40553074115672826"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "500000000000000000", "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "405530481413519414"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5000000000000000000", "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "4055278837907276654"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "50000000000000000000", "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "40550190086890222769"},
		{1, "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "50000000000000000000", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "61561319644088031630"},
		{1, "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "5000000000000000000", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "6155646213875403989"},
		{1, "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "500000000000000000", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "615559766220419520"},
		{1, "0x6c3f90f043a72fa612cbac8115ee7e52bde6e490", "50000000000000000", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "61555928075812489"},

		// meta -> base
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "50000000000000000", "0x6b175474e89094c44da98b954eedeac495271d0f", "41796109159225488"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "500000000000000000", "0x6b175474e89094c44da98b954eedeac495271d0f", "417960823902730183"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5000000000000000000", "0x6b175474e89094c44da98b954eedeac495271d0f", "4179581472710410855"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "50000000000000000000", "0x6b175474e89094c44da98b954eedeac495271d0f", "41793137363823923706"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "500000000000000000001", "0x6b175474e89094c44da98b954eedeac495271d0f", "417662925440083790672"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "50000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "41742"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "500000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "417421"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5000000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "4174188"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "50000000000000000000", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "41739190"},
		{1, "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "500000000000000000001", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "417121309"},

		// base -> meta
		{1, "0x6b175474e89094c44da98b954eedeac495271d0f", "5000000", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5987911"},
		{1, "0x6b175474e89094c44da98b954eedeac495271d0f", "50000001", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "59879116"},
		{1, "0x6b175474e89094c44da98b954eedeac495271d0f", "500000012", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "598791158"},
		{1, "0x6b175474e89094c44da98b954eedeac495271d0f", "5000000123", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5987911596"},
		{1, "0x6b175474e89094c44da98b954eedeac495271d0f", "50000001234", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "59879115969"},
		{1, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "5000000", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5987693738750693871"},
		{1, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "50000001", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "59872245607473997037"},
		{1, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "500000012", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "598256912600466595840"},
		{1, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "5000000123", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "5938130354826791393080"},
		{1, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "50000001234", "0x466a756e9a7401b5e2444a3fcb3c2c12fbea0a54", "56369497076270460945432"},
	}

	baseSimsByAddress := make(map[string]pool.IPoolSimulator, len(basePools))
	for _, basePool := range basePools {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(basePool), &poolEntity)
		require.Nil(t, err)

		if poolEntity.Exchange == stableng.DexType {
			p, err := stableng.NewPoolSimulator(poolEntity)
			require.Nil(t, err)
			baseSimsByAddress[poolEntity.Address] = p
		} else if poolEntity.Exchange == plain.DexType {
			p, err := plain.NewPoolSimulator(poolEntity)
			require.Nil(t, err)
			baseSimsByAddress[poolEntity.Address] = p
		}
	}

	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(poolRedis), &poolEntity)
		require.Nil(t, err)
		var e StaticExtra
		err = json.Unmarshal([]byte(poolEntity.StaticExtra), &e)
		require.Nil(t, err)

		p, err := NewPoolSimulator(poolEntity, baseSimsByAddress)
		require.Nil(t, err)
		return p
	})

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			p := sims[tc.poolIdx]
			out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)},
					TokenOut:      tc.out,
					Limit:         nil,
				})
			})
			if expErr, ok := tc.errorOrAmountOut.(error); ok {
				require.Equal(t, expErr, err)
				return
			}

			require.Nil(t, err)
			assert.Equal(t, bignumber.NewBig10(tc.errorOrAmountOut.(string)), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)

			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  pool.TokenAmount{Token: tc.in, Amount: bignumber.NewBig10(tc.inAmount)},
				TokenAmountOut: *out.TokenAmountOut,
				Fee:            *out.Fee,
				SwapInfo:       out.SwapInfo,
				SwapLimit:      nil,
			})
		})
	}
}
