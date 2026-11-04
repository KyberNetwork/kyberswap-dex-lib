package nami

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNamiFee(t *testing.T) {
	cfg := Config{Floor: 100, Cap: 1000, Premium0For1: 100, Premium1For0: 200, MaxDiscountBps: 5000, Fee0For1: 900, Fee1For0: 100, ExpiresAt: 1000, DecayWindow: 100}
	for _, tc := range []struct {
		name string
		base uint16
		dir  bool
		now  uint64
		want uint16
	}{
		{"active high", 300, true, 999, 900}, {"active low limited discount", 300, false, 999, 250},
		{"decay high", 300, true, 1050, 650}, {"decay low", 300, false, 1050, 375},
		{"decay ended", 300, true, 1100, 400}, {"cap", 1000, true, 1100, 1000}, {"floor", 0, true, 1100, 100},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.want, cfg.Fee(tc.base, tc.dir, tc.now)) })
	}
	cfg.Cap = 0
	require.Equal(t, uint16(300), cfg.Fee(300, true, 999))
}

// Base METAc/USDC pool 0x6dbf1772aee6f453c6dafe705bb388685d5df1ea:
// plugin swapFee(true/false) returned 1000 with a flat feeConfig and unset bounds.
func TestNamiObservedBaseFee(t *testing.T) {
	cfg := Config{}
	require.Equal(t, uint16(1000), cfg.Fee(1000, true, 0))
	require.Equal(t, uint16(1000), cfg.Fee(1000, false, 0))
}

func TestNamiABIOutputs(t *testing.T) {
	bounds := namiFeeBounds{Floor: 100, Cap: 3000, Premium0For1: 12, Premium1For0: 45, DecayWindow: 30, MaxDiscountBps: 500}
	method := namiPluginABI.Methods["feeBounds"]
	encoded, err := method.Outputs.Pack(bounds)
	require.NoError(t, err)
	var decoded struct{ Bounds namiFeeBounds }
	require.NoError(t, namiPluginABI.UnpackIntoInterface(&decoded, "feeBounds", encoded))
	require.Equal(t, bounds, decoded.Bounds)
	original := namiFeeOverride{Fee0For1: 120, Fee1For0: 230, ExpiresAt: big.NewInt(123456789)}
	encoded, err = namiPluginABI.Methods["feeOverride"].Outputs.Pack(original)
	require.NoError(t, err)
	var override struct{ OverrideState namiFeeOverride }
	require.NoError(t, namiPluginABI.UnpackIntoInterface(&override, "feeOverride", encoded))
	require.Equal(t, original, override.OverrideState)
}

func TestMevxGasConfig(t *testing.T) {
	u := func(v uint64) *big.Int { return new(big.Int).SetUint64(v) }
	for _, tc := range []struct {
		name               string
		enabled            bool
		minGasLeft, budget *big.Int
		gas                uint64
		unsupported        string
		wantErr            bool
	}{
		{"router disabled", false, u(0), u(5_000_000), 0, "", false},
		{"deployed config", true, u(0), u(5_000_000), MEVXGasEstimate, "", false},
		{"minGasLeft at limit", true, u(MaxMinGasLeft), u(5_000_000), MEVXGasEstimate, "", false},
		{"minGasLeft above limit", true, u(MaxMinGasLeft + 1), u(5_000_000), 0, "Nami MEV-X minGasLeft is too high", false},
		{"minGasLeft above contract max", true, u(5_000_001), u(5_000_000), 0, "", true},
		{"budget above contract max", true, u(0), u(5_000_001), 0, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gas, unsupported, err := mevxGasConfig(tc.enabled, tc.minGasLeft, tc.budget)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.gas, gas)
			require.Equal(t, tc.unsupported, unsupported)
		})
	}
}

func TestHookGasAddsMinGasLeft(t *testing.T) {
	require.Zero(t, Config{}.HookGas(), "MEV-X disabled")
	require.Equal(t, MEVXGasEstimate, Config{MEVXGas: 5_000_000}.HookGas(), "old inflated snapshot")
	require.Equal(t, MEVXGasEstimate+300_000, Config{MEVXGas: MEVXGasEstimate, MinGasLeft: 300_000}.HookGas())
}
