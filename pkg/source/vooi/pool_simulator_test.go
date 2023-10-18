package vooi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestNewPoolSimulator(t *testing.T) {
	t.Run("it should construct pool simulator successfully", func(t *testing.T) {
		entityPool := entity.Pool{
			Address:  "0xbc7f67fa9c72f9fccf917cbcee2a50deb031462a",
			Exchange: "vooi",
			Type:     "vooi",
			Tokens: entity.PoolTokens{
				{
					Address: "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
				},
				{
					Address: "0xa219439258ca9da29e9cc4ce5596924745e12b93",
				},
				{
					Address: "0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5",
				},
			},
			Reserves: []string{
				"109755508503386757517651",
				"108295197536453495651912",
				"31982154523056912809541",
			},
			Extra: "{\"assetByToken\":{\"0x176211869ca2b568f2a7d4ee941e073a821ee1ff\":{\"cash\":109755508503386757517651,\"liability\":111705981295320099096585,\"maxSupply\":1000000000000000000000000000,\"totalSupply\":111566749817365287931821,\"decimals\":6,\"token\":\"0x176211869ca2b568f2a7d4ee941e073a821ee1ff\",\"active\":true},\"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5\":{\"cash\":31982154523056912809541,\"liability\":32315214600033595639643,\"maxSupply\":1000000000000000000000000000,\"totalSupply\":32294182514254242328808,\"decimals\":18,\"token\":\"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5\",\"active\":true},\"0xa219439258ca9da29e9cc4ce5596924745e12b93\":{\"cash\":108295197536453495651912,\"liability\":106011578468650285163146,\"maxSupply\":1000000000000000000000000000,\"totalSupply\":105904081724488185350374,\"decimals\":6,\"token\":\"0xa219439258ca9da29e9cc4ce5596924745e12b93\",\"active\":true}},\"indexByToken\":{\"0x176211869ca2b568f2a7d4ee941e073a821ee1ff\":0,\"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5\":2,\"0xa219439258ca9da29e9cc4ce5596924745e12b93\":1},\"a\":1000000000000000,\"lpFee\":100000000000000,\"paused\":false}",
		}

		poolSimulator, err := NewPoolSimulator(entityPool)

		assert.Nil(t, err)
		assert.False(t, poolSimulator.paused)
	})
}

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Run("it should return correct amount out", func(t *testing.T) {
		poolSimulator := PoolSimulator{
			a:      utils.NewBig("1000000000000000"),
			lpFee:  utils.NewBig("100000000000000"),
			paused: false,

			assetByToken: map[string]Asset{
				"0x176211869ca2b568f2a7d4ee941e073a821ee1ff": {
					Cash:        utils.NewBig("109755508503386757517651"),
					Liability:   utils.NewBig("111705981295320099096585"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("111566749817365287931821"),
					Decimals:    6,
					Token:       common.HexToAddress("0x176211869ca2b568f2a7d4ee941e073a821ee1ff"),
					Active:      true,
				},
				"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5": {
					Cash:        utils.NewBig("31982154523056912809541"),
					Liability:   utils.NewBig("32315214600033595639643"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("32294182514254242328808"),
					Decimals:    18,
					Token:       common.HexToAddress("0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5"),
					Active:      true,
				},
				"0xa219439258ca9da29e9cc4ce5596924745e12b93": {
					Cash:        utils.NewBig("108295197536453495651912"),
					Liability:   utils.NewBig("106011578468650285163146"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("105904081724488185350374"),
					Decimals:    6,
					Token:       common.HexToAddress("0xa219439258ca9da29e9cc4ce5596924745e12b93"),
					Active:      true,
				},
			},
			indexByToken: map[string]int{
				"0x176211869ca2b568f2a7d4ee941e073a821ee1ff": 0,
				"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5": 2,
				"0xa219439258ca9da29e9cc4ce5596924745e12b93": 1,
			},

			gas: defaultGas,
		}

		result, err := poolSimulator.CalcAmountOut(
			poolpkg.TokenAmount{
				Token:  "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
				Amount: utils.NewBig("1000000"),
			},
			"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5",
		)

		assert.Nil(t, err)
		assert.Equal(t, result.TokenAmountOut.Amount.Cmp(utils.NewBig("999914863605742941")), 0)
	})

	t.Run("it returns correct error when the pool is paused", func(t *testing.T) {
		poolSimulator := PoolSimulator{
			paused: true,
		}

		result, err := poolSimulator.CalcAmountOut(
			poolpkg.TokenAmount{
				Token:  "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
				Amount: utils.NewBig("1000000"),
			},
			"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5",
		)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrPoolIsPaused)
	})

	t.Run("it returns correct errors when tokenIn is tokenOut", func(t *testing.T) {
		poolSimulator := PoolSimulator{
			paused: false,
		}

		result, err := poolSimulator.CalcAmountOut(
			poolpkg.TokenAmount{
				Token:  "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
				Amount: utils.NewBig("1000000"),
			},
			"0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
		)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrSameAddress)
	})

	t.Run("it returns correct errors when amountIn <= 0", func(t *testing.T) {
		poolSimulator := PoolSimulator{
			paused: false,
		}

		result, err := poolSimulator.CalcAmountOut(
			poolpkg.TokenAmount{
				Token:  "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
				Amount: utils.NewBig("-1"),
			},
			"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5",
		)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrInvalidValue)
	})

	t.Run("it returns correct errors when tokenIn is inactive", func(t *testing.T) {
		poolSimulator := PoolSimulator{
			a:      utils.NewBig("1000000000000000"),
			lpFee:  utils.NewBig("100000000000000"),
			paused: false,

			assetByToken: map[string]Asset{
				"0x176211869ca2b568f2a7d4ee941e073a821ee1ff": {
					Cash:        utils.NewBig("109755508503386757517651"),
					Liability:   utils.NewBig("111705981295320099096585"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("111566749817365287931821"),
					Decimals:    6,
					Token:       common.HexToAddress("0x176211869ca2b568f2a7d4ee941e073a821ee1ff"),
					Active:      false,
				},
				"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5": {
					Cash:        utils.NewBig("31982154523056912809541"),
					Liability:   utils.NewBig("32315214600033595639643"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("32294182514254242328808"),
					Decimals:    18,
					Token:       common.HexToAddress("0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5"),
					Active:      true,
				},
				"0xa219439258ca9da29e9cc4ce5596924745e12b93": {
					Cash:        utils.NewBig("108295197536453495651912"),
					Liability:   utils.NewBig("106011578468650285163146"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("105904081724488185350374"),
					Decimals:    6,
					Token:       common.HexToAddress("0xa219439258ca9da29e9cc4ce5596924745e12b93"),
					Active:      true,
				},
			},
			indexByToken: map[string]int{
				"0x176211869ca2b568f2a7d4ee941e073a821ee1ff": 0,
				"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5": 2,
				"0xa219439258ca9da29e9cc4ce5596924745e12b93": 1,
			},

			gas: defaultGas,
		}

		result, err := poolSimulator.CalcAmountOut(
			poolpkg.TokenAmount{
				Token:  "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
				Amount: utils.NewBig("1000000"),
			},
			"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5",
		)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrAssetDeactivated)
	})

	t.Run("it returns correct errors when tokenOut is inactive", func(t *testing.T) {
		poolSimulator := PoolSimulator{
			a:      utils.NewBig("1000000000000000"),
			lpFee:  utils.NewBig("100000000000000"),
			paused: false,

			assetByToken: map[string]Asset{
				"0x176211869ca2b568f2a7d4ee941e073a821ee1ff": {
					Cash:        utils.NewBig("109755508503386757517651"),
					Liability:   utils.NewBig("111705981295320099096585"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("111566749817365287931821"),
					Decimals:    6,
					Token:       common.HexToAddress("0x176211869ca2b568f2a7d4ee941e073a821ee1ff"),
					Active:      true,
				},
				"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5": {
					Cash:        utils.NewBig("31982154523056912809541"),
					Liability:   utils.NewBig("32315214600033595639643"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("32294182514254242328808"),
					Decimals:    18,
					Token:       common.HexToAddress("0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5"),
					Active:      false,
				},
				"0xa219439258ca9da29e9cc4ce5596924745e12b93": {
					Cash:        utils.NewBig("108295197536453495651912"),
					Liability:   utils.NewBig("106011578468650285163146"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("105904081724488185350374"),
					Decimals:    6,
					Token:       common.HexToAddress("0xa219439258ca9da29e9cc4ce5596924745e12b93"),
					Active:      true,
				},
			},
			indexByToken: map[string]int{
				"0x176211869ca2b568f2a7d4ee941e073a821ee1ff": 0,
				"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5": 2,
				"0xa219439258ca9da29e9cc4ce5596924745e12b93": 1,
			},

			gas: defaultGas,
		}

		result, err := poolSimulator.CalcAmountOut(
			poolpkg.TokenAmount{
				Token:  "0x176211869ca2b568f2a7d4ee941e073a821ee1ff",
				Amount: utils.NewBig("1000000"),
			},
			"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5",
		)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrAssetDeactivated)
	})
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Run("it should update pool state correctly", func(t *testing.T) {
		poolSimulator := PoolSimulator{
			a:      utils.NewBig("1000000000000000"),
			lpFee:  utils.NewBig("100000000000000"),
			paused: false,

			assetByToken: map[string]Asset{
				"0x176211869ca2b568f2a7d4ee941e073a821ee1ff": {
					Cash:        utils.NewBig("109755508503386757517651"),
					Liability:   utils.NewBig("111705981295320099096585"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("111566749817365287931821"),
					Decimals:    6,
					Token:       common.HexToAddress("0x176211869ca2b568f2a7d4ee941e073a821ee1ff"),
					Active:      true,
				},
				"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5": {
					Cash:        utils.NewBig("31982154523056912809541"),
					Liability:   utils.NewBig("32315214600033595639643"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("32294182514254242328808"),
					Decimals:    18,
					Token:       common.HexToAddress("0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5"),
					Active:      true,
				},
				"0xa219439258ca9da29e9cc4ce5596924745e12b93": {
					Cash:        utils.NewBig("108295197536453495651912"),
					Liability:   utils.NewBig("106011578468650285163146"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("105904081724488185350374"),
					Decimals:    6,
					Token:       common.HexToAddress("0xa219439258ca9da29e9cc4ce5596924745e12b93"),
					Active:      true,
				},
			},
			indexByToken: map[string]int{
				"0x176211869ca2b568f2a7d4ee941e073a821ee1ff": 0,
				"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5": 2,
				"0xa219439258ca9da29e9cc4ce5596924745e12b93": 1,
			},

			gas: defaultGas,
		}

		poolSimulator.UpdateBalance(poolpkg.UpdateBalanceParams{
			TokenAmountIn:  poolpkg.TokenAmount{Token: "0x176211869ca2b568f2a7d4ee941e073a821ee1ff", Amount: big.NewInt(1000000)},
			TokenAmountOut: poolpkg.TokenAmount{Token: "0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5", Amount: big.NewInt(937047)},
			Fee:            poolpkg.TokenAmount{Token: "0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5", Amount: big.NewInt(12)},
		})

		assert.Equal(t, poolSimulator.assetByToken["0x176211869ca2b568f2a7d4ee941e073a821ee1ff"].Cash.Cmp(utils.NewBig("109755508503386758517651")), 0)
		assert.Equal(t, poolSimulator.assetByToken["0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5"].Cash.Cmp(utils.NewBig("31982154523056911872506")), 0)
	})
}

func TestPoolSimulator_GetMetaInfo(t *testing.T) {
	t.Run("it should return correct meta", func(t *testing.T) {
		poolSimulator := PoolSimulator{
			a:      utils.NewBig("1000000000000000"),
			lpFee:  utils.NewBig("100000000000000"),
			paused: false,

			assetByToken: map[string]Asset{
				"0x176211869ca2b568f2a7d4ee941e073a821ee1ff": {
					Cash:        utils.NewBig("109755508503386757517651"),
					Liability:   utils.NewBig("111705981295320099096585"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("111566749817365287931821"),
					Decimals:    6,
					Token:       common.HexToAddress("0x176211869ca2b568f2a7d4ee941e073a821ee1ff"),
					Active:      true,
				},
				"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5": {
					Cash:        utils.NewBig("31982154523056912809541"),
					Liability:   utils.NewBig("32315214600033595639643"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("32294182514254242328808"),
					Decimals:    18,
					Token:       common.HexToAddress("0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5"),
					Active:      true,
				},
				"0xa219439258ca9da29e9cc4ce5596924745e12b93": {
					Cash:        utils.NewBig("108295197536453495651912"),
					Liability:   utils.NewBig("106011578468650285163146"),
					MaxSupply:   utils.NewBig("1000000000000000000000000000"),
					TotalSupply: utils.NewBig("105904081724488185350374"),
					Decimals:    6,
					Token:       common.HexToAddress("0xa219439258ca9da29e9cc4ce5596924745e12b93"),
					Active:      true,
				},
			},
			indexByToken: map[string]int{
				"0x176211869ca2b568f2a7d4ee941e073a821ee1ff": 0,
				"0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5": 2,
				"0xa219439258ca9da29e9cc4ce5596924745e12b93": 1,
			},

			gas: defaultGas,
		}

		metaI := poolSimulator.GetMetaInfo("0x176211869ca2b568f2a7d4ee941e073a821ee1ff", "0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5")
		meta, ok := metaI.(PoolSimulatorMetadata)

		assert.True(t, ok)
		assert.Equal(t, 0, meta.FromID)
		assert.Equal(t, 2, meta.ToID)
	})
}
