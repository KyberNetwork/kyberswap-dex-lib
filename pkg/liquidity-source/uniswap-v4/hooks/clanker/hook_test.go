package clanker

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticFeeHook_Track(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://arbitrum.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	chainID := valueobject.ChainIDArbitrumOne

	sh := NewStaticFeeHook(&uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0xf7aC669593d2D9D01026Fa5B756DD5B4f7aAa8Cc"),
	})

	_, err := sh.Track(context.Background(), &uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: int(chainID)},
		RpcClient: rpcClient,
		Pool: &entity.Pool{
			Address: "0x3f3ef57297fb9f0a3dca28b15b7b6d8186c0caba8dfc82294d8181da56113a82",
			Tokens: []*entity.PoolToken{
				{
					Address: "0x3079f9fd56c1fbde6f77bcbc387f371513a00b07",
				},
			},
		},
	})
	require.NoError(t, err)
}

func TestDynamicFeeHook_Track(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://arbitrum.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	chainID := valueobject.ChainIDArbitrumOne

	poolStr := `{"address":"0xe2e7c620191449a2228b69231b63f2395f486ca7550de706feddf7ec9b9cd5d7","swapFee":8388608,"exchange":"uniswap-v4-clanker","type":"uniswap-v4","timestamp":1753291890,"reserves":["70747613414474548452393626071","48591284065701221020"],"tokens":[{"address":"0x6488cf3cd609f8fe683e96b461e765d98dcedb07","symbol":"🟦","decimals":18,"swappable":true},{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"liquidity\":1854108243979608403116743,\"sqrtPriceX96\":2076361055636673935789906,\"tickSpacing\":200,\"tick\":-211000,\"ticks\":[{\"index\":-211000,\"liquidityGross\":1854108243979608403116743,\"liquidityNet\":1854108243979608403116743},{\"index\":-120000,\"liquidityGross\":1854108243979608403116743,\"liquidityNet\":-1854108243979608403116743}],\"hX\":\"{\\\"ProtocolFee\\\":1000,\\\"ClankerIsToken0\\\":true,\\\"ClankerTracked\\\":false}\"}","staticExtra":"{\"0x0\":[false,false],\"fee\":8388608,\"tS\":200,\"hooks\":\"0xfd213be7883db36e1049dc42f5bd6a0ec66b68cc\",\"uR\":\"0xa51afafe0263b40edaef0df8781ea9aa03e381a3\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":360796346}`
	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.NoError(t, err)

	dh := NewDynamicFeeHook(&uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{ChainID: int(chainID)},
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0xFd213BE7883db36e1049dC42f5BD6A0ec66B68cC"),
	})

	_, err = dh.Track(context.Background(), &uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: int(chainID)},
		RpcClient: rpcClient,
		Pool:      &pool,
	})
	require.NoError(t, err)
}

func Test_CalcAmountOut(t *testing.T) {
	var p entity.Pool
	poolData := `{"address":"0xabb949ef8d1e86c37e2620de318da773273460a843a623c3c05d2b85dc893e77","swapFee":8388608,"exchange":"uniswap-v4","type":"uniswap-v4","timestamp":1753702228,"reserves":["83499061346031704444327158001","4265567673400257806"],"tokens":[{"address":"0x109ddc73b46b2f8141880a5573d2b0d2acf10b07","symbol":"neged2.0","decimals":18,"swappable":true},{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"liquidity\":596800550298755887846132,\"sqrtPriceX96\":566274760763156353397804,\"tickSpacing\":200,\"tick\":-236988,\"ticks\":[{\"index\":-237400,\"liquidityGross\":596800550298755887846132,\"liquidityNet\":596800550298755887846132},{\"index\":-120000,\"liquidityGross\":596800550298755887846132,\"liquidityNet\":-596800550298755887846132}],\"hX\":\"{\\\"ProtocolFee\\\":6000,\\\"ClankerFee\\\":30000,\\\"PairedFee\\\":30000,\\\"ClankerIsToken0\\\":true,\\\"ClankerTracked\\\":false}\"}","staticExtra":"{\"0x0\":[false,false],\"fee\":8388608,\"tS\":200,\"hooks\":\"0xdd5eeaff7bd481ad55db083062b13a3cdf0a68cc\",\"uR\":\"0x6ff5693b99212da76ad316178a184ab56d299b43\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":33456440}`
	assert.NoError(t, json.Unmarshal([]byte(poolData), &p))

	pSim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDBase)
	assert.NoError(t, err)

	got, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x4200000000000000000000000000000000000006",
			Amount: big.NewInt(1000000000000000),
		},
		TokenOut: "0x109ddc73b46b2f8141880a5573d2b0d2acf10b07",
	})
	assert.NoError(t, err)
	assert.Equal(t, "18870367192095562543977004", got.TokenAmountOut.Amount.String())
}

func Test_CalcAmountIn(t *testing.T) {
	var p entity.Pool
	poolData := `{"address":"0xabb949ef8d1e86c37e2620de318da773273460a843a623c3c05d2b85dc893e77","swapFee":8388608,"exchange":"uniswap-v4","type":"uniswap-v4","timestamp":1753702228,"reserves":["83499061346031704444327158001","4265567673400257806"],"tokens":[{"address":"0x109ddc73b46b2f8141880a5573d2b0d2acf10b07","symbol":"neged2.0","decimals":18,"swappable":true},{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"liquidity\":596800550298755887846132,\"sqrtPriceX96\":566274760763156353397804,\"tickSpacing\":200,\"tick\":-236988,\"ticks\":[{\"index\":-237400,\"liquidityGross\":596800550298755887846132,\"liquidityNet\":596800550298755887846132},{\"index\":-120000,\"liquidityGross\":596800550298755887846132,\"liquidityNet\":-596800550298755887846132}],\"hX\":\"{\\\"ProtocolFee\\\":6000,\\\"ClankerFee\\\":30000,\\\"PairedFee\\\":30000,\\\"ClankerIsToken0\\\":true,\\\"ClankerTracked\\\":false}\"}","staticExtra":"{\"0x0\":[false,false],\"fee\":8388608,\"tS\":200,\"hooks\":\"0xdd5eeaff7bd481ad55db083062b13a3cdf0a68cc\",\"uR\":\"0x6ff5693b99212da76ad316178a184ab56d299b43\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":33456440}`
	assert.NoError(t, json.Unmarshal([]byte(poolData), &p))

	pSim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDBase)
	assert.NoError(t, err)

	testutil.TestCalcAmountIn(t, pSim)
}
