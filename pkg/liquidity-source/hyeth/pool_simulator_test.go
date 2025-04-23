package hyeth

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func getPool() *PoolSimulator {
	var poolE entity.Pool
	_ = json.Unmarshal([]byte("{\"address\":\"0xcb1eea349f25288627f008c5e2a69b684bdddf49\",\"exchange\":\"hyeth\",\"type\":\"hyeth\",\"timestamp\":1745235076,\"reserves\":[\"4946361947932843870115\",\"5005345678839792956730\"],\"tokens\":[{\"address\":\"0xc4506022fb8090774e8a628d5084eed61d9b99ee\",\"name\":\"hyeth\",\"symbol\":\"hyETH\",\"decimals\":18,\"weight\":1,\"swappable\":true},{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"name\":\"WETH\",\"symbol\":\"WETH\",\"decimals\":18,\"weight\":1,\"swappable\":true}],\"extra\":\"{\\\"feeI\\\":\\\"0\\\",\\\"feeR\\\":\\\"0\\\",\\\"comp\\\":\\\"0x701907283a57ff77e255c3f1aad790466b8ce4ef\\\",\\\"compSup\\\":\\\"4946361947932843870115\\\",\\\"compAss\\\":\\\"5005345678839792956730\\\",\\\"compHyb\\\":\\\"1015907674038080762600\\\",\\\"hySup\\\":\\\"809233550815085194542\\\",\\\"dpru\\\":\\\"1255394901774434537\\\",\\\"epru\\\":[],\\\"isDisabled\\\":false,\\\"maxDeposit\\\":\\\"1000000024671486719480691603261\\\",\\\"maxRedeem\\\":\\\"115792089237316195423570985008687907853269984665640564039457584007913129639935\\\"}\"}"), &poolE)
	pool, _ := NewPoolSimulator(poolE)
	return pool
}
func TestPoolSimulator_issue(t *testing.T) {
	// https://etherscan.io/address/0x04b59F9F09750C044D7CfbC177561E409085f0f3#readContract
	// revert of getRequiredComponentIssuanceUnits
	p := getPool()
	assert.Equal(t, uint256.NewInt(1e18), p.getRequiredAmountSetToken(uint256.NewInt(1255394901774434543)))
	amountOut, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: big.NewInt(1e18),
			},
			TokenOut: "0xc4506022fb8090774e8a628d5084eed61d9b99ee",
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(787175295419172397), amountOut.TokenAmountOut.Amount)
}

func TestPoolSimulator_redeem(t *testing.T) {
	// https://etherscan.io/address/0x04b59F9F09750C044D7CfbC177561E409085f0f3#readContract
	// getRequiredComponentRedemptionUnits
	p := getPool()
	assert.Equal(t, uint256.NewInt(1255394901774434537), p.getRequiredComponentRedemptionUnits(uint256.NewInt(1e18)))
	amountOut, err := p.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xc4506022fb8090774e8a628d5084eed61d9b99ee",
				Amount: big.NewInt(1e18),
			},
			TokenOut: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(1270365070930608945), amountOut.TokenAmountOut.Amount)
}
