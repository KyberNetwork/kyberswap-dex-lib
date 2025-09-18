package lglclob

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	susdcPool      = `{"address":"0xc7723fe3df538f76a063eb5e62867960d236accf","swapFee":0.0003,"exchange":"xpress","type":"xpress","reserves":["9543508400000000000000000","75133739247"],"tokens":[{"address":"0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38","symbol":"S","decimals":18,"swappable":true},{"address":"0x29219dd400f2bf60e5a23d13be72b486d4038894","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"b\":{\"p\":[\"3117\",\"3116\",\"3113\",\"3000\",\"2925\",\"2120\",\"1950\",\"1822\",\"1745\",\"890\"],\"s\":[\"933470\",\"2800410\",\"14935837\",\"121630\",\"81729\",\"4000720\",\"2468061\",\"269134\",\"1037263\",\"1048295\"]},\"a\":{\"p\":[\"3119\",\"3120\",\"3123\",\"3200\",\"3375\",\"3450\",\"4000\",\"4400\",\"4464\",\"4895\",\"5689\",\"6818\",\"7000\",\"13300\",\"19900\",\"30000\",\"100000000\"],\"s\":[\"933470\",\"8401546\",\"9334701\",\"45000\",\"10000\",\"15000\",\"172000\",\"2500\",\"212\",\"1202718\",\"1845777\",\"94259\",\"100\",\"11779\",\"200\",\"100\",\"300\"]}}","staticExtra":"{\"sX\":\"10000000000000000\",\"sY\":\"1\",\"eth\":true}","blockNumber":46254472}`
	susdcPoolEmpty = `{"address":"0xc7723fe3df538f76a063eb5e62867960d236accf","swapFee":0.0003,"exchange":"xpress","type":"xpress","reserves":["9543508400000000000000000","75133739247"],"tokens":[{"address":"0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38","symbol":"S","decimals":18,"swappable":true},{"address":"0x29219dd400f2bf60e5a23d13be72b486d4038894","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{}","staticExtra":"{\"sX\":\"10000000000000000\",\"sY\":\"1\",\"eth\":true}","blockNumber":46254472}`
)

func TestPoolSimulator_CalcAmountOut_X_To_Y(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(susdcPool), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
		Amount: bignumber.NewBig10("1000000000000000000"), // 1 S
	}
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x29219dd400f2bf60e5a23d13be72b486d4038894",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "311606", result.TokenAmountOut.Amount.String())
	assert.Equal(t, "0x29219dd400f2bf60e5a23d13be72b486d4038894", result.TokenAmountOut.Token)
	assert.Equal(t, "94", result.Fee.Amount.String()) // 0.0003%
	assert.Equal(t, "0x29219dd400f2bf60e5a23d13be72b486d4038894", result.Fee.Token)
	assert.Equal(t, "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38", result.RemainingTokenAmountIn.Token)
	assert.Equal(t, "0", result.RemainingTokenAmountIn.Amount.String())

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, "3117", poolSim.Bids.ArrayPrices[0].String())
	assert.Equal(t, "933370", poolSim.Bids.ArrayShares[0].String()) // 933470 - 100 = 933370
	assert.Equal(t, "3119", poolSim.Asks.ArrayPrices[0].String())
	assert.Equal(t, "933470", poolSim.Asks.ArrayShares[0].String())
}

func TestPoolSimulator_CalcAmountOut_X_To_Y_FillBid(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(susdcPool), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
		Amount: bignumber.NewBig10("9334700000000000000000"), // 9334.70 S
	}
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x29219dd400f2bf60e5a23d13be72b486d4038894",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "0", result.RemainingTokenAmountIn.Amount.String())

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, "2800410", poolSim.Bids.ArrayShares[0].String())
	assert.Equal(t, "3116", poolSim.Bids.ArrayPrices[0].String())
}

func TestPoolSimulator_CalcAmountOut_X_To_Y_FillAll(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(susdcPool), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
		Amount: bignumber.NewBig10("276965490000000000000000"), // 27696.549 S
	}
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x29219dd400f2bf60e5a23d13be72b486d4038894",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "0", result.RemainingTokenAmountIn.Amount.String())

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, 0, len(poolSim.Bids.ArrayShares))
	assert.Equal(t, 0, len(poolSim.Bids.ArrayPrices))
}

func TestPoolSimulator_CalcAmountOut_X_To_Y_FillAllWithRemainder(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(susdcPool), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
		Amount: bignumber.NewBig10("276965500000000000000000"), // 27696.550 S
	}
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x29219dd400f2bf60e5a23d13be72b486d4038894",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "10000000000000000", result.RemainingTokenAmountIn.Amount.String())

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, 0, len(poolSim.Bids.ArrayShares))
	assert.Equal(t, 0, len(poolSim.Bids.ArrayPrices))
}

func TestPoolSimulator_CalcAmountOut_X_To_Y_WithDust(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(susdcPool), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
		Amount: bignumber.NewBig10("1000000000000000123"), // 1 S + dust
	}
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x29219dd400f2bf60e5a23d13be72b486d4038894",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "311606", result.TokenAmountOut.Amount.String())
	assert.Equal(t, "94", result.Fee.Amount.String()) // 0.0003%
	assert.Equal(t, "123", result.RemainingTokenAmountIn.Amount.String())

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, "3117", poolSim.Bids.ArrayPrices[0].String())
	assert.Equal(t, "933370", poolSim.Bids.ArrayShares[0].String()) // 933470 - 100 = 933370
}

func TestPoolSimulator_CalcAmountOut_X_To_Y_EmptyPool(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(susdcPoolEmpty), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
		Amount: bignumber.NewBig10("1000000000000000000"), // 1 S
	}
	_, err = poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x29219dd400f2bf60e5a23d13be72b486d4038894",
	})
	require.Error(t, err)
}

