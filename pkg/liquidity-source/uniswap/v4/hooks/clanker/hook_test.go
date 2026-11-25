package clanker

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestStaticFeeHook_Track(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://arbitrum-rpc.kyberswap.com").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	chainID := valueobject.ChainIDArbitrumOne

	sh := NewStaticFeeHook(Clanker)(&uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0xf7aC669593d2D9D01026Fa5B756DD5B4f7aAa8Cc"),
	})

	_, err := sh.Track(context.Background(), &uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{ChainID: chainID},
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0xf7aC669593d2D9D01026Fa5B756DD5B4f7aAa8Cc"),
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

	rpcClient := ethrpc.New("https://arbitrum-rpc.kyberswap.com").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	chainID := valueobject.ChainIDArbitrumOne

	poolStr := `{"address":"0xe2e7c620191449a2228b69231b63f2395f486ca7550de706feddf7ec9b9cd5d7","swapFee":30000,"exchange":"uniswap-v4-clanker","type":"uniswap-v4","timestamp":1753291890,"reserves":["70747613414474548452393626071","48591284065701221020"],"tokens":[{"address":"0x6488cf3cd609f8fe683e96b461e765d98dcedb07","symbol":"🟦","decimals":18,"swappable":true},{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"liquidity\":1854108243979608403116743,\"sqrtPriceX96\":2076361055636673935789906,\"tickSpacing\":200,\"tick\":-211000,\"ticks\":[{\"index\":-211000,\"liquidityGross\":1854108243979608403116743,\"liquidityNet\":1854108243979608403116743},{\"index\":-120000,\"liquidityGross\":1854108243979608403116743,\"liquidityNet\":-1854108243979608403116743}],\"hX\":{\"p\":1000,\"0\":true,\"t\":false}}","staticExtra":"{\"0x0\":[false,false],\"fee\":8388608,\"tS\":200,\"hooks\":\"0xfd213be7883db36e1049dc42f5bd6a0ec66b68cc\",\"uR\":\"0xa51afafe0263b40edaef0df8781ea9aa03e381a3\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":360796346}`
	var pool entity.Pool
	err := json.Unmarshal([]byte(poolStr), &pool)
	require.NoError(t, err)

	dh := NewDynamicFeeHook(Clanker)(&uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{ChainID: chainID},
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0xFd213BE7883db36e1049dC42f5BD6A0ec66B68cC"),
	})

	_, err = dh.Track(context.Background(), &uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{ChainID: chainID},
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0xFd213BE7883db36e1049dC42f5BD6A0ec66B68cC"),
		Pool:        &pool,
	})
	require.NoError(t, err)
}

func Test_CalcAmountOut(t *testing.T) {
	var p entity.Pool
	poolData := `{"address":"0xabb949ef8d1e86c37e2620de318da773273460a843a623c3c05d2b85dc893e77","swapFee":30000,"exchange":"uniswap-v4","type":"uniswap-v4","timestamp":1753702228,"reserves":["83499061346031704444327158001","4265567673400257806"],"tokens":[{"address":"0x109ddc73b46b2f8141880a5573d2b0d2acf10b07","symbol":"neged2.0","decimals":18,"swappable":true},{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"liquidity\":596800550298755887846132,\"sqrtPriceX96\":566274760763156353397804,\"tickSpacing\":200,\"tick\":-236988,\"ticks\":[{\"index\":-237400,\"liquidityGross\":596800550298755887846132,\"liquidityNet\":596800550298755887846132},{\"index\":-120000,\"liquidityGross\":596800550298755887846132,\"liquidityNet\":-596800550298755887846132}],\"hX\":{\"p\":6000,\"c\":30000,\"f\":30000,\"0\":true}}","staticExtra":"{\"0x0\":[false,false],\"fee\":8388608,\"tS\":200,\"hooks\":\"0xdd5eeaff7bd481ad55db083062b13a3cdf0a68cc\",\"uR\":\"0x6ff5693b99212da76ad316178a184ab56d299b43\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":33456440}`
	assert.NoError(t, json.Unmarshal([]byte(poolData), &p))

	pSim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDBase)
	assert.NoError(t, err)

	got, err := testutil.CalcAmountOutNative(pSim,
		pool.TokenAmount{
			Token:  "0x4200000000000000000000000000000000000006",
			Amount: big.NewInt(1000000000000000),
		},
		"0x109ddc73b46b2f8141880a5573d2b0d2acf10b07", nil)

	assert.NoError(t, err)
	assert.Equal(t, "18870367192095562543977004", got.TokenAmountOut.Amount.String())
}

