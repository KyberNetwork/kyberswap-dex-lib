package cpmm

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	t.Run("1. should return correct value", func(t *testing.T) {
		desc := "pool 2 tokens, lp not involved"
		t.Log(desc)

		// block 659738, linea

		entityPoolStr := `{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocorev2-cpmm","type":"velocorev2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":0}","staticExtra":"{\"poolTokenNumber\":3,\"weights\":[2,1,1]}"}`
		var entityPool entity.Pool
		err := json.Unmarshal([]byte(entityPoolStr), &entityPool)
		assert.Nil(t, err)

		var (
			tokenAmountIn = pool.TokenAmount{
				Token:  "0xa219439258ca9da29e9cc4ce5596924745e12b93",
				Amount: new(big.Int).Mul(bignumber.BONE, big.NewInt(23)),
			}
			tokenOut = "0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1"
		)

		simulator, err := NewPoolSimulator(entityPool)
		assert.Nil(t, err)

		result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return simulator.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})
		assert.Nil(t, err)

		assert.Equal(t, "1310912514297532736758", result.TokenAmountOut.Amount.String())

		// simulator: -1310912514297532736758
		// contract:  -1310912514297532736758
	})

	t.Run("2. should return error", func(t *testing.T) {
		desc := "pool 2 tokens, lp involved and known r"
		t.Log(desc)

		// block 659738, linea

		entityPoolStr := `{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocore-v2-cpmm","type":"velocore-v2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":0}","staticExtra":"{\"poolTokenNumber\":3,\"weights\":[2,1,1]}"}`
		var entityPool entity.Pool
		err := json.Unmarshal([]byte(entityPoolStr), &entityPool)
		assert.Nil(t, err)

		var (
			tokenAmountIn = pool.TokenAmount{
				Token:  "0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca",
				Amount: new(big.Int).Mul(big.NewInt(1e4), big.NewInt(4)),
			}
			tokenOut = "0xa219439258ca9da29e9cc4ce5596924745e12b93"
		)

		simulator, err := NewPoolSimulator(entityPool)
		assert.Nil(t, err)

		_, err = testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return simulator.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})
		assert.ErrorIs(t, err, ErrNonPositiveAmountOut)

		// contract:  0
	})

	t.Run("3. should return correct value", func(t *testing.T) {
		desc := "pool 2 tokens, lp involved and unknown r"
		t.Log(desc)

		// block 659738, linea

		entityPoolStr := `{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocore-v2-cpmm","type":"velocore-v2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":0}","staticExtra":"{\"poolTokenNumber\":3,\"weights\":[2,1,1]}"}`
		var entityPool entity.Pool
		err := json.Unmarshal([]byte(entityPoolStr), &entityPool)
		assert.Nil(t, err)

		var (
			tokenAmountIn = pool.TokenAmount{
				Token:  "0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1",
				Amount: new(big.Int).Mul(bignumber.BONE, big.NewInt(3)),
			}
			tokenOut = "0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca"
		)

		simulator, err := NewPoolSimulator(entityPool)
		assert.Nil(t, err)

		result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return simulator.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})
		assert.Nil(t, err)

		assert.Equal(t, "114875306101", result.TokenAmountOut.Amount.String())

		// simulator: -114875306101
		// contract:  -114875306101
	})

	t.Run("4. should not panic", func(t *testing.T) {
		entityPoolStr := `{"address":"0xc53a048e4211a81e68001c6fa56364019f973e0b","reserveUsd":0.009998963162763646,"amplifiedTvl":0.009998963162763646,"exchange":"velocore-v2-cpmm","type":"velocore-v2-cpmm","timestamp":1718001015,"reserves":["5192281750178086030566074723720301","9998963156478118","1704"],"tokens":[{"address":"0xc53a048e4211a81e68001c6fa56364019f973e0b","swappable":true},{"address":"0x2039bb4116b4efc145ec4f0e2ea75012d6c0f181","swappable":true},{"address":"0x5aea5775959fbc2557cc8789bc1bf90a239d9a91","swappable":true}],"extra":"{\"fee1e9\":0,\"feeMultiplier\":44391380623778508}","staticExtra":"{\"weights\":[2,1,1],\"poolTokenNumber\":3,\"nativeTokenIndex\":2,\"vault\":\"0xf5E67261CB357eDb6C7719fEFAFaaB280cB5E2A6\"}","blockNumber":36206567}`
		var entityPool entity.Pool
		err := json.Unmarshal([]byte(entityPoolStr), &entityPool)
		assert.Nil(t, err)

		var (
			tokenAmountIn = pool.TokenAmount{
				Token:  "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
				Amount: new(big.Int).Mul(bignumber.BONE, big.NewInt(3)),
			}
			tokenOut = "0x2039bb4116b4efc145ec4f0e2ea75012d6c0f181"
		)

		simulator, err := NewPoolSimulator(entityPool)
		assert.Nil(t, err)

		result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
			return simulator.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})
		assert.Nil(t, err)
		assert.Equal(t, "9998963156478113", result.TokenAmountOut.Amount.String())
	})
}

