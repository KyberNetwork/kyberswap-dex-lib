package velocorev2cpmm

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"
)

func TestCalcAmountOut1(t *testing.T) {
	desc := "pool 2 tokens, lp not involved"
	t.Log(desc)

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

	// simulator: -1310912514297532736759
	// contract:  -1310912514297532736758
}

func TestCalcAmountOut2(t *testing.T) {
	desc := "pool 2 tokens, lp involved and known r"
	t.Log(desc)

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

	// simulator: 2
	// contract:  1
}

func TestCalcAmountOut3(t *testing.T) {
	desc := "pool 2 tokens, lp involved and unknown r"
	t.Log(desc)

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

	assert.Equal(t, "114875306101", result.TokenAmountOut.Amount.String())

	// simulator: -114875306101
	// contract:  -114875306101
}

func TestVelocoreExecute1(t *testing.T) {
	desc := "pool 2 tokens, all token involved, lp token included, lp token known r"
	t.Log(desc)

	// block 659738, linea

	entityPoolStr := `{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocorev2-cpmm","type":"velocorev2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","weight":2,"swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","weight":1,"swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","weight":1,"swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":\"0\"}","staticExtra":"{\"poolTokenNumber\":3}"}`
	var entityPool entity.Pool
	err := json.Unmarshal([]byte(entityPoolStr), &entityPool)
	assert.Nil(t, err)

	var (
		tokens = []string{
			"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca",
			"0xa219439258ca9da29e9cc4ce5596924745e12b93",
			"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1",
		}

		r = []*big.Int{
			new(big.Int).Mul(bigint1e4, big.NewInt(-6996)),
			unknownBI,
			unknownBI,
		}
	)

	simulator, err := NewPoolSimulator(entityPool)
	assert.Nil(t, err)

	result, err := simulator.velocoreExecute(tokens, r)
	assert.Nil(t, err)

	assert.Equal(t, "7", result.R[1].String())
	assert.Equal(t, "908433084452167", result.R[2].String())

	// simulator: [-69960000,7,908433084452167]
	// contract:  [-69960000,7,908433084452167]
}

func TestVelocoreExecute2(t *testing.T) {
	desc := "pool 2 tokens, all token involved, lp token included, lp token unknown r"
	t.Log(desc)

	// block 659738, linea

	entityPoolStr := `{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocorev2-cpmm","type":"velocorev2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","weight":2,"swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","weight":1,"swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","weight":1,"swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":\"0\"}","staticExtra":"{\"poolTokenNumber\":3}"}`
	var entityPool entity.Pool
	err := json.Unmarshal([]byte(entityPoolStr), &entityPool)
	assert.Nil(t, err)

	var (
		tokens = []string{
			"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca",
			"0xa219439258ca9da29e9cc4ce5596924745e12b93",
			"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1",
		}

		r = []*big.Int{
			unknownBI,
			unknownBI,
			new(big.Int).Mul(bigint1e4, big.NewInt(-6996)),
		}
	)

	simulator, err := NewPoolSimulator(entityPool)
	assert.Nil(t, err)

	result, err := simulator.velocoreExecute(tokens, r)
	assert.Nil(t, err)

	assert.Equal(t, "7", result.R[0].String())
	assert.Equal(t, "0", result.R[1].String())

	// simulator: [7,0,-69960000]
	// contract:  [7,0,-69960000]
}

func TestVelocoreExecuteFallback1(t *testing.T) {
	desc := "pool 2 tokens, all token involved, lp token included, lp token known r"
	t.Log(desc)

	
}

func TestVelocoreExecuteFallback2(t *testing.T) {
	desc := "pool 2 tokens, all token involved, lp token included, lp token known r"
	t.Log(desc)

}
