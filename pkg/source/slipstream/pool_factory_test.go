package slipstream

import (
	"testing"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

func TestSlipstreamDecoder(t *testing.T) {
	// https://optimistic.etherscan.io/tx/0xc5c94ecb2ba3e552b6e59b5c87765a23caefa296e23e2dc948ff23b0e9298bd9
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

	factory := poolfactory.Factory(DexType)
	assert.NotNil(t, factory)

	decoder, err := factory("slipstream-x", poolfactory.FactoryParams{})
	assert.NoError(t, err)

	pool, err := decoder.DecodePoolCreated(event)
	assert.NoError(t, err)

	assert.Equal(t, "0x0b2c639c533813f4aa9d7837caf62653d097ff85", pool.Tokens[0].Address)
	assert.Equal(t, "0xdfa46478f9e5ea86d57387849598dbfb2e964b02", pool.Tokens[1].Address)
	assert.Equal(t, float64(0), pool.SwapFee) // No feeTier
	assert.Equal(t, "0x20efb6b14640fa4e20cd04456ccf8bba9937307b", pool.Address)

	var extra uniswapv3.Extra
	err = json.Unmarshal([]byte(pool.Extra), &extra)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), extra.TickSpacing) // No feeTier
	assert.Equal(t, DexType, pool.Type)
	assert.Equal(t, "slipstream-x", pool.Exchange)
}
