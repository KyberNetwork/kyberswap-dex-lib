package slyngfun

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Slyng on Robinhood Chain mainnet, and SLYNG, the launchpad's own coin: created in block
// 63269245 by tx 0x9d78fdebafe25ad4ba9d3a7b9967d5e67f474a76f8ff5103d650b8d8c01a9fe6, priced in ETH.
const (
	robinhoodRPC        = "https://rpc.mainnet.chain.robinhood.com"
	robinhoodMulticall3 = "0xcA11bde05977b3631167028862be2a173976CA11"
	robinhoodWETH       = "0x0bd7d308f8e1639fab988df18a8011f41eacad73"
	mainnetLaunchpad    = "0xCe0ABC33eC4264377045Ae17F9B887DB33Cc95B4"
	mainnetDeployBlock  = 60372065

	slyngCreatedBlock = 63269245
	slyngCreator      = "0x6d3e4ec533d34dd87db1117d0b3b650ebb45e2a2"
	usdg              = "0x5fc5360d0400a0fd4f2af552add042d716f1d168"
)

func mainnetConfig() *Config {
	return &Config{DexID: DexType, ChainID: valueobject.ChainIDRobinhood, Launchpad: mainnetLaunchpad}
}

func tokenCreatedLog(t *testing.T, launchpad, quote string, graduationTarget *big.Int) types.Log {
	t.Helper()

	data, err := tokenCreatedEvent.Inputs.NonIndexed().Pack("SLYNG", "SLYNG", big.NewInt(2_592_000),
		big.NewInt(1792027818), graduationTarget)
	require.NoError(t, err)

	return types.Log{
		Address: common.HexToAddress(launchpad),
		Topics: []common.Hash{
			tokenCreatedEvent.ID,
			common.BytesToHash(common.HexToAddress(slyngToken).Bytes()),
			common.BytesToHash(common.HexToAddress(slyngCreator).Bytes()),
			common.BytesToHash(common.HexToAddress(quote).Bytes()),
		},
		Data:        data,
		BlockNumber: slyngCreatedBlock,
	}
}

func TestDecodePoolCreated(t *testing.T) {
	t.Parallel()

	decoder := NewPoolFactoryDecoder(mainnetConfig())

	t.Run("an ETH coin is listed under the wrapped native", func(t *testing.T) {
		p, err := decoder.DecodePoolCreated(tokenCreatedLog(t, mainnetLaunchpad,
			"0x0000000000000000000000000000000000000000", big.NewInt(5e18)))
		require.NoError(t, err)
		require.NotNil(t, p)

		assert.Equal(t, slyngToken, p.Address, "the coin is the pool")
		assert.Equal(t, DexType, p.Type)
		assert.Equal(t, DexType, p.Exchange)
		assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
		assert.Equal(t, robinhoodWETH, p.Tokens[0].Address, "the quote comes first")
		assert.Equal(t, slyngToken, p.Tokens[1].Address)
		assert.True(t, p.Tokens[0].Swappable && p.Tokens[1].Swappable)
		assert.EqualValues(t, slyngCreatedBlock, p.BlockNumber)

		var staticExtra StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &staticExtra))
		assert.Equal(t, slyngLaunchpad, staticExtra.Launchpad)
		assert.True(t, staticExtra.IsNativeQuote)
		assert.Equal(t, "5000000000000000000", staticExtra.GraduationTarget.Dec())
	})

	t.Run("an ERC-20 coin keeps its quote", func(t *testing.T) {
		p, err := decoder.DecodePoolCreated(tokenCreatedLog(t, mainnetLaunchpad, usdg, big.NewInt(12_545_000_000)))
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, usdg, p.Tokens[0].Address)

		var staticExtra StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &staticExtra))
		assert.False(t, staticExtra.IsNativeQuote)
		assert.Equal(t, "12545000000", staticExtra.GraduationTarget.Dec())
	})

	t.Run("another contract is ignored", func(t *testing.T) {
		p, err := decoder.DecodePoolCreated(tokenCreatedLog(t, "0x0000000000000000000000000000000000000001",
			usdg, big.NewInt(1)))
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("a truncated log is an error, not a pool", func(t *testing.T) {
		log := tokenCreatedLog(t, mainnetLaunchpad, usdg, big.NewInt(1))
		log.Topics = log.Topics[:3]
		_, err := decoder.DecodePoolCreated(log)
		assert.ErrorIs(t, err, ErrMalformedLog)
	})
}

func TestDecodePoolAddressesFromFactoryLog(t *testing.T) {
	t.Parallel()

	decoder := NewPoolFactoryDecoder(mainnetConfig())
	tokenTopic := common.BytesToHash(common.HexToAddress(slyngToken).Bytes())

	for _, ev := range []struct {
		name string
		id   common.Hash
	}{{"Bought", boughtEvent.ID}, {"Sold", soldEvent.ID}, {"Graduated", graduatedEvent.ID}} {
		t.Run(ev.name+" refreshes the coin it names", func(t *testing.T) {
			addrs, err := decoder.DecodePoolAddressesFromFactoryLog(t.Context(), types.Log{
				Address: common.HexToAddress(mainnetLaunchpad),
				Topics:  []common.Hash{ev.id, tokenTopic},
			})
			require.NoError(t, err)
			assert.Equal(t, []string{slyngToken}, addrs)
		})
	}

	t.Run("TokenCreated is discovery, not a refresh", func(t *testing.T) {
		addrs, err := decoder.DecodePoolAddressesFromFactoryLog(t.Context(),
			tokenCreatedLog(t, mainnetLaunchpad, usdg, big.NewInt(1)))
		require.NoError(t, err)
		assert.Nil(t, addrs)
	})

	t.Run("another contract's log is ignored", func(t *testing.T) {
		addrs, err := decoder.DecodePoolAddressesFromFactoryLog(t.Context(), types.Log{
			Address: common.HexToAddress("0x0000000000000000000000000000000000000001"),
			Topics:  []common.Hash{boughtEvent.ID, tokenTopic},
		})
		require.NoError(t, err)
		assert.Nil(t, addrs)
	})
}
