package bunniv2

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	minPriceLimit, _ = new(big.Int).SetString("4295128740", 10)
	maxPriceLimit, _ = new(big.Int).SetString("1461446703485210103287273052203988822378723970341", 10)
)

func TestHookV121_Track(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://unichain.drpc.org").
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))

	p := &entity.Pool{
		Address: "0xe7b110a6045c9e17b97902a414604b96ef0ccd227abbb0f0761da09437522e4d",
		Tokens: []*entity.PoolToken{
			{
				Address: "0x4200000000000000000000000000000000000006",
			},
			{
				Address: "0x7dcc39b4d1c53cb31e1abc0e358b43987fef80f7",
			},
		},
		StaticExtra: "{\"tickSpacing\":1}",
	}

	cfg := &uniswapv4.Config{
		ChainID: 130,
	}

	hookExtra := ""

	h := NewHook(&uniswapv4.HookParam{
		Cfg:         cfg,
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0x000052423c1dB6B7ff8641b85A7eEfc7B2791888"),
		Pool:        p,
	})

	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		Cfg:       cfg,
		RpcClient: rpcClient,
		Pool:      p,
		HookExtra: hookExtra,
	})

	require.NoError(t, err)

	_, err = h.GetReserves(context.Background(), &uniswapv4.HookParam{
		Cfg:       cfg,
		RpcClient: rpcClient,
		Pool:      p,
		HookExtra: hookExtra,
	})

	require.NoError(t, err)
}

func TestHookV120_Track(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://unichain.drpc.org").
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))

	p := &entity.Pool{
		Address: "0xeec51c6b1a9e7c4bb4fc4fa9a02fc4fff3fe94efd044f895d98b5bfbd2ff9433",
		Tokens: []*entity.PoolToken{
			{
				Address: "0x078d782b760474a361dda0af3839290b0ef57ad6",
			},
			{
				Address: "0x9151434b16b9763660705744891fa906f660ecc5",
			},
		},
		StaticExtra: "{\"tickSpacing\":1}",
	}

	cfg := &uniswapv4.Config{
		ChainID: 130,
	}

	hookExtra := ""

	h := NewHook(&uniswapv4.HookParam{
		Cfg:         cfg,
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0x005af73a245d8171a0550ffae2631f12cc211888"),
		Pool:        p,
	})

	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		Cfg:       cfg,
		RpcClient: rpcClient,
		Pool:      p,
		HookExtra: hookExtra,
	})

	require.NoError(t, err)

	_, err = h.GetReserves(context.Background(), &uniswapv4.HookParam{
		Cfg:       cfg,
		RpcClient: rpcClient,
		Pool:      p,
		HookExtra: hookExtra,
	})

	require.NoError(t, err)
}