func TestPoolSimulator_CalcAmountOut_Y_To_X(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(susdcPool), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x29219dd400f2bf60e5a23d13be72b486d4038894",
		Amount: bignumber.NewBig10("998380"), // 0.998080 USDC + 0.0003 USDC (fee)
	}
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "3200000000000000000", result.TokenAmountOut.Amount.String()) // 3.2 S
	assert.Equal(t, "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38", result.TokenAmountOut.Token)
	assert.Equal(t, "300", result.Fee.Amount.String()) // 0.0003%
	assert.Equal(t, "0x29219dd400f2bf60e5a23d13be72b486d4038894", result.Fee.Token)
	assert.Equal(t, "0x29219dd400f2bf60e5a23d13be72b486d4038894", result.RemainingTokenAmountIn.Token)
	assert.Equal(t, "0", result.RemainingTokenAmountIn.Amount.String())

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, "933150", poolSim.Asks.ArrayShares[0].String()) // 933470 - 320 = 933150
	assert.Equal(t, "3119", poolSim.Asks.ArrayPrices[0].String())
	assert.Equal(t, "3117", poolSim.Bids.ArrayPrices[0].String())
	assert.Equal(t, "933470", poolSim.Bids.ArrayShares[0].String())
}

func TestPoolSimulator_CalcAmountOut_Y_To_X_FillAsk(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(susdcPool), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x29219dd400f2bf60e5a23d13be72b486d4038894",
		Amount: bignumber.NewBig10("2912366378"), // 2911.492930 USDC + 0.873448 USDC (fee)
	}
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "9334700000000000000000", result.TokenAmountOut.Amount.String()) // 9334.70 S
	assert.Equal(t, "873448", result.Fee.Amount.String())                            // 0.0003%
	assert.Equal(t, "0", result.RemainingTokenAmountIn.Amount.String())

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, "8401546", poolSim.Asks.ArrayShares[0].String())
	assert.Equal(t, "3120", poolSim.Asks.ArrayPrices[0].String())
}

func TestPoolSimulator_CalcAmountOut_Y_To_X_FillAll(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(susdcPool), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x29219dd400f2bf60e5a23d13be72b486d4038894",
		Amount: bignumber.NewBig10("106432882855"), // 106400.962566 USDC + 31.920289 USDC (fee)
	}
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "220696620000000000000000", result.TokenAmountOut.Amount.String()) // 220696.62 S
	assert.Equal(t, "31920289", result.Fee.Amount.String())                            // 0.0003%
	assert.Equal(t, "0", result.RemainingTokenAmountIn.Amount.String())

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, 0, len(poolSim.Asks.ArrayShares))
	assert.Equal(t, 0, len(poolSim.Asks.ArrayPrices))
}

func TestPoolSimulator_CalcAmountOut_Y_To_X_FillAllWithRemainder(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(susdcPool), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x29219dd400f2bf60e5a23d13be72b486d4038894",
		Amount: bignumber.NewBig10("106432882856"), // 106400.962566 USDC + 31.920289 USDC (fee) + 0.000001 USDC
	}
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "220696620000000000000000", result.TokenAmountOut.Amount.String()) // 220696.62 S
	assert.Equal(t, "31920289", result.Fee.Amount.String())                            // 0.0003%
	assert.Equal(t, "1", result.RemainingTokenAmountIn.Amount.String())

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, 0, len(poolSim.Asks.ArrayShares))
	assert.Equal(t, 0, len(poolSim.Asks.ArrayPrices))
}

func TestPoolSimulator_CalcAmountOut_Y_To_X_WithDust(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(susdcPool), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x29219dd400f2bf60e5a23d13be72b486d4038894",
		Amount: bignumber.NewBig10("998381"), // 0.998080 USDC + 0.0003 USDC (fee) + 0.000001 USDC
	}
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "3200000000000000000", result.TokenAmountOut.Amount.String()) // 3.2 S
	assert.Equal(t, "300", result.Fee.Amount.String())                            // 0.0003%
	assert.Equal(t, "1", result.RemainingTokenAmountIn.Amount.String())

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, "933150", poolSim.Asks.ArrayShares[0].String()) // 933470 - 320 = 933150
	assert.Equal(t, "3119", poolSim.Asks.ArrayPrices[0].String())
}

func TestPoolSimulator_CalcAmountOut_Y_To_X_EmptyPool(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(susdcPoolEmpty), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(*poolEntity)
	require.NoError(t, err)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x29219dd400f2bf60e5a23d13be72b486d4038894",
		Amount: bignumber.NewBig10("998380"),
	}
	_, err = poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
	})
	require.Error(t, err)
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()

	poolEntity := new(entity.Pool)
	_ = json.Unmarshal([]byte(susdcPool), poolEntity)
	poolSim, _ := NewPoolSimulator(*poolEntity)

	tokenAmtIn := pool.TokenAmount{
		Token:  "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
		Amount: bignumber.NewBig10("1000000000000000000"), // 1 S
	}
	result, _ := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x29219dd400f2bf60e5a23d13be72b486d4038894",
	})

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})

	tokenAmtIn = pool.TokenAmount{
		Token:  "0x039e2fb66102314ce7b64ce5ce3e5183bc94ad38",
		Amount: bignumber.NewBig10("9333700000000000000000"), // 9333.70 S
	}
	result, _ = poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      "0x29219dd400f2bf60e5a23d13be72b486d4038894",
	})

	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, "2800410", poolSim.Bids.ArrayShares[0].String())
	assert.Equal(t, "3116", poolSim.Bids.ArrayPrices[0].String())
}
