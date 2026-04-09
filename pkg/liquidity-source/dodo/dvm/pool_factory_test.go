package dvm

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
	// https://etherscan.io/tx/0x3f624f211f11e25f16ad9554ee4edd129b4b53f84a03ec2d5b6ec1a0ca1a27af#eventlog
	factory := NewPoolFactory(&shared.Config{
		DexID: "dodo-dvm",
	})
	pool, err := factory.DecodePoolCreated(types.Log{
		Address: common.HexToAddress("0x8f11519f4f7c498e1f940b9de187d9c390321016"),
		Topics:  []common.Hash{common.HexToHash("0xaf5c5f12a80fc937520df6fcaed66262a4cc775e0f3fceaf7a7cfe476d9a751d")},
		Data:    common.FromHex("0x0000000000000000000000009f3f332e3238a5123fc8c03fd213ed3ca7cddd46000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc20000000000000000000000004876fc01ae0c51a79a422eca6da6d785b6e822ca000000000000000000000000e4e667b380a8fd317d06a6b1f5b3b5e4ee0d14ad"),
	})
	require.NoError(t, err)
	assert.Equal(t, "0xe4e667b380a8fd317d06a6b1f5b3b5e4ee0d14ad", pool.Address)
	assert.Equal(t, "dodo-dvm", pool.Exchange)
	assert.Equal(t, "dodo-dvm", pool.Type)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, pool.Reserves)
}
