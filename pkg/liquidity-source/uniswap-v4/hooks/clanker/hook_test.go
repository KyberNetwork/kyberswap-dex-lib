package clanker

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
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
		Cfg:         &uniswapv4.Config{ChainID: int(chainID)},
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0xf7aC669593d2D9D01026Fa5B756DD5B4f7aAa8Cc"),
	})

	_, err := sh.Track(context.Background(), &uniswapv4.HookParam{
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

	poolStr := `{"address":"0xe2e7c620191449a2228b69231b63f2395f486ca7550de706feddf7ec9b9cd5d7","swapFee":8388608,"exchange":"uniswap-v4-clanker","type":"uniswap-v4","timestamp":1753291890,"reserves":["70747613414474548452393626071","48591284065701221020"],"tokens":[{"address":"0x6488cf3cd609f8fe683e96b461e765d98dcedb07","symbol":"🟦","decimals":18,"swappable":true},{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"liquidity\":1854108243979608403116743,\"sqrtPriceX96\":2076361055636673935789906,\"tickSpacing\":200,\"tick\":-211000,\"ticks\":[{\"index\":-211000,\"liquidityGross\":1854108243979608403116743,\"liquidityNet\":1854108243979608403116743},{\"index\":-120000,\"liquidityGross\":1854108243979608403116743,\"liquidityNet\":-1854108243979608403116743}],\"hX\":\"{\\\"ProtocolFee\\\":1000,\\\"ClankerIsToken0\\\":true,\\\"ClankerTracked\\\":true}\"}","staticExtra":"{\"0x0\":[false,false],\"fee\":8388608,\"tS\":200,\"hooks\":\"0xfd213be7883db36e1049dc42f5bd6a0ec66b68cc\",\"uR\":\"0xa51afafe0263b40edaef0df8781ea9aa03e381a3\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":360796346}`
	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.NoError(t, err)

	dh := NewDynamicFeeHook(&uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{ChainID: int(chainID)},
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0xFd213BE7883db36e1049dC42f5BD6A0ec66B68cC"),
	})

	_, err = dh.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient: rpcClient,
		Pool:      &pool,
	})
	require.NoError(t, err)
}
