package eethorweeth

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"
)

func TestPoolSimulator(t *testing.T) {
	entityStr := "{\"address\":\"0x9ffdf407cde9a93c47611799da23924af3ef764f\",\"exchange\":\"eeth-or-weeth\",\"type\":\"eeth-or-weeth\",\"timestamp\":1732816463,\"reserves\":[\"1000000000000000000000\",\"1000000000000000000000\",\"1000000000000000000000\",\"1000000000000000000000\"],\"tokens\":[{\"address\":\"0xae7ab96520de3a18e5e111b5eaab095312d7fe84\",\"symbol\":\"stETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0\",\"symbol\":\"wstETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x35fa164735182de50811e8e2e824cfb9b6118ac2\",\"symbol\":\"eETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee\",\"symbol\":\"weETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"StETH\\\":{\\\"TotalPooledEther\\\":9796738809418974583538078,\\\"TotalShares\\\":8258952045397760272638590},\\\"StETHTokenInfo\\\":{\\\"DiscountInBasisPoints\\\":0,\\\"TotalDepositedThisPeriod\\\":39990256514091518,\\\"TotalDeposited\\\":512171900270894130150671,\\\"TimeBoundCapClockStartTime\\\":1732799075,\\\"TimeBoundCapInEther\\\":6000,\\\"TotalCapInEther\\\":1000000},\\\"Vampire\\\":{\\\"QuoteStEthWithCurve\\\":true,\\\"TimeBoundCapRefreshInterval\\\":3600},\\\"EtherFiPool\\\":{\\\"TotalPooledEther\\\":2232186054140230276362460},\\\"EETH\\\":{\\\"TotalShares\\\":2117963364874273931196687},\\\"CurveStETHToETH\\\":{\\\"Reserves\\\":[\\\"25582722458228443901566\\\",\\\"29152736312348263774387\\\",\\\"0\\\"],\\\"Extra\\\":\\\"{\\\\\\\"InitialA\\\\\\\":20000,\\\\\\\"FutureA\\\\\\\":90000,\\\\\\\"InitialATime\\\\\\\":1731805535,\\\\\\\"FutureATime\\\\\\\":1732495784,\\\\\\\"SwapFee\\\\\\\":1000000,\\\\\\\"AdminFee\\\\\\\":5000000000}\\\",\\\"StaticExtra\\\":\\\"{\\\\\\\"APrecision\\\\\\\":\\\\\\\"100\\\\\\\",\\\\\\\"LpToken\\\\\\\":\\\\\\\"0x06325440D014e39736583c165C2963BA99fAf14E\\\\\\\",\\\\\\\"IsNativeCoin\\\\\\\":[true,false]}\\\"}}\"}"

	var entity entity.Pool
	err := json.Unmarshal([]byte(entityStr), &entity)
	assert.NoError(t, err)

	simulator, err := NewPoolSimulator(entity)
	assert.NoError(t, err)

	// Swap stETH -> eETH
	amount := big.NewInt(1000000000000000000)
	res, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  stETH,
			Amount: amount,
		},
		TokenOut: eETH,
	})

	assert.NoError(t, err)
	assert.Equal(t, "948595357455051091", res.TokenAmountOut.Amount.String())
}