func Test_CalcAmountIn(t *testing.T) {
	var p entity.Pool
	poolData := `{"address":"0xabb949ef8d1e86c37e2620de318da773273460a843a623c3c05d2b85dc893e77","swapFee":30000,"exchange":"uniswap-v4","type":"uniswap-v4","timestamp":1753702228,"reserves":["83499061346031704444327158001","4265567673400257806"],"tokens":[{"address":"0x109ddc73b46b2f8141880a5573d2b0d2acf10b07","symbol":"neged2.0","decimals":18,"swappable":true},{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"liquidity\":596800550298755887846132,\"sqrtPriceX96\":566274760763156353397804,\"tickSpacing\":200,\"tick\":-236988,\"ticks\":[{\"index\":-237400,\"liquidityGross\":596800550298755887846132,\"liquidityNet\":596800550298755887846132},{\"index\":-120000,\"liquidityGross\":596800550298755887846132,\"liquidityNet\":-596800550298755887846132}],\"hX\":{\"p\":6000,\"c\":30000,\"f\":30000,\"t\":true}}","staticExtra":"{\"0x0\":[false,false],\"fee\":8388608,\"tS\":200,\"hooks\":\"0xdd5eeaff7bd481ad55db083062b13a3cdf0a68cc\",\"uR\":\"0x6ff5693b99212da76ad316178a184ab56d299b43\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":33456440}`
	assert.NoError(t, json.Unmarshal([]byte(poolData), &p))

	pSim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDBase)
	assert.NoError(t, err)

	testutil.TestCalcAmountInIPoolSim(t, pSim, goldenStaticFee)
}

// goldenAsymFee is the legacy CalcAmountIn output for a static hook with ClankerFee (c) != PairedFee (f), so the
// swap direction picks which fee applies (28 cases). goldenStaticFee's fixture has c == f and cannot tell them apart.
var goldenAsymFee = map[int]map[int]map[string]string{
	0: {
		1: {
			"1":                            "20513134276",
			"1000":                         "20513109861798",
			"1000000":                      "20513109798140181",
			"1000000000":                   "20513109802872365298",
			"1000000000000":                "20513114607063257839907",
			"1000000000000000":             "20517919924521408644527008",
			"3000000000000000":             "61582640844547651602958169",
			"100000000000000000":           "",
			"500000000000000000":           "",
			"1000000000000000000":          "",
			"2000000000000000000":          "",
			"10000000000000000000":         "",
			"100000000000000000000000000":  "",
			"1000000000000000000000000000": "",
		},
	},
	1: {
		0: {
			"1":                            "2",
			"1000":                         "2",
			"1000000":                      "2",
			"1000000000":                   "2",
			"1000000000000":                "53",
			"1000000000000000":             "51914",
			"3000000000000000":             "155739",
			"100000000000000000":           "5191271",
			"500000000000000000":           "25956354",
			"1000000000000000000":          "51912707",
			"2000000000000000000":          "103825413",
			"10000000000000000000":         "519127057",
			"100000000000000000000000000":  "5197532792533735",
			"1000000000000000000000000000": "52545802289633989",
		},
	},
}

