package dexv2

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2/abis"
)

var testDexID = common.HexToHash("0xe018f09af38956affdfeab72c2cefbcd4e6fee44d09df7525ec9dba3e51356a5")

// positionLog builds a LogDeposit/LogWithdraw/LogBorrow/LogPayback with the deployed layout:
// indexed dexType, dexId, user; data = tickLower, tickUpper, salt, amount0, amount1, fee0, fee1, liquidity.
func positionLog(t *testing.T, name string, dexType int64, lower, upper int64, liquidity int64) ethtypes.Log {
	t.Helper()
	ev := abis.DexV2ABI.Events[name]
	data, err := ev.Inputs.NonIndexed().Pack(big.NewInt(lower), big.NewInt(upper), [32]byte{},
		big.NewInt(1), big.NewInt(1), big.NewInt(0), big.NewInt(0), big.NewInt(liquidity))
	require.NoError(t, err)
	return ethtypes.Log{
		Address: common.HexToAddress("0xaBE4D5AA1Bb188f4B5d3a1aE3B8ec5e300A78261"),
		Topics:  []common.Hash{ev.ID, common.BigToHash(big.NewInt(dexType)), testDexID, common.HexToHash("0x01")},
		Data:    data,
	}
}

func TestD4PositionEventsDriveTicks(t *testing.T) {
	tr := &PoolTracker{}
	for name, sign := range map[string]int64{"LogDeposit": 1, "LogBorrow": 1, "LogWithdraw": -1, "LogPayback": -1} {
		lower, upper, delta, err := tr.extractEventData(positionLog(t, name, 4, -10, 20, 500))
		require.NoError(t, err, name)
		require.Equal(t, int64(-10), int64(lower), name)
		require.Equal(t, int64(20), int64(upper), name)
		require.Equal(t, sign*500, delta.Int64(), name)
	}
}

func TestPoolAddressFromEveryPoolEvent(t *testing.T) {
	f := NewPoolFactory(&Config{DexID: "fluid-dex-v2", ChainID: 1})
	want := []string{encodeFluidDexV2PoolAddress(testDexID.Hex(), 4)}
	for _, name := range []string{"LogDeposit", "LogWithdraw", "LogBorrow", "LogPayback"} {
		pools, err := f.DecodePoolAddressesFromFactoryLog(t.Context(), positionLog(t, name, 4, -10, 20, 500))
		require.NoError(t, err, name)
		require.Equal(t, want, pools, name)
	}
	for _, name := range []string{"LogSwapIn", "LogSwapOut"} {
		ev := abis.DexV2ABI.Events[name]
		data, err := ev.Inputs.Pack(big.NewInt(4), [32]byte(testDexID), common.Address{1}, true,
			big.NewInt(1), big.NewInt(1), big.NewInt(0), big.NewInt(0))
		require.NoError(t, err, name)
		pools, err := f.DecodePoolAddressesFromFactoryLog(t.Context(), ethtypes.Log{
			Address: common.HexToAddress("0xaBE4D5AA1Bb188f4B5d3a1aE3B8ec5e300A78261"),
			Topics:  []common.Hash{ev.ID}, Data: data})
		require.NoError(t, err, name)
		require.Equal(t, want, pools, name)
	}
}
