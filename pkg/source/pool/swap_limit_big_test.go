package pool

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// fakeLimit is a minimal uint256 SwapLimit. Its balances are the state the adapter must reflect.
type fakeLimit struct {
	balance map[string]*uint256.Int
}

func (f *fakeLimit) Clone() SwapLimit                    { return f }
func (f *fakeLimit) GetExchange() string                 { return "x" }
func (f *fakeLimit) GetSwapped() map[string]*uint256.Int { return nil }
func (f *fakeLimit) GetAllowedSenders() string           { return "" }
func (f *fakeLimit) GetLimit(k string) *uint256.Int {
	return f.balance[k]
}

// UpdateLimit subtracts dd. A nil dd is ignored, as SingleSwapLimit ignores its deltas.
func (f *fakeLimit) UpdateLimit(dec, inc string, dd, id *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	if dd == nil {
		return f.balance[dec], uint256.NewInt(0), nil
	}
	f.balance[dec] = new(uint256.Int).Sub(f.balance[dec], dd)
	return f.balance[dec], uint256.NewInt(0), nil
}

func TestAsSwapLimitBigMatchesUint256Limit(t *testing.T) {
	inner := &fakeLimit{balance: map[string]*uint256.Int{"a": uint256.NewInt(100)}}
	legacy := AsSwapLimitBig(inner)

	require.Equal(t, big.NewInt(100), legacy.GetLimit("a"), "legacy read must equal the uint256 balance")
	require.Nil(t, legacy.GetLimit("missing"), "absent key stays absent")

	after, _, err := legacy.UpdateLimit("a", "b", big.NewInt(30), big.NewInt(0))
	require.NoError(t, err)
	require.Equal(t, big.NewInt(70), after, "update must go through the uint256 limit")
	require.Equal(t, big.NewInt(70), inner.balance["a"].ToBig(), "uint256 state must change")
}

func TestAsSwapLimitBigRejectsNegativeAndOversized(t *testing.T) {
	legacy := AsSwapLimitBig(&fakeLimit{balance: map[string]*uint256.Int{"a": uint256.NewInt(100)}})

	_, _, err := legacy.UpdateLimit("a", "b", big.NewInt(-1), big.NewInt(0))
	require.ErrorIs(t, err, ErrNotEnoughInventory, "negative delta must not wrap into a huge uint256")

	tooBig := new(big.Int).Lsh(big.NewInt(1), 256)
	_, _, err = legacy.UpdateLimit("a", "b", tooBig, big.NewInt(0))
	require.ErrorIs(t, err, ErrNotEnoughInventory, "a delta over 256 bits must be rejected")
}

// A nil delta is an unset amount. The adapter forwards it, so the inner limit decides. It must not panic here.
func TestAsSwapLimitBigForwardsNilDelta(t *testing.T) {
	inner := &fakeLimit{balance: map[string]*uint256.Int{"a": uint256.NewInt(100)}}
	legacy := AsSwapLimitBig(inner)
	require.NotPanics(t, func() {
		_, _, err := legacy.UpdateLimit("a", "b", nil, big.NewInt(0))
		require.NoError(t, err)
	})
}

func TestAsSwapLimitBigNilStaysNil(t *testing.T) {
	require.Nil(t, AsSwapLimitBig(nil))
}