const asymPoolData = `{"address":"0xabb949ef8d1e86c37e2620de318da773273460a843a623c3c05d2b85dc893e77","swapFee":30000,"exchange":"uniswap-v4","type":"uniswap-v4","timestamp":1753702228,"reserves":["83499061346031704444327158001","4265567673400257806"],"tokens":[{"address":"0x109ddc73b46b2f8141880a5573d2b0d2acf10b07","symbol":"neged2.0","decimals":18,"swappable":true},{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"liquidity\":596800550298755887846132,\"sqrtPriceX96\":566274760763156353397804,\"tickSpacing\":200,\"tick\":-236988,\"ticks\":[{\"index\":-237400,\"liquidityGross\":596800550298755887846132,\"liquidityNet\":596800550298755887846132},{\"index\":-120000,\"liquidityGross\":596800550298755887846132,\"liquidityNet\":-596800550298755887846132}],\"hX\":{\"p\":6000,\"c\":10000,\"f\":40000,\"t\":true}}","staticExtra":"{\"0x0\":[false,false],\"fee\":8388608,\"tS\":200,\"hooks\":\"0xdd5eeaff7bd481ad55db083062b13a3cdf0a68cc\",\"uR\":\"0x6ff5693b99212da76ad316178a184ab56d299b43\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":33456440}`

func Test_CalcAmountIn_AsymStaticFee(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(asymPoolData), &p))

	pSim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDBase)
	require.NoError(t, err)

	testutil.TestCalcAmountInIPoolSim(t, pSim, goldenAsymFee)
}

// goldenStaticFee is the legacy CalcAmountIn output over the amountOut grid the randomized sweep covered (28 cases).
var goldenStaticFee = map[int]map[int]map[string]string{
	0: {
		1: {
			"1":                            "20301658665",
			"1000":                         "20301634502397",
			"1000000":                      "20301634439396468",
			"1000000000":                   "20301634444079866686",
			"1000000000000":                "20301639198743018068361",
			"1000000000000000":             "20306394976845930204892709",
			"3000000000000000":             "60947768258521387153443136",
			"100000000000000000":           "",
			"500000000000000000":           "",
			"1000000000000000000":          "",
			"2000000000000000000":          "",
			"10000000000000000000":         "",
			"100000000000000000000000000":  "",
			"1000000000000000000000000000": "",
		},
	},
	1: {
		0: {
			"1":                            "2",
			"1000":                         "2",
			"1000000":                      "2",
			"1000000000":                   "2",
			"1000000000000":                "54",
			"1000000000000000":             "52984",
			"3000000000000000":             "158950",
			"100000000000000000":           "5298308",
			"500000000000000000":           "26491537",
			"1000000000000000000":          "52983072",
			"2000000000000000000":          "105966143",
			"10000000000000000000":         "529830708",
			"100000000000000000000000000":  "5304698417122059",
			"1000000000000000000000000000": "53629220893543969",
		},
	},
}

const dynamicFeePoolData = `{"address":"0xe2e7c620191449a2228b69231b63f2395f486ca7550de706feddf7ec9b9cd5d7","swapFee":30000,"exchange":"uniswap-v4-clanker","type":"uniswap-v4","timestamp":1753291890,"reserves":["70747613414474548452393626071","48591284065701221020"],"tokens":[{"address":"0x6488cf3cd609f8fe683e96b461e765d98dcedb07","symbol":"🟦","decimals":18,"swappable":true},{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"hX\":{\"p\":2000,\"f\":{\"r\":-210992,\"t\":-210992,\"s\":\"1753071293\",\"l\":\"1753071293\",\"p\":\"5\"},\"c\":{\"b\":5000,\"m\":50000,\"r\":\"30\",\"p\":\"120\",\"f\":200,\"c\":\"500000000\",\"d\":\"7500\"},\"0\":true,\"t\":true},\"liquidity\":1854108243979608403116743,\"sqrtPriceX96\":2076361055636673935789906,\"tick\":-211000,\"tickSpacing\":200,\"ticks\":[{\"index\":-211000,\"liquidityGross\":1854108243979608403116743,\"liquidityNet\":1854108243979608403116743},{\"index\":-120000,\"liquidityGross\":1854108243979608403116743,\"liquidityNet\":-1854108243979608403116743}]}","staticExtra":"{\"0x0\":[false,false],\"fee\":8388608,\"tS\":200,\"hooks\":\"0xfd213be7883db36e1049dc42f5bd6a0ec66b68cc\",\"uR\":\"0xa51afafe0263b40edaef0df8781ea9aa03e381a3\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":360796346}`

