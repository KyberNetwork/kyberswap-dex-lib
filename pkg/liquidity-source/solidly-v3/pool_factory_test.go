package solidlyv3

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

func TestSolidlyV3Decoder(t *testing.T) {
	// https://etherscan.io/tx/0x9d2c75e29a60bb0b6a56e94135bc1ff999e68b870b0d799da02bc47801b2c775
	event := types.Log{
		Address: common.HexToAddress("0x70Fe4a44EA505cFa3A57b95cF2862D4fd5F0f687"),
		Topics: []common.Hash{
			common.HexToHash("0x783cca1c0412dd0d695e784568c96da2e9c22ff989357a2e8b1d9b2b4e6b7118"),
			common.HexToHash("0x000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"),
			common.HexToHash("0x000000000000000000000000d555498a524612c67f286df0e0a9a64a73a7cdc7"),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000064"),
		},
		Data:        common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000027100000000000000000000000006339962d8b80749ce86d65affc0b2a4290aef42f"),
		BlockNumber: 19178289,
		TxHash:      common.HexToHash("0x9d2c75e29a60bb0b6a56e94135bc1ff999e68b870b0d799da02bc47801b2c775"),
		TxIndex:     26,
		BlockHash:   common.HexToHash("0x30df901df9ea1af28b88bbc268a2c3659613bd6659f9f630d02ffb21f7c3dd49"),
		Index:       132,
		Removed:     false,
	}

	factory := poolfactory.Factory(DexTypeSolidlyV3)
	assert.NotNil(t, factory)

	decoder, err := factory("solidlyv3", poolfactory.FactoryParams{})
	assert.NoError(t, err)

	pool, err := decoder.DecodePoolCreated(event)
	assert.NoError(t, err)

	assert.Equal(t, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", pool.Tokens[0].Address)
	assert.Equal(t, "0xd555498a524612c67f286df0e0a9a64a73a7cdc7", pool.Tokens[1].Address)
	assert.Equal(t, float64(10000), pool.SwapFee)
	assert.Equal(t, "0x6339962d8b80749ce86d65affc0b2a4290aef42f", pool.Address)
	assert.Equal(t, DexTypeSolidlyV3, pool.Type)
	assert.Equal(t, "solidlyv3", pool.Exchange)
}
