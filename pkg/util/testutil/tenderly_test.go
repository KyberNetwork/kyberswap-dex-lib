package testutil

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCalcMappingSlot(t *testing.T) {
	// Solidity: keccak256(abi.encode(key, slot))
	slot := calcMappingSlot(big.NewInt(3), DefaultSender, false)
	require.NotEmpty(t, slot)
	require.Equal(t, 66, len(slot)) // "0x" + 64 hex chars

	// Vyper: keccak256(abi.encode(slot, key)) — different result
	vyperSlot := calcMappingSlot(big.NewInt(3), DefaultSender, true)
	require.NotEmpty(t, vyperSlot)
	require.NotEqual(t, slot, vyperSlot)
}

func TestCalcNestedMappingSlot(t *testing.T) {
	slot := calcNestedMappingSlot(big.NewInt(1), DefaultSender, DefaultSpender, false)
	require.NotEmpty(t, slot)
	require.Equal(t, 66, len(slot))

	vyperSlot := calcNestedMappingSlot(big.NewInt(1), DefaultSender, DefaultSpender, true)
	require.NotEmpty(t, vyperSlot)
	require.NotEqual(t, slot, vyperSlot)
}

func TestEncodeSwapCalldata(t *testing.T) {
	amount := big.NewInt(1000000)
	calldata, err := EncodeSwapCalldata("swap(uint256,bool)", amount, true)
	require.NoError(t, err)
	require.Equal(t, 4+32+32, len(calldata)) // 4 byte selector + 2 args × 32 bytes
}

func TestEncodeSwapCalldataAddress(t *testing.T) {
	amount := big.NewInt(1000000)
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	calldata, err := EncodeSwapCalldata("swap(address,uint256)", addr, amount)
	require.NoError(t, err)
	require.Equal(t, 4+32+32, len(calldata))
}

func TestWithTokenBalance(t *testing.T) {
	cfg := defaultSimulateConfig()
	token := common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2") // WETH
	amount := new(big.Int).Mul(big.NewInt(100), big.NewInt(1e18))

	opt := WithTokenBalance(token, DefaultSender, 3, amount)
	opt(cfg)

	tokenAddr := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	require.NotNil(t, cfg.stateOverrides[tokenAddr])
	require.Len(t, cfg.stateOverrides[tokenAddr].Storage, 1)
}

func TestMaxUint256(t *testing.T) {
	v := MaxUint256()
	require.Equal(t, 256, v.BitLen())
}

// --- Slot finder tests (require Tenderly API) ---

func TestFindBalanceOfSlot_WETH(t *testing.T) {
	tc := RequireTenderly(t)
	weth := common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")

	slots, err := tc.FindBalanceOfSlot(1, weth)
	require.NoError(t, err)
	require.False(t, slots.IsVyper)
	require.Empty(t, slots.StateProxy)

	// WETH balanceOf slot = 3
	require.Equal(t, big.NewInt(3), slots.BalanceSlot)
	t.Logf("WETH balanceOf slot: %d, isVyper: %v, proxy: %q", slots.BalanceSlot, slots.IsVyper, slots.StateProxy)
}

func TestFindBalanceOfSlot_AgoraAUSD(t *testing.T) {
	tc := RequireTenderly(t)
	ausd := common.HexToAddress("0x00000000efe302beaa2b3e6e1b18d08d69a9012a")

	slots, err := tc.FindBalanceOfSlot(43114, ausd)
	require.NoError(t, err)
	require.False(t, slots.IsVyper)

	// AgoraDollar uses ERC7201 namespace "AgoraDollarErc1967Proxy.Erc20CoreStorage"
	// Base slot = 0x455730fed596673e69db1907be2e521374ba893f1a04cc5f5dd931616cd6b700
	// accountData mapping is at offset 0
	expectedSlot, _ := new(big.Int).SetString("455730fed596673e69db1907be2e521374ba893f1a04cc5f5dd931616cd6b700", 16)
	require.Equal(t, expectedSlot, slots.BalanceSlot)
	t.Logf("AUSD balanceOf slot: 0x%x, isVyper: %v, proxy: %q", slots.BalanceSlot, slots.IsVyper, slots.StateProxy)
}

func TestFindAllowanceSlot_WETH(t *testing.T) {
	tc := RequireTenderly(t)
	weth := common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")

	slots, err := tc.FindAllowanceSlot(1, weth)
	require.NoError(t, err)
	require.False(t, slots.IsVyper)

	// WETH allowance slot = 4
	require.Equal(t, big.NewInt(4), slots.AllowSlot)
	t.Logf("WETH allowance slot: %d, isVyper: %v, proxy: %q", slots.AllowSlot, slots.IsVyper, slots.StateProxy)
}

func TestFindTokenSlots_USDC(t *testing.T) {
	tc := RequireTenderly(t)
	usdc := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")

	slots, err := tc.FindTokenSlots(1, usdc)
	require.NoError(t, err)

	// USDC balanceOf slot = 9, allowance slot = 10
	require.Equal(t, big.NewInt(9), slots.BalanceSlot)
	require.Equal(t, big.NewInt(10), slots.AllowSlot)
	t.Logf("USDC balance: %d, allowance: %d, isVyper: %v, proxy: %q",
		slots.BalanceSlot, slots.AllowSlot, slots.IsVyper, slots.StateProxy)
}

func TestFindTokenSlots_DAI(t *testing.T) {
	tc := RequireTenderly(t)
	dai := common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F")

	slots, err := tc.FindTokenSlots(1, dai)
	require.NoError(t, err)

	// DAI balanceOf slot = 2, allowance slot = 3
	require.Equal(t, big.NewInt(2), slots.BalanceSlot)
	require.Equal(t, big.NewInt(3), slots.AllowSlot)
	t.Logf("DAI balance: %d, allowance: %d, isVyper: %v, proxy: %q",
		slots.BalanceSlot, slots.AllowSlot, slots.IsVyper, slots.StateProxy)
}

func TestFindTokenSlots_UpgradeableProxy(t *testing.T) {
	tc := RequireTenderly(t)
	myToken := common.HexToAddress("0x61e24ce4efe61eb2efd6ac804445df65f8032955")

	slots, err := tc.FindBalanceOfSlot(1, myToken)
	require.NoError(t, err)

	t.Logf("MyToken balance slot: %d, isVyper: %v, proxy: %q",
		slots.BalanceSlot, slots.IsVyper, slots.StateProxy)
	require.True(t, slots.BalanceSlot.Cmp(erc7201BaseSlot) >= 0,
		"expected slot >= ERC7201 base, got %s", slots.BalanceSlot)
}
