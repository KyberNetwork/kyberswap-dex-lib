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

// getEthenaARMPool builds a Pricable4626 pool (USDe/sUSDe) from a live on-chain snapshot of
// 0xCEDa2d856238aA0D12f6329de20B9115f07C366d taken after the ARM's upgrade to the shared
// AbstractARM contract (buyPrice/sellPrice via baseAssetConfigs(), no more token0/token1/traderate0/1).
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
		"extra":"{\"la\":\"0x4c9edd5852cd905f086c759e8383e09bff1e68b3\",\"ps\":\"1000000000000000000000000000000000000\",\"swapType\":3,\"armType\":2,\"hasWithdrawalQueue\":false,\"v\":{\"ba\":\"0x9d39a5de30e57443bff2a8307a4256c8797a3497\",\"ta\":\"1642140898458895542142337558\",\"ts\":\"1326183113092221454904213555\",\"bp\":\"999600000000000000000000000000000000\",\"sp\":\"999990000000000000000000000000000000\"}}"
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
	assert.Equal(t, bignumber.NewBig("807602061613940163083"), amountOut.TokenAmountOut.Amount)
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
	assert.Equal(t, bignumber.NewBig("1237750674016737246720"), amountOut.TokenAmountOut.Amount)
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
