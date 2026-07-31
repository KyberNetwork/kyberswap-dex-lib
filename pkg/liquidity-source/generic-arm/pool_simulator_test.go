package genericarm

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func getPool() *PoolSimulator {
	var poolE entity.Pool
	_ = json.Unmarshal([]byte("{\"address\":\"0x85b78aca6deae198fbf201c82daf6ca21942acc6\",\"exchange\":\"lidoarm\",\"type\":\"lidoarm\",\"timestamp\":1749541899,\"reserves\":[\"3240609312343444932413\",\"104337404939163039097\"],\"tokens\":[{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"symbol\":\"WETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xae7ab96520de3a18e5e111b5eaab095312d7fe84\",\"symbol\":\"stETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"r0\\\":\\\"1000001576063044561835090408175422814\\\",\\\"r1\\\":\\\"999898426597041524878150000000000000\\\",\\\"ps\\\":\\\"1000000000000000000000000000000000000\\\",\\\"wq\\\":\\\"8824843694584167917191\\\",\\\"wc\\\":\\\"8816768469433561587106\\\",\\\"la\\\":\\\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\\\",\\\"swapType\\\":3,\\\"armType\\\":1,\\\"hasWithdrawalQueue\\\":true}\"}"), &poolE)
	pool, _ := NewPoolSimulator(poolE)
	return pool
}

func TestPoolSimulator01(t *testing.T) {
	p := getPool()
	// https://etherscan.io/tx/0xa0656206651d095e2bf678225ad55a860481a3467fb61c59fe0d41f635f597ec
	// r0 0x0000000000000000000000000000000000c097e26051d2821a7698803345cd5e
	amountOut, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: big.NewInt(1824827840682786465),
			},
			TokenOut: "0xae7ab96520de3a18e5e111b5eaab095312d7fe84",
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(1824830716726508852), amountOut.TokenAmountOut.Amount)
}

func TestPoolSimulator10(t *testing.T) {
	p := getPool()
	// https://etherscan.io/tx/0x332289850d386bef8bc8a90fb6ec31519b6a64a0756e442f2546dc51db87fb32
	// r1 0x0000000000000000000000000000000000c092cc726b59717c60bc6e06d26000
	amountOut, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xae7ab96520de3a18e5e111b5eaab095312d7fe84",
				Amount: big.NewInt(5019524698851081465),
			},
			TokenOut: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(5019014848646185045), amountOut.TokenAmountOut.Amount)
}

// getEthenaARMPool builds a Pricable4626 pool (USDe/sUSDe, 1 base asset) from a live on-chain snapshot
// of 0xCEDa2d856238aA0D12f6329de20B9115f07C366d taken after the ARM's upgrade to the shared AbstractARM
// contract: no more token0/token1/traderate0/1/baseAsset/PRICE_SCALE, buyPrice/sellPrice come from
// baseAssetConfigs(), and sUSDe's conversion rate is snapshotted from its (non-pegged) adapter.
func getEthenaARMPool() *PoolSimulator {
	var poolE entity.Pool
	_ = json.Unmarshal([]byte(`{
		"address":"0xceda2d856238aa0d12f6329de20b9115f07c366d",
		"exchange":"ethenaarm",
		"type":"ethenaarm",
		"timestamp":1749541899,
		"reserves":["9719573042480775686418","51606389896075379654910"],
		"tokens":[
			{"address":"0x4c9edd5852cd905f086c759e8383e09bff1e68b3","symbol":"USDe","decimals":18,"swappable":true},
			{"address":"0x9d39a5de30e57443bff2a8307a4256c8797a3497","symbol":"sUSDe","decimals":18,"swappable":true}
		],
		"extra":"{\"la\":\"0x4c9edd5852cd905f086c759e8383e09bff1e68b3\",\"lad\":18,\"ps\":\"1000000000000000000000000000000000000\",\"swapType\":3,\"armType\":2,\"hasWithdrawalQueue\":false,\"bas\":[{\"d\":18,\"pg\":false,\"bp\":\"999600000000000000000000000000000000\",\"sp\":\"999990000000000000000000000000000000\",\"blr\":\"340282366920938463463374607431768211455\",\"slr\":\"340282366920938463463374607431768211455\",\"cra\":\"1238245972405699526\",\"crs\":\"807593985593324023\"}]}"
	}`), &poolE)
	pool, _ := NewPoolSimulator(poolE)
	return pool
}

func TestPoolSimulatorPricable4626_USDeToSUSDe(t *testing.T) {
	p := getEthenaARMPool()
	amountOut, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x4c9edd5852cd905f086c759e8383e09bff1e68b3",
				Amount: bignumber.NewBig("1000000000000000000000"),
			},
			TokenOut: "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, bignumber.NewBig("807602061613940162401"), amountOut.TokenAmountOut.Amount)
}

func TestPoolSimulatorPricable4626_SUSDeToUSDe(t *testing.T) {
	p := getEthenaARMPool()
	amountOut, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
				Amount: bignumber.NewBig("1000000000000000000000"),
			},
			TokenOut: "0x4c9edd5852cd905f086c759e8383e09bff1e68b3",
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, bignumber.NewBig("1237750674016737246189"), amountOut.TokenAmountOut.Amount)
}

