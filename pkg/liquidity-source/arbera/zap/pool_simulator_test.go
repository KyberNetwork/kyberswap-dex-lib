package arberazap

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	arberaden "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/arbera/den"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	genericsimplerate "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/generic-simple-rate"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0xface73a169e2ca2934036c8af9f464b5de9ef0ca","exchange":"erc4626","type":"erc4626","timestamp":1760325161,"reserves":["0","826550308605061016110006"],"tokens":[{"address":"0xface73a169e2ca2934036c8af9f464b5de9ef0ca","symbol":"stLBGT","decimals":18,"swappable":true},{"address":"0xbaadcc2962417c01af99fb2b7c75706b9bd6babe","symbol":"LBGT","decimals":18,"swappable":true}],"extra":"{\"g\":{\"d\":70481,\"r\":49289},\"sT\":3,\"dR\":[\"348724\",\"348724589834\",\"348724589834892976\",\"348724589834892976145851\",\"348724589834892976145851533150\"],\"rR\":[\"2867592\",\"2867592447304\",\"2867592447304790449\",\"2867592447304790449681589\",\"2867592447304790449681589753105\"]}","blockNumber":11708261}`), &entityPool)
	erc4626Sim = lo.Must(erc4626.NewPoolSimulator(entityPool))

	_ = json.Unmarshal([]byte(`{"address":"0x883899d0111d69f85fdfd19e4b89e613f231b781","exchange":"arbera-den","type":"arbera-den","timestamp":1760324380,"reserves":["100000000000000000000000000","100000000000000000000000000"],"tokens":[{"address":"0x883899d0111d69f85fdfd19e4b89e613f231b781","symbol":"brLBGT","decimals":18,"swappable":true},{"address":"0xface73a169e2ca2934036c8af9f464b5de9ef0ca","symbol":"stLBGT","decimals":18,"swappable":true}],"extra":"{\"assets\":[{\"token\":\"0xface73a169e2ca2934036c8af9f464b5de9ef0ca\",\"weighting\":\"1000000000000000000\",\"basePriceUSDX96\":\"0\",\"c1\":\"0x0000000000000000000000000000000000000000\",\"q1\":\"79228162514264337593543950336000000000000000000\"}],\"assetSupplies\":[\"12600849266904540184172\"],\"supply\":\"12295798672905393125427\",\"fee\":{\"bond\":\"69\",\"debond\":\"69\",\"burn\":\"2000\",\"buy\":\"69\",\"sell\":\"69\"}}"}
	`), &entityPool)
	den1Sim = lo.Must(arberaden.NewPoolSimulator(entityPool))

	_         = json.Unmarshal([]byte(`{"address":"0xdc06ec361cf28a610b2f0fc3d25854cf68141610","exchange":"arbera-den-amm","type":"uniswap-v2","timestamp":1760324733,"reserves":["94183703149452617690341","2772328081415220135969"],"tokens":[{"address":"0x0c1f965eb5221b8daca960dac1ccfda5a97b7dd7","symbol":"brarBERO","decimals":18,"swappable":true},{"address":"0x883899d0111d69f85fdfd19e4b89e613f231b781","symbol":"brLBGT","decimals":18,"swappable":true}],"extra":"{\"fee\":3,\"feePrecision\":1000}","blockNumber":11708261}`), &entityPool)
	denAmmSim = lo.Must(uniswapv2.NewPoolSimulator(entityPool))

	_       = json.Unmarshal([]byte(`{"address":"0x0c1f965eb5221b8daca960dac1ccfda5a97b7dd7","exchange":"arbera-den","type":"arbera-den","timestamp":1760324380,"reserves":["100000000000000000000000000","100000000000000000000000000"],"tokens":[{"address":"0x0c1f965eb5221b8daca960dac1ccfda5a97b7dd7","symbol":"brarBERO","decimals":18,"swappable":true},{"address":"0xfa7767bbb3d832217abaa86e5f2654429b3bf29f","symbol":"arBERO","decimals":18,"swappable":true}],"extra":"{\"assets\":[{\"token\":\"0xfa7767bbb3d832217abaa86e5f2654429b3bf29f\",\"weighting\":\"1000000000000000000\",\"basePriceUSDX96\":\"0\",\"c1\":\"0x0000000000000000000000000000000000000000\",\"q1\":\"79228162514264337593543950336000000000000000000\"}],\"assetSupplies\":[\"124403165937860479291712\"],\"supply\":\"121922873438761813187052\",\"fee\":{\"bond\":\"69\",\"debond\":\"69\",\"burn\":\"2000\",\"buy\":\"69\",\"sell\":\"69\"}}"}`), &entityPool)
	den2Sim = lo.Must(arberaden.NewPoolSimulator(entityPool))

	_        = json.Unmarshal([]byte(`{"address":"0x3fd02eaddb07080b8e2640afb6d52f10d6396926","exchange":"arbera-stake","type":"generic-simple-rate","timestamp":1760324379,"reserves":["100000000000000000000000000","100000000000000000000000000"],"tokens":[{"address":"0x3fd02eaddb07080b8e2640afb6d52f10d6396926","symbol":"starBERO","decimals":18,"swappable":true},{"address":"0xfa7767bbb3d832217abaa86e5f2654429b3bf29f","symbol":"arBERO","decimals":18,"swappable":true}],"extra":"{\"paused\":false,\"rate\":\"1\",\"rateUnit\":\"1\",\"isRateInversed\":false,\"isBidirectional\":true,\"defaultGas\":60000}"}`), &entityPool)
	stakeSim = lo.Must(genericsimplerate.NewPoolSimulator(entityPool))

	_      = json.Unmarshal([]byte(`{"address":"0xcec42c8ddcc73065090d36db1e17188d0767fcc6","exchange":"arbera-zap","type":"arbera-zap","reserves":["100000000000000000000000000","100000000000000000000000000","100000000000000000000000000","100000000000000000000000000","100000000000000000000000000","100000000000000000000000000"],"tokens":[{"address":"0xbaadcc2962417c01af99fb2b7c75706b9bd6babe","symbol":"LBGT","decimals":18,"swappable":true},{"address":"0xface73a169e2ca2934036c8af9f464b5de9ef0ca","symbol":"stLBGT","decimals":18,"swappable":true},{"address":"0x883899d0111d69f85fdfd19e4b89e613f231b781","symbol":"brLBGT","decimals":18,"swappable":true},{"address":"0x0c1f965eb5221b8daca960dac1ccfda5a97b7dd7","symbol":"brarBERO","decimals":18,"swappable":true},{"address":"0xfa7767bbb3d832217abaa86e5f2654429b3bf29f","symbol":"arBERO","decimals":18,"swappable":true},{"address":"0x3fd02eaddb07080b8e2640afb6d52f10d6396926","symbol":"starBERO","decimals":18,"swappable":true}],"extra":"{}","staticExtra":"{\"basePools\":[\"0xface73a169e2ca2934036c8af9f464b5de9ef0ca\",\"0x883899d0111d69f85fdfd19e4b89e613f231b781\",\"0xdc06ec361cf28a610b2f0fc3d25854cf68141610\",\"0x0c1f965eb5221b8daca960dac1ccfda5a97b7dd7\",\"0x3fd02eaddb07080b8e2640afb6d52f10d6396926\"]}"}`), &entityPool)
	zapSim = lo.Must(NewPoolSimulator(entityPool, map[string]pool.IPoolSimulator{
		"0xface73a169e2ca2934036c8af9f464b5de9ef0ca": erc4626Sim,
		"0x883899d0111d69f85fdfd19e4b89e613f231b781": den1Sim,
		"0xdc06ec361cf28a610b2f0fc3d25854cf68141610": denAmmSim,
		"0x0c1f965eb5221b8daca960dac1ccfda5a97b7dd7": den2Sim,
		"0x3fd02eaddb07080b8e2640afb6d52f10d6396926": stakeSim,
	}))
)

func adjustFee(amountStr string, feeStr string) string {
	amount, _ := new(big.Int).SetString(amountStr, 10)
	fee, _ := new(big.Int).SetString(feeStr, 10)
	burnedAmount := new(big.Int)
	burnedAmount.Mul(amount, fee).Div(burnedAmount, arberaden.DEN.ToBig())
	amount.Sub(amount, burnedAmount)
	return amount.String()
}

func TestForwardCalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, zapSim, map[int]map[int]map[string]string{
		0: {
			5: {
				"100000000000000000": "1151822894979744447",
			},
			4: {
				"100000000000000000": "1151822894979744447",
			},
			3: {
				"100000000000000000": adjustFee("1144599362298207513", "69"),
			},
			2: {
				"100000000000000000": "",
			},
			1: {
				"100000000000000000": "",
			},
			0: {
				"100000000000000000": "",
			},
		},
		1: {
			5: {
				"34872458983489297": "1151822894979744447",
			},
			4: {
				"34872458983489297": "1151822894979744447",
			},
			3: {
				"34872458983489297": adjustFee("1144599362298207513", "69"),
			},
			2: {
				"34872458983489297": "",
			},
			1: {
				"34872458983489297": "",
			},
			0: {
				"34872458983489297": "",
			},
		},
		2: {
			5: {
				"33793446076511935": "1151822894979744447",
			},
			4: {
				"33793446076511935": "1151822894979744447",
			},
			3: {
				"33793446076511935": adjustFee("1144599362298207513", "69"),
			},
			2: {
				"33793446076511935": "",
			},
			1: {
				"33793446076511935": "",
			},
			0: {
				"33793446076511935": "",
			},
		},
	})
	testutil.TestCalcAmountOut(t, erc4626Sim, map[int]map[int]map[string]string{
		1: {
			0: {
				"100000000000000000": "34872458983489297",
			},
		},
	})
	testutil.TestCalcAmountOut(t, den1Sim, map[int]map[int]map[string]string{
		1: {
			0: {
				"34872458983489297": "33793446076511935",
			},
		},
	})
	testutil.TestCalcAmountOut(t, denAmmSim, map[int]map[int]map[string]string{
		1: {
			0: {
				"33793446076511935": "1144599362298207513",
			},
		},
	})
	testutil.TestCalcAmountOut(t, den2Sim, map[int]map[int]map[string]string{
		0: {
			1: {
				adjustFee("1144599362298207513", "69"): "1151822894979744447",
			},
		},
	})
	testutil.TestCalcAmountOut(t, stakeSim, map[int]map[int]map[string]string{
		1: {
			0: {
				"1151822894979744447": "1151822894979744447",
			},
		},
	})
}

func TestBackwardCalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, zapSim, map[int]map[int]map[string]string{
		5: {
			0: {
				"100000000000000000": "8278597706612320",
			},
			1: {
				"100000000000000000": "2886950589646467",
			},
			2: {
				"100000000000000000": "2836633948218530",
			},
			3: {
				"100000000000000000": "",
			},
			4: {
				"100000000000000000": "",
			},
			5: {
				"100000000000000000": "",
			},
		},
		4: {
			0: {
				"100000000000000000": "8278597706612320",
			},
			1: {
				"100000000000000000": "2886950589646467",
			},
			2: {
				"100000000000000000": "2836633948218530",
			},
			3: {
				"100000000000000000": "",
			},
			4: {
				"100000000000000000": "",
			},
			5: {
				"100000000000000000": "",
			},
		},
		3: {
			0: {
				"97330003379909764": "8278597706612320",
			},
			1: {
				"97330003379909764": "2886950589646467",
			},
			2: {
				"97330003379909764": "2836633948218530",
			},
			3: {
				"100000000000000000": "",
			},
			4: {
				"100000000000000000": "",
			},
			5: {
				"100000000000000000": "",
			},
		},
	})

	testutil.TestCalcAmountOut(t, stakeSim, map[int]map[int]map[string]string{
		0: {
			1: {
				"100000000000000000": "100000000000000000",
			},
		},
	})

	testutil.TestCalcAmountOut(t, den2Sim, map[int]map[int]map[string]string{
		1: {
			0: {
				"100000000000000000": "97330003379909764",
			},
		},
	})

	testutil.TestCalcAmountOut(t, denAmmSim, map[int]map[int]map[string]string{
		0: {
			1: {
				adjustFee("97330003379909764", "69"): "2836633948218530",
			},
		},
	})

	testutil.TestCalcAmountOut(t, den1Sim, map[int]map[int]map[string]string{
		0: {
			1: {
				"2836633948218530": "2886950589646467",
			},
		},
	})

	testutil.TestCalcAmountOut(t, erc4626Sim, map[int]map[int]map[string]string{
		0: {
			1: {
				"2886950589646467": "8278597706612320",
			},
		},
	})
}
