package dsp

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
	// https://etherscan.io/tx/0x9eade01c80e7e63ac1921d6648ff93353018cf0455c76eb379c76c8c5e7f8b86#eventlog
	factory := NewPoolFactory(&shared.Config{
		DexID: "dodo-dsp",
	})
	pool, err := factory.DecodePoolCreated(types.Log{
		Address: common.HexToAddress("0x8f11519f4f7c498e1f940b9de187d9c390321016"),
		Topics:  []common.Hash{common.HexToHash("0xbc1083a2c1c5ef31e13fb436953d22b47880cf7db279c2c5666b16083afd6b9d")},
		Data:    common.FromHex("0x0000000000000000000000001abaea1f7c830bd89acc67ec4af516284b1bc33c0000000000000000000000005f7827fdeb7c20b443265fc2f40845b715385ff20000000000000000000000004fc356c863d1306497a2e6635bbb063e0f564bdb0000000000000000000000005a94174992634c05c9dce6b0edb9fecabd51773d"),
	})
	require.NoError(t, err)
	assert.Equal(t, "0x5a94174992634c05c9dce6b0edb9fecabd51773d", pool.Address)
	assert.Equal(t, "dodo-dsp", pool.Exchange)
	assert.Equal(t, "dodo-dsp", pool.Type)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, pool.Reserves)
}
