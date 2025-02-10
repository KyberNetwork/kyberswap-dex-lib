package ambient_test

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	rpcURL           = "http://localhost:8545"
	multicallAddress = "0x5ba1e12693dc8f9c48aad8770482f4739beed696" // UniswapV3: Multicall 2
)

func TestPoolTracker(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	cfg := &ambient.Config{
		DexID:                    ambient.DexTypeAmbient,
		PoolIdx:                  big.NewInt(420),
		NativeTokenAddress:       "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		QueryContractAddress:     "0xCA00926b6190c2C59336E73F02569c356d7B6b56",
		SwapDexContractAddress:   "0xAaAaAAAaA24eEeb8d57D431224f73832bC34f688",
		MulticallContractAddress: multicallAddress,
	}

	client := ethrpc.New(rpcURL)
	client.SetMulticallContract(common.HexToAddress(multicallAddress))
	tracker, err := ambient.NewPoolTracker(cfg, client)
	require.NoError(t, err)

	encodedPoolEntity := `{
  "address": "0xAaAaAAAaA24eEeb8d57D431224f73832bC34f688",
  "exchange": "ambient",
  "type": "ambient",
  "reserves": [
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0",
    "0"
  ],
  "tokens": [
    {
      "address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
      "swappable": true
    },
    {
      "address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
      "swappable": true
    },
    {
      "address": "0x0f2d719407fdbeff09d87557abb7232601fd9f29",
      "swappable": true
    },
    {
      "address": "0x4e3fbd56cd56c3e72c1403e103b45db9da5b9d2b",
      "swappable": true
    },
    {
      "address": "0xd533a949740bb3306d119cc777fa900ba034cd52",
      "swappable": true
    },
    {
      "address": "0x6982508145454ce325ddbe47a25d4ec3d2311933",
      "swappable": true
    },
    {
      "address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
      "swappable": true
    },
    {
      "address": "0x6b175474e89094c44da98b954eedeac495271d0f",
      "swappable": true
    },
    {
      "address": "0x64aa3364f17a4d01c6f1751fd97c2bd3d7e7f1d5",
      "swappable": true
    },
    {
      "address": "0x5f98805a4e8be255a32880fdec7f6728c6568ba0",
      "swappable": true
    },
    {
      "address": "0x03ab458634910aad20ef5f1c8ee96f1d6ac54919",
      "swappable": true
    },
    {
      "address": "0x5a98fcbea516cf06857215779fd812ca3bef1b32",
      "swappable": true
    },
    {
      "address": "0xf344b01da08b142d2466dae9e47e333f22e64588",
      "swappable": true
    },
    {
      "address": "0x853d955acef822db058eb8505911ed77f175b99e",
      "swappable": true
    },
    {
      "address": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
      "swappable": true
    },
    {
      "address": "0xdef1ca1fb7fbcdc777520aa7f396b4e015f497ab",
      "swappable": true
    },
    {
      "address": "0x72e4f9f808c49a2a61de9c5896298920dc4eeea9",
      "swappable": true
    },
    {
      "address": "0x68648580d1fc22c79f7fbfcbc4ed0495dca8f1f9",
      "swappable": true
    },
    {
      "address": "0x320623b8e4ff03373931769a31fc52a4e78b5d70",
      "swappable": true
    },
    {
      "address": "0xbbbbca6a901c926f240b89eacb641d8aec7aeafd",
      "swappable": true
    },
    {
      "address": "0x18aaa7115705e8be94bffebde57af9bfc265b998",
      "swappable": true
    },
    {
      "address": "0x152649ea73beab28c5b49b26eb48f7ead6d4c898",
      "swappable": true
    },
    {
      "address": "0xf951e335afb289353dc249e82926178eac7ded78",
      "swappable": true
    },
    {
      "address": "0xe45dfc26215312edc131e34ea9299fbca53275ca",
      "swappable": true
    },
    {
      "address": "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
      "swappable": true
    },
    {
      "address": "0x16cda4028e9e872a38acb903176719299beaed87",
      "swappable": true
    },
    {
      "address": "0x9d65ff81a3c488d585bbfb0bfe3c7707c7917f54",
      "swappable": true
    },
    {
      "address": "0x3aada3e213abf8529606924d8d1c55cbdc70bf74",
      "swappable": true
    },
    {
      "address": "0xbaac2b4491727d78d2b78815144570b9f2fe8899",
      "swappable": true
    },
    {
      "address": "0xe60779cc1b2c1d0580611c526a8df0e3f870ec48",
      "swappable": true
    },
    {
      "address": "0x2890df158d76e584877a1d17a85fea3aeeb85aa6",
      "swappable": true
    },
    {
      "address": "0x0d438f3b5175bebc262bf23753c1e53d03432bde",
      "swappable": true
    },
    {
      "address": "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
      "swappable": true
    },
    {
      "address": "0x61e90a50137e1f645c9ef4a0d3a4f01477738406",
      "swappable": true
    },
    {
      "address": "0xa8b919680258d369114910511cc87595aec0be6d",
      "swappable": true
    },
    {
      "address": "0xdbdb4d16eda451d0503b854cf79d55697f90c8df",
      "swappable": true
    },
    {
      "address": "0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0",
      "swappable": true
    },
    {
      "address": "0x183015a9ba6ff60230fdeadc3f43b3d788b13e21",
      "swappable": true
    },
    {
      "address": "0xd9fcd98c322942075a5c3860693e9f4f03aae07b",
      "swappable": true
    },
    {
      "address": "0xd807f7e2818db8eda0d28b5be74866338eaedb86",
      "swappable": true
    },
    {
      "address": "0x5283d291dbcf85356a21ba090e6db59121208b44",
      "swappable": true
    },
    {
      "address": "0x046eee2cc3188071c02bfc1745a6b17c656e3f3d",
      "swappable": true
    },
    {
      "address": "0x6e2a43be0b1d33b726f0ca3b8de60b3482b8b050",
      "swappable": true
    },
    {
      "address": "0x9e32b13ce7f2e80a01932b42553652e053d6ed8e",
      "swappable": true
    },
    {
      "address": "0x514910771af9ca656af840dff83e8264ecf986ca",
      "swappable": true
    },
    {
      "address": "0xb1f1f47061a7be15c69f378cb3f69423bd58f2f8",
      "swappable": true
    },
    {
      "address": "0x9559aaa82d9649c7a7b220e7c461d2e74c9a3593",
      "swappable": true
    },
    {
      "address": "0xf819d9cb1c2a819fd991781a822de3ca8607c3c9",
      "swappable": true
    },
    {
      "address": "0xa3c31927a092bd54eb9a0b5dfe01d9db5028bd4f",
      "swappable": true
    },
    {
      "address": "0xc5102fe9359fd9a28f877a67e36b0f050d81a3cc",
      "swappable": true
    },
    {
      "address": "0x6123b0049f904d730db3c36a31167d9d4121fa6b",
      "swappable": true
    },
    {
      "address": "0x3c3a81e81dc49a522a592e7622a7e711c06bf354",
      "swappable": true
    },
    {
      "address": "0xae78736cd615f374d3085123a210448e74fc6393",
      "swappable": true
    },
    {
      "address": "0xbe9895146f7af43049ca1c1ae358b0541ea49704",
      "swappable": true
    },
    {
      "address": "0xc011a73ee8576fb46f5e1c5751ca3b9fe0af2a6f",
      "swappable": true
    },
    {
      "address": "0x78a0a62fba6fb21a83fe8a3433d44c73a4017a6f",
      "swappable": true
    },
    {
      "address": "0x0ab87046fbb341d058f17cbc4c1133f25a20a52f",
      "swappable": true
    },
    {
      "address": "0x6db32ba9c42117837c269ae35b87db2f197bb861",
      "swappable": true
    },
    {
      "address": "0x9ae380f0272e2162340a5bb646c354271c0f5cfc",
      "swappable": true
    },
    {
      "address": "0xa0b73e1ff0b80914ab6fe0444e65848c4c34450b",
      "swappable": true
    },
    {
      "address": "0x549020a9cb845220d66d3e9c6d9f9ef61c981102",
      "swappable": true
    },
    {
      "address": "0x1e2c4fb7ede391d116e6b41cd0608260e8801d59",
      "swappable": true
    },
    {
      "address": "0xb23d80f5fefcddaa212212f028021b41ded428cf",
      "swappable": true
    },
    {
      "address": "0x04c17b9d3b29a78f7bd062a57cf44fc633e71f85",
      "swappable": true
    },
    {
      "address": "0xb5b1b659da79a2507c27aad509f15b4874edc0cc",
      "swappable": true
    },
    {
      "address": "0xfa3e941d1f6b7b10ed84a0c211bfa8aee907965e",
      "swappable": true
    },
    {
      "address": "0xdffa3a7f5b40789c7a437dbe7b31b47f9b08fe75",
      "swappable": true
    },
    {
      "address": "0x5f64ab1544d28732f0a24f4713c2c8ec0da089f0",
      "swappable": true
    },
    {
      "address": "0x6de037ef9ad2725eb40118bb1702ebb27e4aeb24",
      "swappable": true
    },
    {
      "address": "0x53020f42f6da51b50cf6e23e45266ef223122376",
      "swappable": true
    },
    {
      "address": "0x5516ac1aaca7bb2fd5b7bdde1549ef1ea242953d",
      "swappable": true
    },
    {
      "address": "0xfca59cd816ab1ead66534d82bc21e7515ce441cf",
      "swappable": true
    },
    {
      "address": "0xb3207935ff56120f3499e8ad08461dd403bf16b8",
      "swappable": true
    },
    {
      "address": "0x4d224452801aced8b2f0aebe155379bb5d594381",
      "swappable": true
    },
    {
      "address": "0xd33526068d116ce69f19a9ee46f0bd304f21a51f",
      "swappable": true
    },
    {
      "address": "0xaf5191b0de278c7286d6c7cc6ab6bb8a73ba2cd6",
      "swappable": true
    },
    {
      "address": "0xfae103dc9cf190ed75350761e95403b7b8afa6c0",
      "swappable": true
    },
    {
      "address": "0xf6d2224916ddfbbab6e6bd0d1b7034f4ae0cab18",
      "swappable": true
    },
    {
      "address": "0xe92344b4edf545f3209094b192e46600a19e7c2d",
      "swappable": true
    },
    {
      "address": "0xba3335588d9403515223f109edc4eb7269a9ab5d",
      "swappable": true
    },
    {
      "address": "0xc71b5f631354be6853efe9c3ab6b9590f8302e81",
      "swappable": true
    },
    {
      "address": "0xbdab72602e9ad40fc6a6852caf43258113b8f7a5",
      "swappable": true
    },
    {
      "address": "0xfe0c30065b384f05761f15d0cc899d4f9f9cc0eb",
      "swappable": true
    },
    {
      "address": "0xa1290d69c65a6fe4df752f95823fae25cb99e5a7",
      "swappable": true
    },
    {
      "address": "0x1a4b46696b2bb4794eb3d4c26f1c55f9170fa4c5",
      "swappable": true
    },
    {
      "address": "0x8881562783028f5c1bcb985d2283d5e170d88888",
      "swappable": true
    },
    {
      "address": "0x808507121b80c02388fad14726482e061b8da827",
      "swappable": true
    },
    {
      "address": "0x9334504d513b68f94f18514c71fb73a472e67e7f",
      "swappable": true
    },
    {
      "address": "0xc5f0f7b66764f6ec8c8dff7ba683102295e16409",
      "swappable": true
    },
    {
      "address": "0x7039cd6d7966672f194e8139074c3d5c4e6dcf65",
      "swappable": true
    },
    {
      "address": "0x7122985656e38bdc0302db86685bb972b145bd3c",
      "swappable": true
    }
  ],
  "extra": "{\"tokenPairs\":{\"0x0000000000000000000000000000000000000000:0x03ab458634910aad20ef5f1c8ee96f1d6ac54919\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x046eee2cc3188071c02bfc1745a6b17c656e3f3d\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x0d438f3b5175bebc262bf23753c1e53d03432bde\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x0f2d719407fdbeff09d87557abb7232601fd9f29\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x152649ea73beab28c5b49b26eb48f7ead6d4c898\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x16cda4028e9e872a38acb903176719299beaed87\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x183015a9ba6ff60230fdeadc3f43b3d788b13e21\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x18aaa7115705e8be94bffebde57af9bfc265b998\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x1a4b46696b2bb4794eb3d4c26f1c55f9170fa4c5\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x1e2c4fb7ede391d116e6b41cd0608260e8801d59\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x2890df158d76e584877a1d17a85fea3aeeb85aa6\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x3aada3e213abf8529606924d8d1c55cbdc70bf74\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x3c3a81e81dc49a522a592e7622a7e711c06bf354\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x4d224452801aced8b2f0aebe155379bb5d594381\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x4e3fbd56cd56c3e72c1403e103b45db9da5b9d2b\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x514910771af9ca656af840dff83e8264ecf986ca\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x5283d291dbcf85356a21ba090e6db59121208b44\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x53020f42f6da51b50cf6e23e45266ef223122376\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x549020a9cb845220d66d3e9c6d9f9ef61c981102\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x5516ac1aaca7bb2fd5b7bdde1549ef1ea242953d\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x5a98fcbea516cf06857215779fd812ca3bef1b32\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x5f64ab1544d28732f0a24f4713c2c8ec0da089f0\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x5f98805a4e8be255a32880fdec7f6728c6568ba0\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x6123b0049f904d730db3c36a31167d9d4121fa6b\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x61e90a50137e1f645c9ef4a0d3a4f01477738406\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x68648580d1fc22c79f7fbfcbc4ed0495dca8f1f9\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x6982508145454ce325ddbe47a25d4ec3d2311933\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x6b175474e89094c44da98b954eedeac495271d0f\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x6db32ba9c42117837c269ae35b87db2f197bb861\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x6de037ef9ad2725eb40118bb1702ebb27e4aeb24\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x6e2a43be0b1d33b726f0ca3b8de60b3482b8b050\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x7039cd6d7966672f194e8139074c3d5c4e6dcf65\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x7122985656e38bdc0302db86685bb972b145bd3c\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x72e4f9f808c49a2a61de9c5896298920dc4eeea9\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x808507121b80c02388fad14726482e061b8da827\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x853d955acef822db058eb8505911ed77f175b99e\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x8881562783028f5c1bcb985d2283d5e170d88888\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x9334504d513b68f94f18514c71fb73a472e67e7f\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x9559aaa82d9649c7a7b220e7c461d2e74c9a3593\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x9ae380f0272e2162340a5bb646c354271c0f5cfc\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x9d65ff81a3c488d585bbfb0bfe3c7707c7917f54\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x9e32b13ce7f2e80a01932b42553652e053d6ed8e\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xa0b73e1ff0b80914ab6fe0444e65848c4c34450b\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xa1290d69c65a6fe4df752f95823fae25cb99e5a7\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xa3c31927a092bd54eb9a0b5dfe01d9db5028bd4f\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xae78736cd615f374d3085123a210448e74fc6393\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xaf5191b0de278c7286d6c7cc6ab6bb8a73ba2cd6\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xb1f1f47061a7be15c69f378cb3f69423bd58f2f8\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xb23d80f5fefcddaa212212f028021b41ded428cf\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xb3207935ff56120f3499e8ad08461dd403bf16b8\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xbaac2b4491727d78d2b78815144570b9f2fe8899\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xbdab72602e9ad40fc6a6852caf43258113b8f7a5\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xc011a73ee8576fb46f5e1c5751ca3b9fe0af2a6f\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xc5102fe9359fd9a28f877a67e36b0f050d81a3cc\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xc5f0f7b66764f6ec8c8dff7ba683102295e16409\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xc71b5f631354be6853efe9c3ab6b9590f8302e81\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xd33526068d116ce69f19a9ee46f0bd304f21a51f\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xd533a949740bb3306d119cc777fa900ba034cd52\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xd807f7e2818db8eda0d28b5be74866338eaedb86\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xdac17f958d2ee523a2206206994597c13d831ec7\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xdbdb4d16eda451d0503b854cf79d55697f90c8df\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xdffa3a7f5b40789c7a437dbe7b31b47f9b08fe75\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xe45dfc26215312edc131e34ea9299fbca53275ca\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xe60779cc1b2c1d0580611c526a8df0e3f870ec48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xe92344b4edf545f3209094b192e46600a19e7c2d\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xf344b01da08b142d2466dae9e47e333f22e64588\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xf6d2224916ddfbbab6e6bd0d1b7034f4ae0cab18\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xf819d9cb1c2a819fd991781a822de3ca8607c3c9\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xf951e335afb289353dc249e82926178eac7ded78\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xfa3e941d1f6b7b10ed84a0c211bfa8aee907965e\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xfae103dc9cf190ed75350761e95403b7b8afa6c0\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0000000000000000000000000000000000000000:0xfca59cd816ab1ead66534d82bc21e7515ce441cf\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x03ab458634910aad20ef5f1c8ee96f1d6ac54919:0x6b175474e89094c44da98b954eedeac495271d0f\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x046eee2cc3188071c02bfc1745a6b17c656e3f3d:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x04c17b9d3b29a78f7bd062a57cf44fc633e71f85:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x0ab87046fbb341d058f17cbc4c1133f25a20a52f:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x183015a9ba6ff60230fdeadc3f43b3d788b13e21:0xdac17f958d2ee523a2206206994597c13d831ec7\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599:0xdac17f958d2ee523a2206206994597c13d831ec7\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x320623b8e4ff03373931769a31fc52a4e78b5d70:0xbbbbca6a901c926f240b89eacb641d8aec7aeafd\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x5283d291dbcf85356a21ba090e6db59121208b44:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x5f98805a4e8be255a32880fdec7f6728c6568ba0:0x6b175474e89094c44da98b954eedeac495271d0f\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x64aa3364f17a4d01c6f1751fd97c2bd3d7e7f1d5:0x6b175474e89094c44da98b954eedeac495271d0f\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x6b175474e89094c44da98b954eedeac495271d0f:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x6b175474e89094c44da98b954eedeac495271d0f:0xbaac2b4491727d78d2b78815144570b9f2fe8899\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x6b175474e89094c44da98b954eedeac495271d0f:0xdac17f958d2ee523a2206206994597c13d831ec7\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x72e4f9f808c49a2a61de9c5896298920dc4eeea9:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x78a0a62fba6fb21a83fe8a3433d44c73a4017a6f:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x9d65ff81a3c488d585bbfb0bfe3c7707c7917f54:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2:0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48:0xa8b919680258d369114910511cc87595aec0be6d\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48:0xb5b1b659da79a2507c27aad509f15b4874edc0cc\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48:0xba3335588d9403515223f109edc4eb7269a9ab5d\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48:0xd9fcd98c322942075a5c3860693e9f4f03aae07b\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48:0xdac17f958d2ee523a2206206994597c13d831ec7\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48:0xdef1ca1fb7fbcdc777520aa7f396b4e015f497ab\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48:0xf951e335afb289353dc249e82926178eac7ded78\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48:0xfe0c30065b384f05761f15d0cc899d4f9f9cc0eb\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0xae78736cd615f374d3085123a210448e74fc6393:0xbe9895146f7af43049ca1c1ae358b0541ea49704\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420},\"0xdac17f958d2ee523a2206206994597c13d831ec7:0xf951e335afb289353dc249e82926178eac7ded78\":{\"sqrtPriceX64\":\"\",\"liquidity\":\"\",\"poolIdx\":420}}}",
  "staticExtra": "{\"nativeTokenAddress\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\"}"
}`
	poolEntity := entity.Pool{}
	err = json.Unmarshal([]byte(encodedPoolEntity), &poolEntity)
	require.NoError(t, err)

	pool, err := tracker.GetNewPoolState(context.Background(), poolEntity, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	require.Equal(t, len(poolEntity.Tokens), len(poolEntity.Reserves))
	for _, reserve := range poolEntity.Reserves {
		require.NotEmpty(t, reserve)
	}

	extra := ambient.Extra{}
	err = json.Unmarshal([]byte(pool.Extra), &extra)
	require.NoError(t, err)
	for _, info := range extra.TokenPairs {
		require.NotNil(t, info.Liquidity)
		require.NotNil(t, info.SqrtPriceX64)
	}

	var (
		tokenAddrs          []common.Address
		tokenAddrsFromPairs []common.Address
	)
	for _, token := range poolEntity.Tokens {
		tokenAddrs = append(tokenAddrs, common.HexToAddress(token.Address))
	}
	for pair := range extra.TokenPairs {
		if pair.Base == ambient.NativeTokenPlaceholderAddress {
			pair.Base = common.HexToAddress(cfg.NativeTokenAddress)
		}
		tokenAddrsFromPairs = append(tokenAddrsFromPairs, pair.Base, pair.Quote)
	}
	require.Subsetf(t, tokenAddrs, tokenAddrsFromPairs,
		".Tokens[].Address and .Extra.TokenPairs[].{Base,Quote} must be the same set")
	require.Subsetf(t, tokenAddrsFromPairs, tokenAddrs,
		".Tokens[].Address and .Extra.TokenPairs[].{Base,Quote} must be the same set")
}