func Test_Pool_V120(t *testing.T) {
	var p entity.Pool
	poolData := `{"address":"0xeec51c6b1a9e7c4bb4fc4fa9a02fc4fff3fe94efd044f895d98b5bfbd2ff9433","exchange":"uniswap-v4-bunni-v2","type":"uniswap-v4","timestamp":1754977104,"reserves":["12398375835635","6482803061177"],"tokens":[{"address":"0x078d782b760474a361dda0af3839290b0ef57ad6","symbol":"USDC","decimals":6,"swappable":true},{"address":"0x9151434b16b9763660705744891fa906f660ecc5","symbol":"USD₮0","decimals":6,"swappable":true}],"extra":"{\"liquidity\":0,\"sqrtPriceX96\":79208352997136529422885942753,\"tickSpacing\":1,\"tick\":-6,\"ticks\":null,\"hX\":\"{\\\"he\\\":\\\"\\\",\\\"ha\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"la\\\":\\\"0x000000e22477c615223e430266ad8d5285636e30\\\",\\\"hf\\\":\\\"0\\\",\\\"pmr\\\":[\\\"112319328837422\\\",\\\"73696823448326\\\"],\\\"ls\\\":[1,255,255,251,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],\\\"v\\\":[{\\\"a\\\":\\\"0x6eae95ee783e4d862867c4e0e4c3f4b95aa682ba\\\",\\\"d\\\":6,\\\"rr\\\":\\\"1013345126683462917\\\",\\\"md\\\":\\\"86240534255872\\\"},{\\\"a\\\":\\\"0xd49181c522ecdb265f0d9c175cf26fface64ead3\\\",\\\"d\\\":6,\\\"rr\\\":\\\"1009198311436472844\\\",\\\"md\\\":\\\"77087553264993\\\"}],\\\"aa\\\":{\\\"am\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"sf01\\\":\\\"0\\\",\\\"sf10\\\":\\\"0\\\"},\\\"os\\\":{\\\"i\\\":44,\\\"c\\\":97,\\\"cn\\\":97,\\\"io\\\":{\\\"bt\\\":1754969019,\\\"pt\\\":-2,\\\"tc\\\":-21376024,\\\"i\\\":true}},\\\"cf\\\":{\\\"fr\\\":\\\"0\\\"},\\\"o\\\":[{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":3953048515,\\\"pt\\\":-974589,\\\"tc\\\":-9923440841788703,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":86400,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":4080116892,\\\"pt\\\":-6542852,\\\"tc\\\":16647030664248144,\\\"i\\\":false},{\\\"bt\\\":1749155135,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false}],\\\"hp\\\":{\\\"fmin\\\":\\\"3\\\",\\\"fmax\\\":\\\"500\\\",\\\"fqm\\\":\\\"30000\\\",\\\"ftsa\\\":43200,\\\"sfhl\\\":\\\"30\\\",\\\"sfat\\\":60,\\\"vst0\\\":\\\"100\\\",\\\"vst1\\\":\\\"100\\\",\\\"aae\\\":false,\\\"omi\\\":1800},\\\"s0\\\":{\\\"spx96\\\":\\\"79224177920714641852577854858\\\",\\\"t\\\":-2,\\\"lst\\\":1754969019,\\\"lsgt\\\":1754928776},\\\"bs\\\":{\\\"h\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"tsa\\\":172800,\\\"lp\\\":[0,255,255,252,0,4,1,252,147,80,29,205,101,0,0,4,17,225,163,0,29,205,101,0,59,154,202,0,0,0,0,0],\\\"hp\\\":\\\"AAADAAH0AHUwAKjAAAAAAB4APABkAGQACgAyBwgBLAAAAAcIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==\\\",\\\"lt\\\":2,\\\"mrtr0\\\":\\\"50000\\\",\\\"trtr0\\\":\\\"100000\\\",\\\"xrtr0\\\":\\\"150000\\\",\\\"mrtr1\\\":\\\"50000\\\",\\\"trtr1\\\":\\\"100000\\\",\\\"xrtr1\\\":\\\"150000\\\",\\\"c0d\\\":6,\\\"c1d\\\":6,\\\"rb0\\\":\\\"2357552360228\\\",\\\"rb1\\\":\\\"833465697000\\\",\\\"r0\\\":\\\"16950594868796\\\",\\\"r1\\\":\\\"7152894616530\\\",\\\"ib\\\":[128,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,10,97,223,22,69,152]},\\\"vsp\\\":{\\\"i\\\":true,\\\"sp0\\\":\\\"1013345126683454711\\\",\\\"sp1\\\":\\\"1009198311436457056\\\"},\\\"bt\\\":1754969020}\"}","staticExtra":"{\"0x0\":[false,false],\"fee\":0,\"tS\":1,\"hooks\":\"0x005af73a245d8171a0550ffae2631f12cc211888\",\"uR\":\"0xef740bf23acae26f6492b10de645d6b98dc8eaf3\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":24228740}`

	assert.NoError(t, json.Unmarshal([]byte(poolData), &p))

	pSim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDUnichain)
	assert.NoError(t, err)

	got, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x9151434b16b9763660705744891fa906f660ecc5",
			Amount: big.NewInt(6439251628),
		},
		TokenOut: "0x078d782b760474a361dda0af3839290b0ef57ad6",
	})
	assert.NoError(t, err)
	assert.Equal(t, "6439872800", got.TokenAmountOut.Amount.String())
}

