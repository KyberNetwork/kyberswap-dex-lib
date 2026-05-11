package dpp

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	shared "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/shared"
)

func TestPoolFactory_DecodePoolCreated(t *testing.T) {
	// https://etherscan.io/tx/0xece024c278f0d063d41c6caf3135e65529f005862b374097bcfb17fb9af17776#eventlog
	factory := NewPoolFactory(&shared.Config{
		DexID: "dodo-dpp",
	})
	pool, err := factory.DecodePoolCreated(types.Log{
		Address: common.HexToAddress("0x8f11519f4f7c498e1f940b9de187d9c390321016"),
		Topics:  []common.Hash{common.HexToHash("0x8494fe594cd5087021d4b11758a2bbc7be28a430e94f2b268d668e5991ed3b8a")},
		Data:    common.FromHex("0x000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2000000000000000000000000b5274d9f803b4b15607c6e180ef8f68fe990f73b0000000000000000000000008b939f6dddc85eea61ed1ca0fdb853dd6c2b455500000000000000000000000089acc537e9fea324f4f6c2e91db4498e0eda05f6"),
	})
	require.NoError(t, err)
	assert.Equal(t, "0x89acc537e9fea324f4f6c2e91db4498e0eda05f6", pool.Address)
	assert.Equal(t, "dodo-dpp", pool.Exchange)
	assert.Equal(t, "dodo-dpp", pool.Type)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, pool.Reserves)
}