// CalcAmountIn reverses exact-in, so the dynamic fee hook must invert the exact-in protocol fees in both directions.
func Test_CalcAmountIn_DynamicFee(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(dynamicFeePoolData), &p))

	pSim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDArbitrumOne)
	require.NoError(t, err)

	testutil.TestCalcAmountInIPoolSim(t, pSim, goldenDynamicFee)
}

// goldenDynamicFee is the legacy CalcAmountIn output over the same grid (28 cases). The dynamic fee reads
// wall-clock time, so these hold while the fee vars stay in the same reset window as the fixture.
var goldenDynamicFee = map[int]map[int]map[string]string{
	0: {
		1: {
			"1":                            "1463319158",
			"1000":                         "1464753095399",
			"1000000":                      "1464754526699339",
			"1000000000":                   "1464754528163865452",
			"1000000000000":                "1464754558309698033254",
			"1000000000000000":             "",
			"3000000000000000":             "",
			"100000000000000000":           "",
			"500000000000000000":           "",
			"1000000000000000000":          "",
			"2000000000000000000":          "",
			"10000000000000000000":         "",
			"100000000000000000000000000":  "",
			"1000000000000000000000000000": "",
		},
	},
	1: {
		0: {
			"1":                            "2",
			"1000":                         "2",
			"1000000":                      "2",
			"1000000000":                   "2",
			"1000000000000":                "691",
			"1000000000000000":             "690968",
			"3000000000000000":             "2072903",
			"100000000000000000":           "69096744",
			"500000000000000000":           "345483717",
			"1000000000000000000":          "690967434",
			"2000000000000000000":          "1381934867",
			"10000000000000000000":         "6909674331",
			"100000000000000000000000000":  "69197744234868890",
			"1000000000000000000000000000": "704292432293261559",
		},
	},
}

// Quotes run concurrently on shared hook and pool state, so a quote must not write it. Only a swap
// applied through UpdateBalance moves the fee vars, as the on-chain hook writes them on every swap.
func TestDynamicFeeHook_QuoteLeavesStateUntouched(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(dynamicFeePoolData), &p))
	var extra uniswapv4.ExtraU256
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	h := NewDynamicFeeHook(Clanker)(&uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: valueobject.ChainIDArbitrumOne},
		Pool:      &p,
		HookExtra: uniswapv4.HookExtra(extra.HookExtra),
	}).(*DynamicFeeHook)
	fVars, protocolFee, lpFee := *h.PoolFVars, new(big.Int).Set(h.ProtocolFee), h.poolSim.V3Pool.Fee

	params := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: uint256.NewInt(1e18)}
	res, err := h.BeforeSwap(params)
	require.NoError(t, err)
	_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: params, AmountIn: params.AmountSpecified,
		AmountOut: uint256.NewInt(1e9), BeforeSwapResult: res})
	require.NoError(t, err)

	require.Equal(t, fVars, *h.PoolFVars)
	require.Equal(t, protocolFee, h.ProtocolFee)
	require.Equal(t, lpFee, h.poolSim.V3Pool.Fee)

	h.UpdateBalance(res.SwapInfo)
	require.NotEqual(t, fVars.LastSwapTimestamp, h.PoolFVars.LastSwapTimestamp, "a swap must update the fee vars")
}
