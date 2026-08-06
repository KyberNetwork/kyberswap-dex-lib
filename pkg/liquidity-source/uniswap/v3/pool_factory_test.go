package uniswapv3

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDecodePoolCreated_SolidlyShape covers the shape that shares a topic0 with
// uniswap-v3/pancake-v3/ramses-v2/nuri-v2 (same param types) but swaps which field is
// indexed (fee not indexed, tickSpacing indexed). It must still decode token0/token1/pool
// correctly since decoding never reads fee/tickSpacing positionally.
// https://etherscan.io/tx/0x9d2c75e29a60bb0b6a56e94135bc1ff999e68b870b0d799da02bc47801b2c775
func TestDecodePoolCreated_SolidlyShape(t *testing.T) {
	t.Parallel()

	event := types.Log{
		Address: common.HexToAddress("0x70Fe4a44EA505cFa3A57b95cF2862D4fd5F0f687"),
		Topics: []common.Hash{
			common.HexToHash("0x783cca1c0412dd0d695e784568c96da2e9c22ff989357a2e8b1d9b2b4e6b7118"),
			common.HexToHash("0x000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"),
			common.HexToHash("0x000000000000000000000000d555498a524612c67f286df0e0a9a64a73a7cdc7"),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000064"),
		},
		Data: common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000027100000000000000000000000006339962d8b80749ce86d65affc0b2a4290aef42f"),
	}

	factory := NewSolidlyV3PoolFactory(&Config{DexID: "solidlyv3"})
	require.True(t, factory.IsEventSupported(event.Topics[0]))

	p, err := factory.DecodePoolCreated(event)
	require.NoError(t, err)

	assert.Equal(t, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", p.Tokens[0].Address)
	assert.Equal(t, "0xd555498a524612c67f286df0e0a9a64a73a7cdc7", p.Tokens[1].Address)
	assert.Equal(t, "0x6339962d8b80749ce86d65affc0b2a4290aef42f", p.Address)
	assert.Equal(t, DexTypeSolidlyV3, p.Type)
	assert.Equal(t, "solidlyv3", p.Exchange)
	// fee/tickSpacing are deliberately left at zero here; the tracker's first refresh
	// resolves them generically regardless of which fork this pool belongs to.
	assert.Equal(t, float64(0), p.SwapFee)
}

// TestDecodePoolCreated_SlipstreamShape covers the only shape with a genuinely different
// topic0 (no fee param at all).
// https://optimistic.etherscan.io/tx/0xc5c94ecb2ba3e552b6e59b5c87765a23caefa296e23e2dc948ff23b0e9298bd9
func TestDecodePoolCreated_SlipstreamShape(t *testing.T) {
	t.Parallel()

	event := types.Log{
		Address: common.HexToAddress("0x548118C7E0B865C2CfA94D15EC86B666468ac758"),
		Topics: []common.Hash{
			common.HexToHash("0xab0d57f0df537bb25e80245ef7748fa62353808c54d6e528a9dd20887aed9ac2"),
			common.HexToHash("0x0000000000000000000000000b2c639c533813f4aa9d7837caf62653d097ff85"),
			common.HexToHash("0x000000000000000000000000dfa46478f9e5ea86d57387849598dbfb2e964b02"),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
		},
		Data: common.Hex2Bytes("00000000000000000000000020efb6b14640fa4e20cd04456ccf8bba9937307b"),
	}

	factory := NewSlipstreamPoolFactory(&Config{DexID: "slipstream-x"})
	require.True(t, factory.IsEventSupported(event.Topics[0]))

	p, err := factory.DecodePoolCreated(event)
	require.NoError(t, err)

	assert.Equal(t, "0x0b2c639c533813f4aa9d7837caf62653d097ff85", p.Tokens[0].Address)
	assert.Equal(t, "0xdfa46478f9e5ea86d57387849598dbfb2e964b02", p.Tokens[1].Address)
	assert.Equal(t, "0x20efb6b14640fa4e20cd04456ccf8bba9937307b", p.Address)
	assert.Equal(t, DexTypeSlipstream, p.Type)
	assert.Equal(t, "slipstream-x", p.Exchange)
	assert.Equal(t, float64(0), p.SwapFee)
}

func TestPoolFactory_IsEventSupported(t *testing.T) {
	t.Parallel()

	factory := NewUniswapV3PoolFactory(&Config{DexID: "uniswapv3"})
	assert.True(t, factory.IsEventSupported(poolCreatedEventIDWithFee))
	assert.True(t, factory.IsEventSupported(poolCreatedEventIDNoFee))
	assert.False(t, factory.IsEventSupported(mintEventID))
}
