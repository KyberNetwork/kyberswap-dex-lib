package lunyafun

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// The Lunya launchpad on Arc testnet, and three of the launches its factory has listed: two still on
// their curve and one that has graduated to a DEX pool.
const (
	arcTestnetRPC        = "https://rpc.testnet.arc.network"
	arcTestnetMulticall3 = "0xcA11bde05977b3631167028862be2a173976CA11"
	arcTestnetFactory    = "0x59a9d2037f321320a28e29083570b045f430d03a"
	arcTestnetUSDC       = "0x3600000000000000000000000000000000000000"

	testnetLaunch           = "0xef3f2667979ec8ce67c115dd2134eeddbd91f516"
	testnetLaunchBlock      = 61812087
	testnetLaunchToken      = "0x33cfca858036d4ff663c51263c4d6c96f8c54aaf"
	testnetLaunch2          = "0x3b32782cccc4001c4978281ba1907b988a080199"
	testnetLaunch2Block     = 61812092
	testnetGraduated        = "0x48d2c2e63a35b15ee5c88e46e0f27e032182ea8d"
	testnetGraduatedBlock   = 61812083
	testnetLaunchCreator    = "0xec95b9ecb93c9b475a87e1930e04d0114337d987"
	testnetLaunchStartBlock = 61811877
)

func testnetConfig() *Config {
	return &Config{DexID: DexType, Factory: arcTestnetFactory}
}

func launchCreatedLog(t *testing.T, factory string, launchType uint8) types.Log {
	t.Helper()

	data, err := launchCreatedEvent.Inputs.NonIndexed().Pack(launchType, common.HexToAddress(arcTestnetUSDC),
		"A launch", "LAUNCH", "ipfs://contract", []byte{})
	require.NoError(t, err)

	return types.Log{
		Address: common.HexToAddress(factory),
		Topics: []common.Hash{
			launchCreatedEvent.ID,
			common.BytesToHash(common.HexToAddress(testnetLaunch).Bytes()),
			common.BytesToHash(common.HexToAddress(testnetLaunchToken).Bytes()),
			common.BytesToHash(common.HexToAddress(testnetLaunchCreator).Bytes()),
		},
		Data:        data,
		BlockNumber: testnetLaunchBlock,
	}
}

func TestDecodePoolCreated(t *testing.T) {
	t.Parallel()

	decoder := NewPoolFactoryDecoder(testnetConfig())

	t.Run("a CP launch", func(t *testing.T) {
		p, err := decoder.DecodePoolCreated(launchCreatedLog(t, arcTestnetFactory, launchTypeCP))
		require.NoError(t, err)
		require.NotNil(t, p)

		assert.Equal(t, testnetLaunch, p.Address, "the launch contract is the pool")
		assert.Equal(t, DexType, p.Type)
		assert.Equal(t, DexType, p.Exchange)
		assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
		assert.Equal(t, arcTestnetUSDC, p.Tokens[0].Address, "the quote token comes first")
		assert.Equal(t, testnetLaunchToken, p.Tokens[1].Address)
		assert.True(t, p.Tokens[0].Swappable && p.Tokens[1].Swappable)
		assert.EqualValues(t, testnetLaunchBlock, p.BlockNumber)

		var staticExtra StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &staticExtra))
		assert.EqualValues(t, launchTypeCP, staticExtra.LaunchType)
	})

	t.Run("a curve this source cannot price is left out", func(t *testing.T) {
		p, err := decoder.DecodePoolCreated(launchCreatedLog(t, arcTestnetFactory, launchTypeCP+1))
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("another factory is ignored", func(t *testing.T) {
		p, err := decoder.DecodePoolCreated(
			launchCreatedLog(t, "0x0000000000000000000000000000000000000001", launchTypeCP))
		require.NoError(t, err)
		assert.Nil(t, p)
	})
}
