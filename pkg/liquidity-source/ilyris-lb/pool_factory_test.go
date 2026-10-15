package ilyrislb

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The pool is data word 3 of PoolCreated; keying the entity by the emitter (the factory) would
// collapse every pool into one row.
func TestDecodePoolCreated(t *testing.T) {
	require.Equal(t, factoryABI.Events["PoolCreated"].ID, poolCreatedTopic)

	const poolAddr = "0xeec7cdd6834b4f1fea1d63d32ee0cb7cbb3f24d7"
	data := make([]byte, 128)
	big.NewInt(1).FillBytes(data[:32])
	copy(data[96+12:], common.HexToAddress(poolAddr).Bytes())
	topics := []common.Hash{poolCreatedTopic,
		common.BytesToHash(common.HexToAddress(tokX).Bytes()),
		common.BytesToHash(common.HexToAddress(tokY).Bytes()),
		common.BigToHash(big.NewInt(10))}
	f := NewPoolFactory(&Config{DexID: DexType})

	p, err := f.DecodePoolCreated(types.Log{Address: common.HexToAddress("0x3Bf76F2E41Ac7996c822455f4c78fa2026465C4D"),
		Topics: topics, Data: data})
	require.NoError(t, err)
	assert.Equal(t, poolAddr, p.Address)
	assert.Equal(t, tokX, p.Tokens[0].Address)
	assert.Equal(t, tokY, p.Tokens[1].Address)

	for name, ev := range map[string]types.Log{
		"short data":     {Topics: topics, Data: data[:96]},
		"missing topics": {Topics: topics[:1], Data: data},
		"zero pool":      {Topics: topics, Data: make([]byte, 128)},
		"wrong topic":    {Topics: append([]common.Hash{{1}}, topics[1:]...), Data: data},
	} {
		_, err := f.DecodePoolCreated(ev)
		assert.Error(t, err, name)
	}
}
