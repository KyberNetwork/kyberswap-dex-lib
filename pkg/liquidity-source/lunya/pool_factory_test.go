package lunya

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func TestEventIDs(t *testing.T) {
	t.Parallel()

	assert.Equal(t, common.HexToHash("0x3871766f55926cc6499881a4481d190672266d76354ee598765dea432553fac7"),
		poolCreatedEvent.ID)
	// Mint and Burn are Uniswap V3's, byte for byte.
	assert.Equal(t, crypto.Keccak256Hash([]byte("Mint(address,address,int24,int24,uint128,uint256,uint256)")),
		mintEvent.ID)
	assert.Equal(t, crypto.Keccak256Hash([]byte("Burn(address,int24,int24,uint128,uint256,uint256)")), burnEvent.ID)
}

func poolCreatedLog(t *testing.T, factory string, poolType int64) types.Log {
	t.Helper()

	data, err := poolCreatedEvent.Inputs.NonIndexed().Pack(big.NewInt(60), big.NewInt(500),
		common.HexToAddress(testnetCLPool))
	require.NoError(t, err)

	return types.Log{
		Address: common.HexToAddress(factory),
		Topics: []common.Hash{
			poolCreatedEvent.ID,
			common.BytesToHash(common.HexToAddress(arcTestnetUSDC).Bytes()),
			common.BytesToHash(common.HexToAddress(testnetCLToken1).Bytes()),
			common.BigToHash(big.NewInt(poolType)),
		},
		Data:        data,
		BlockNumber: testnetCLPoolBlock,
	}
}

func TestDecodePoolCreated(t *testing.T) {
	t.Parallel()

	decoder := NewPoolFactoryDecoder(testnetConfig())

	for _, poolType := range []int64{poolTypeCL, poolTypeCP, poolTypeStable} {
		t.Run("pool type", func(t *testing.T) {
			p, err := decoder.DecodePoolCreated(poolCreatedLog(t, arcTestnetFactory, poolType))
			require.NoError(t, err)
			require.NotNil(t, p)

			assert.Equal(t, testnetCLPool, p.Address)
			assert.Equal(t, DexType, p.Type)
			assert.Equal(t, DexType, p.Exchange)
			assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
			assert.Equal(t, arcTestnetUSDC, p.Tokens[0].Address)
			assert.Equal(t, testnetCLToken1, p.Tokens[1].Address)
			assert.True(t, p.Tokens[0].Swappable && p.Tokens[1].Swappable)
			assert.EqualValues(t, testnetCLPoolBlock, p.BlockNumber)

			var staticExtra StaticExtra
			require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &staticExtra))
			assert.EqualValues(t, poolType, staticExtra.PoolType)
		})
	}

	t.Run("a pool type this source cannot price is left out", func(t *testing.T) {
		p, err := decoder.DecodePoolCreated(poolCreatedLog(t, arcTestnetFactory, poolTypeStable+1))
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("another factory is ignored", func(t *testing.T) {
		p, err := decoder.DecodePoolCreated(poolCreatedLog(t, "0x0000000000000000000000000000000000000001", poolTypeCL))
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("the Uniswap V3-shaped compatibility event is not decoded", func(t *testing.T) {
		assert.False(t, decoder.IsEventSupported(
			crypto.Keccak256Hash([]byte("PoolCreated(address,address,uint24,int24,address)"))))
	})
}