// getWethARMPool builds a Pricable4626 pool with multiple base assets (WETH vs stETH/wstETH), modeled
// after 0x68025A4615407993A680102b08a23A61D11C657C (WETH_ARM), which reverted on listing under the
// single-base-asset assumption: stETH is pegged (1:1 decimal scale, no adapter call), wstETH is not
// (adapter conversion rate snapshot, cra/crs from live wstETH.getStETHByWstETH/getWstETHByStETH(1e18)).
func getWethARMPool() *PoolSimulator {
	var poolE entity.Pool
	_ = json.Unmarshal([]byte(`{
		"address":"0x68025a4615407993a680102b08a23a61d11c657c",
		"exchange":"wetharm",
		"type":"wetharm",
		"timestamp":1749541899,
		"reserves":["10000000000000000000000","500000000000000000000","500000000000000000000"],
		"tokens":[
			{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},
			{"address":"0xae7ab96520de3a18e5e111b5eaab095312d7fe84","symbol":"stETH","decimals":18,"swappable":true},
			{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0","symbol":"wstETH","decimals":18,"swappable":true}
		],
		"extra":"{\"la\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"lad\":18,\"ps\":\"1000000000000000000000000000000000000\",\"swapType\":3,\"armType\":2,\"hasWithdrawalQueue\":false,\"bas\":[{\"d\":18,\"pg\":true,\"bp\":\"999700000000000000000000000000000000\",\"sp\":\"1000000000000000000000000000000000000\",\"blr\":\"340282366920938463463374607431768211455\",\"slr\":\"340282366920938463463374607431768211455\"},{\"d\":18,\"pg\":false,\"bp\":\"999700000000000000000000000000000000\",\"sp\":\"1000000000000000000000000000000000000\",\"blr\":\"340282366920938463463374607431768211455\",\"slr\":\"340282366920938463463374607431768211455\",\"cra\":\"1240026075893070633\",\"crs\":\"806434654432405296\"}]}"
	}`), &poolE)
	pool, _ := NewPoolSimulator(poolE)
	return pool
}

func TestPoolSimulatorPricable4626MultiAsset(t *testing.T) {
	weth := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	steth := "0xae7ab96520de3a18e5e111b5eaab095312d7fe84"
	wsteth := "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"
	amountIn := bignumber.NewBig("100000000000000000000")

	t.Run("WETH to stETH (pegged)", func(t *testing.T) {
		amountOut, err := getWethARMPool().CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: weth, Amount: amountIn},
			TokenOut:      steth,
		})
		assert.NoError(t, err)
		assert.Equal(t, bignumber.NewBig("100000000000000000000"), amountOut.TokenAmountOut.Amount)
	})

	t.Run("stETH to WETH (pegged)", func(t *testing.T) {
		amountOut, err := getWethARMPool().CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: steth, Amount: amountIn},
			TokenOut:      weth,
		})
		assert.NoError(t, err)
		assert.Equal(t, bignumber.NewBig("99970000000000000000"), amountOut.TokenAmountOut.Amount)
	})

	t.Run("WETH to wstETH (adapter)", func(t *testing.T) {
		amountOut, err := getWethARMPool().CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: weth, Amount: amountIn},
			TokenOut:      wsteth,
		})
		assert.NoError(t, err)
		assert.Equal(t, bignumber.NewBig("80643465443240529600"), amountOut.TokenAmountOut.Amount)
	})

	t.Run("wstETH to WETH (adapter)", func(t *testing.T) {
		amountOut, err := getWethARMPool().CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: wsteth, Amount: amountIn},
			TokenOut:      weth,
		})
		assert.NoError(t, err)
		assert.Equal(t, bignumber.NewBig("123965406807030271181"), amountOut.TokenAmountOut.Amount)
	})

	t.Run("stETH to wstETH rejected (base-to-base)", func(t *testing.T) {
		_, err := getWethARMPool().CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: steth, Amount: amountIn},
			TokenOut:      wsteth,
		})
		assert.ErrorIs(t, err, ErrUnsupportedSwap)
	})
}

func TestPoolSimulatorErrInsufficientLiquidity(t *testing.T) {
	p := getPool()
	// https://etherscan.io/tx/0x332289850d386bef8bc8a90fb6ec31519b6a64a0756e442f2546dc51db87fb32
	// r1 0x0000000000000000000000000000000000c092cc726b59717c60bc6e06d26000
	reserveOut := new(big.Int).Set(p.Info.Reserves[0])
	reserveOut.Sub(reserveOut, bignumber.NewBig("8824843694584167917191")).Add(reserveOut, bignumber.NewBig("8816768469433561587106"))

	amountIn := reserveOut.Mul(reserveOut, bignumber.NewBig("1000000000000000000000000000000000000")).Div(reserveOut, bignumber.NewBig("999898426597041524878150000000000000"))
	_, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xae7ab96520de3a18e5e111b5eaab095312d7fe84",
				Amount: amountIn,
			},
			TokenOut: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		},
	)
	assert.Nil(t, err)
	_, err = p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xae7ab96520de3a18e5e111b5eaab095312d7fe84",
				Amount: amountIn.Add(amountIn, big.NewInt(2)),
			},
			TokenOut: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		},
	)
	assert.Error(t, err)
}