func TestVelocoreExecute(t *testing.T) {
	t.Run("1. should return correct value", func(t *testing.T) {
		desc := "pool 2 tokens, all token involved, lp token included, lp token known r"
		t.Log(desc)

		// block 659738, linea

		entityPoolStr := `{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocore-v2-cpmm","type":"velocore-v2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":0}","staticExtra":"{\"poolTokenNumber\":3,\"weights\":[2,1,1]}"}`
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
				new(big.Int).Mul(big.NewInt(1e4), big.NewInt(-6996)),
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

		// contract:  [-69960000,7,908433084452167]
	})

	t.Run("2. should return correct value", func(t *testing.T) {
		desc := "pool 2 tokens, all token involved, lp token included, lp token unknown r"
		t.Log(desc)

		// block 659738, linea

		entityPoolStr := `{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocore-v2-cpmm","type":"velocore-v2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":0}","staticExtra":"{\"poolTokenNumber\":3,\"weights\":[2,1,1]}"}`
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
				new(big.Int).Mul(big.NewInt(1e4), big.NewInt(-6996)),
			}
		)

		simulator, err := NewPoolSimulator(entityPool)
		assert.Nil(t, err)

		result, err := simulator.velocoreExecute(tokens, r)
		assert.Nil(t, err)

		assert.Equal(t, "7", result.R[0].String())
		assert.Equal(t, "0", result.R[1].String())

		// contract:  [7,0,-69960000]
	})
}

func TestVelocoreExecuteFallback(t *testing.T) {
	t.Run("1. should return correct value", func(t *testing.T) {
		desc := "pool 2 tokens, all token involved, lp token included, lp token known r"
		t.Log(desc)

		// block 659738, linea

		entityPoolStr := `{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocore-v2-cpmm","type":"velocore-v2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":0}","staticExtra":"{\"poolTokenNumber\":3,\"weights\":[2,1,1]}"}`
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
				new(big.Int).Mul(big.NewInt(1e4), big.NewInt(-6996)),
				unknownBI,
				unknownBI,
			}
		)

		simulator, err := NewPoolSimulator(entityPool)
		assert.Nil(t, err)

		result, err := simulator.velocoreExecuteFallback(tokens, r)
		assert.Nil(t, err)

		assert.Equal(t, "5", result.R[1].String())
		assert.Equal(t, "908433163091179", result.R[2].String())

		// contract:  [-69960000,5,908433163091179]
	})

	t.Run("2. should return correct value", func(t *testing.T) {
		desc := "pool 2 tokens, all token involved, lp token included, lp token unknown r"
		t.Log(desc)

		// block 659738, linea

		entityPoolStr := `{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","exchange":"velocore-v2-cpmm","type":"velocore-v2-cpmm","timestamp":1697617544,"reserves":["5192296858534827628430578339644164","7774767","1310912514297980345043"],"tokens":[{"address":"0x515ac85ef7d21b5033a0ad71b194d4c52661b8ca","swappable":true},{"address":"0xa219439258ca9da29e9cc4ce5596924745e12b93","swappable":true},{"address":"0xcc22f6aa610d1b2a0e89ef228079cb3e1831b1d1","swappable":true}],"extra":"{\"fee1e9\":10000000,\"feeMultiplier\":0}","staticExtra":"{\"poolTokenNumber\":3,\"weights\":[2,1,1]}"}`
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
				new(big.Int).Mul(bigint1e9, big.NewInt(-6996)),
			}
		)

		simulator, err := NewPoolSimulator(entityPool)
		assert.Nil(t, err)

		result, err := simulator.velocoreExecuteFallback(tokens, r)
		assert.Nil(t, err)

		assert.Equal(t, "538774", result.R[0].String())
		assert.Equal(t, "0", result.R[1].String())

		// contract:  [538774,0,-6996000000000]
	})
}
