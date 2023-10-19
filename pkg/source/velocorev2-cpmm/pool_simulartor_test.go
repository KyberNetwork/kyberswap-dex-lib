package velocorev2cpmm

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"
)

func TestPool2TokensCalcAmountOutLpNotInvolved(t *testing.T) {
	// block 659738, linea

	entityPoolStr := `{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocorev2-cpmm","type":"velocorev2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","weight":2,"swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","weight":1,"swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","weight":1,"swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":\"0\"}","staticExtra":"{\"poolTokenNumber\":3}"}`
	var entityPool entity.Pool
	err := json.Unmarshal([]byte(entityPoolStr), &entityPool)
	assert.Nil(t, err)

	var (
		tokenAmountIn = pool.TokenAmount{
			Token:  "0xa219439258ca9da29e9cc4ce5596924745e12b93",
			Amount: new(big.Int).Mul(bigint1e18, big.NewInt(23)),
		}
		tokenOut = "0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1"
	)

	simulator, err := NewPoolSimulator(entityPool)
	assert.Nil(t, err)

	result, err := simulator.CalcAmountOut(tokenAmountIn, tokenOut)
	assert.Nil(t, err)

	assert.Equal(t, "1310912514297532736759", result.TokenAmountOut.Amount.String())

	// resultBytes, _ := json.Marshal(result)
	// t.Error(string(resultBytes))

	// simulator: -1310912514297532736759
	// contract: -1310912514297532736758
}

func TestPool2TokensCalcAmountOutLpInvolvedKnown(t *testing.T) {
	// block 659738, linea

	entityPoolStr := `{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocorev2-cpmm","type":"velocorev2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","weight":2,"swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","weight":1,"swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","weight":1,"swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":\"0\"}","staticExtra":"{\"poolTokenNumber\":3}"}`
	var entityPool entity.Pool
	err := json.Unmarshal([]byte(entityPoolStr), &entityPool)
	assert.Nil(t, err)

	var (
		tokenAmountIn = pool.TokenAmount{
			Token:  "0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca",
			Amount: new(big.Int).Mul(bigint1e4, big.NewInt(4)),
		}
		tokenOut = "0xa219439258ca9da29e9cc4ce5596924745e12b93"
	)

	simulator, err := NewPoolSimulator(entityPool)
	assert.Nil(t, err)

	result, err := simulator.CalcAmountOut(tokenAmountIn, tokenOut)
	assert.Nil(t, err)

	assert.Equal(t, "-2", result.TokenAmountOut.Amount.String())

	// resultBytes, _ := json.Marshal(result)
	// t.Error(string(resultBytes))

	// simulator: 2
	// contract: 1
}

func TestPool2TokensCalcAmountOutLpInvolvedUnknown(t *testing.T) {
	// block 659738, linea

	entityPoolStr := `{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocorev2-cpmm","type":"velocorev2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","weight":2,"swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","weight":1,"swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","weight":1,"swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":\"0\"}","staticExtra":"{\"poolTokenNumber\":3}"}`
	var entityPool entity.Pool
	err := json.Unmarshal([]byte(entityPoolStr), &entityPool)
	assert.Nil(t, err)

	var (
		tokenAmountIn = pool.TokenAmount{
			Token:  "0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1",
			Amount: new(big.Int).Mul(bigint1e18, big.NewInt(3)),
		}
		tokenOut = "0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca"
	)

	simulator, err := NewPoolSimulator(entityPool)
	assert.Nil(t, err)

	result, err := simulator.CalcAmountOut(tokenAmountIn, tokenOut)
	assert.Nil(t, err)

	resultBytes, _ := json.Marshal(result)
	t.Error(string(resultBytes))

	// simulator: 114874976405

	// contract: 114875306101
}

// TODO: test for pool > 2 tokens
