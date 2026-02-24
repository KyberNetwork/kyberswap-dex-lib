package v2

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolSimulator_InsolvencyReproduction(t *testing.T) {
	poolData := `{"address":"0x6fcfdf043faef634e0ae7dc7573cf308fdbb28a8","exchange":"uniswap-v4-euler-v2","type":"euler-swap-v2","timestamp":1768385188,"reserves":["10000036786200862068965518","11599957328007"],"tokens":[{"address":"0x66bcf6151d5558afb47c38b20663589843156078","symbol":"liUSD-4w","decimals":18,"swappable":true},{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"er0\":\"10000000000000000000000000\",\"er1\":\"11600000000000\",\"mr0\":\"0\",\"mr1\":\"0\",\"px\":\"1160000\",\"py\":\"1000000000000000000\",\"cx\":\"1000000000000000000\",\"cy\":\"1000000000000000000\",\"f0\":\"0\",\"f1\":\"0\",\"sh\":\"0x0000000000000000000000000000000000000000\",\"p\":1,\"sv\":[{\"c\":\"43175303515055042970\",\"d\":\"0\",\"mD\":\"1999956824696484944957030\",\"tB\":\"0\",\"eAA\":\"36786201780892283272\",\"bC\":\"1800000000000000000000000\",\"dP\":\"1\",\"vP\":[\"1\",\"999740000000\"],\"vVP\":[\"1\",\"999740000000\"],\"ltv\":[0,10000],\"vLtv\":[0,10000]},{\"c\":\"57328007\",\"d\":\"42915300\",\"mD\":\"9999899756692\",\"tB\":\"42915300\",\"eAA\":\"0\",\"bC\":\"10000000000000\",\"dP\":\"999740000000\",\"vP\":[\"1\",\"999740000000\"],\"vVP\":[\"1\",\"999740000000\"],\"ltv\":[10000,0],\"vLtv\":[10000,0],\"iCE\":true}],\"bv\":[null,{\"c\":\"57328007\",\"d\":\"42915300\",\"mD\":\"9999899756692\",\"tB\":\"42915300\",\"eAA\":\"0\",\"bC\":\"10000000000000\",\"dP\":\"999740000000\",\"vP\":[\"1\",\"999740000000\"],\"vVP\":[\"1\",\"999740000000\"],\"ltv\":[10000,0],\"vLtv\":[10000,0],\"iCE\":true},{\"c\":\"57328007\",\"d\":\"42915300\",\"mD\":\"9999899756692\",\"tB\":\"42915300\",\"eAA\":\"0\",\"bC\":\"10000000000000\",\"dP\":\"999740000000\",\"vP\":[\"1\",\"999740000000\"],\"vVP\":[\"1\",\"999740000000\"],\"ltv\":[10000,0],\"vLtv\":[10000,0],\"iCE\":true}],\"cV\":\"0xdc6d457b6cf5dfad338a7982608e3306fd9474c7\",\"c\":[\"36786201780892283272\",\"0\"]}","staticExtra":"{\"sv0\":\"0xb04ad3337dc567a68a6f4D571944229320Ad1740\",\"sv1\":\"0xDc6D457b6cf5dfaD338a7982608e3306FD9474c7\",\"bv1\":\"0xDc6D457b6cf5dfaD338a7982608e3306FD9474c7\",\"ea\":\"0x5304ebB378186b081B99dbb8B6D17d9005eA0448\",\"evc\":\"0x0C9a3dd6b8F28529d72d7f9cE918D493519EE383\"}","blockNumber":24232233}`

	var entityPool entity.Pool
	err := json.Unmarshal([]byte(poolData), &entityPool)
	require.NoError(t, err)

	simulator, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	// swap 1000000 USDC => liUSD-4w
	tokenIn := entityPool.Tokens[1].Address
	tokenOut := entityPool.Tokens[0].Address
	amountIn := big.NewInt(1000000)

	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		TokenOut: tokenOut,
	}

	result, err := simulator.CalcAmountOut(params)

	assert.Error(t, err, "Expected insolvency error")
	if err == nil {
		t.Errorf("Expected insolvency error, but got result: %v", result.TokenAmountOut.Amount.String())
	}
}