func Test_Pool_ETH_weETH(t *testing.T) {
	var p entity.Pool
	poolData := `{"address":"0x6923777072439713c7b8ab34903e0ea96078d7148449bf54f420320d59ede857","exchange":"uniswap-v4-bunni-v2","type":"uniswap-v4","timestamp":1754919528,"reserves":["562502117403109941594","2262989498277861850577"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x7dcc39b4d1c53cb31e1abc0e358b43987fef80f7","symbol":"weETH","decimals":18,"swappable":true}],"extra":"{\"liquidity\":0,\"sqrtPriceX96\":76540554206939071665077752050,\"tickSpacing\":1,\"tick\":-691,\"ticks\":null,\"hX\":\"{\\\"he\\\":\\\"{\\\\\\\"OverrideZeroToOne\\\\\\\":false,\\\\\\\"FeeZeroToOne\\\\\\\":\\\\\\\"0\\\\\\\",\\\\\\\"OverrideOneToZero\\\\\\\":false,\\\\\\\"FeeOneToZero\\\\\\\":\\\\\\\"0\\\\\\\"}\\\",\\\"ha\\\":\\\"0x00ece5a72612258f20eb24573c544f9dd8c5000c\\\",\\\"la\\\":\\\"0x000000000b757686c9596cada54fa28f8c429e0d\\\",\\\"hf\\\":\\\"0\\\",\\\"pmr\\\":[\\\"6986395919281507635\\\",\\\"4945746110165988959275\\\"],\\\"ls\\\":[1,255,253,45,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],\\\"v\\\":[{\\\"a\\\":\\\"0x1f3134c3f3f8add904b9635acbefc0ea0d0e1ffc\\\",\\\"d\\\":18,\\\"rr\\\":\\\"1006357737310987480\\\",\\\"md\\\":\\\"8884004443478294841207\\\"},{\\\"a\\\":\\\"0xe36da4ea4d07e54b1029ef26a896a656a3729f86\\\",\\\"d\\\":18,\\\"rr\\\":\\\"1000523215254771914\\\",\\\"md\\\":\\\"16549659997395619212349\\\"}],\\\"aa\\\":{\\\"am\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"sf01\\\":\\\"0\\\",\\\"sf10\\\":\\\"0\\\"},\\\"os\\\":{\\\"i\\\":8,\\\"c\\\":49,\\\"cn\\\":49,\\\"io\\\":{\\\"bt\\\":1754919491,\\\"pt\\\":-701,\\\"tc\\\":-1529224472,\\\"i\\\":true}},\\\"cf\\\":{\\\"fr\\\":\\\"0\\\"},\\\"o\\\":[{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":3708425450,\\\"pt\\\":-3005853,\\\"tc\\\":-26698954285081392,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":43200,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":4080116892,\\\"pt\\\":-6542852,\\\"tc\\\":16647030664248144,\\\"i\\\":false},{\\\"bt\\\":1749686191,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false},{\\\"bt\\\":0,\\\"pt\\\":0,\\\"tc\\\":0,\\\"i\\\":false}],\\\"hp\\\":{\\\"fmin\\\":\\\"90\\\",\\\"fmax\\\":\\\"90\\\",\\\"fqm\\\":\\\"0\\\",\\\"ftsa\\\":0,\\\"sfhl\\\":\\\"60\\\",\\\"sfat\\\":120,\\\"vst0\\\":\\\"100\\\",\\\"vst1\\\":\\\"100\\\",\\\"aae\\\":false,\\\"omi\\\":1800},\\\"s0\\\":{\\\"spx96\\\":\\\"76500355042434966106117051222\\\",\\\"t\\\":-701,\\\"lst\\\":1754919491,\\\"lsgt\\\":1754730019},\\\"bs\\\":{\\\"h\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"tsa\\\":86400,\\\"lp\\\":[1,255,255,236,0,10,4,67,196,48,23,215,132,0,0,20,6,222,186,96,35,195,70,0,59,154,202,0,0,0,0,0],\\\"hp\\\":\\\"AABaAABaAAAAAAAAAAAAADwAeABkAGQAZAH0BwgBLAAAAAcIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==\\\",\\\"lt\\\":2,\\\"mrtr0\\\":\\\"150000\\\",\\\"trtr0\\\":\\\"200000\\\",\\\"xrtr0\\\":\\\"250000\\\",\\\"mrtr1\\\":\\\"150000\\\",\\\"trtr1\\\":\\\"200000\\\",\\\"xrtr1\\\":\\\"250000\\\",\\\"c0d\\\":18,\\\"c1d\\\":18,\\\"rb0\\\":\\\"127631271511127115511\\\",\\\"rb1\\\":\\\"369862546000473865984\\\",\\\"r0\\\":\\\"434870845891982826083\\\",\\\"r1\\\":\\\"1893126952277387984593\\\",\\\"ib\\\":[128,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1,17,231,107,18,111,238,233,66]},\\\"vsp\\\":{\\\"i\\\":true,\\\"sp0\\\":\\\"1006357720641017152\\\",\\\"sp1\\\":\\\"1000523215210075409\\\"},\\\"bt\\\":1754730020}\"}","staticExtra":"{\"0x0\":[true,false],\"fee\":0,\"tS\":1,\"hooks\":\"0x000052423c1db6b7ff8641b85a7eefc7b2791888\",\"uR\":\"0xef740bf23acae26f6492b10de645d6b98dc8eaf3\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":24171169}`

	assert.NoError(t, json.Unmarshal([]byte(poolData), &p))

	pSim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDUnichain)
	assert.NoError(t, err)

	got, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x7dcc39b4d1c53cb31e1abc0e358b43987fef80f7",
			Amount: big.NewInt(20951525654502637),
		},
		TokenOut: "0x4200000000000000000000000000000000000006",
	})
	assert.NoError(t, err)
	assert.Equal(t, "22470297792091537", got.TokenAmountOut.Amount.String())

	got, err = pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x4200000000000000000000000000000000000006",
			Amount: big.NewInt(1e12),
		},
		TokenOut: "0x7dcc39b4d1c53cb31e1abc0e358b43987fef80f7",
	})
	assert.NoError(t, err)
	assert.Equal(t, "932241970076", got.TokenAmountOut.Amount.String())
}

func Test_Quoter(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://unichain.drpc.org").
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))

	sender := common.HexToAddress("0x1f3134C3f3f8AdD904B9635acBeFC0eA0D0E1ffC")

	poolKey := uniswapv4.PoolKey{
		Currency0:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Currency1:   common.HexToAddress("0x7dcc39b4d1c53cb31e1abc0e358b43987fef80f7"),
		Fee:         big.NewInt(0),
		TickSpacing: big.NewInt(1),
		Hooks:       common.HexToAddress("0x000052423c1db6b7ff8641b85a7eefc7b2791888"),
	}

	swapParams := SwapParams{
		ZeroForOne:        true,
		AmountSpecified:   big.NewInt(-1e12),
		SqrtPriceLimitX96: minPriceLimit,
	}

	var swapResult SwapResult
	_, err := rpcClient.R().AddCall(&ethrpc.Call{
		ABI:    bunniQuoterABI,
		Target: "0x00000000E15009D51C6d57f7164f4Ed4996ae55C",
		Method: "quoteSwap",
		Params: []interface{}{sender, poolKey, swapParams},
	}, []interface{}{&swapResult}).Call()

	assert.NoError(t, err)
	assert.Greater(t, swapResult.OutputAmount.Int64(), int64(0))
}
