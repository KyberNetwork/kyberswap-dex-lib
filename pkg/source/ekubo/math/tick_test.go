package math

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExamples(t *testing.T) {
	require.Zero(t, ToSqrtRatio(1_000_000).Cmp(IntFromString("561030636129153856592777659729523183729")))
	require.Zero(t, ToSqrtRatio(10_000_000).Cmp(IntFromString("50502254805927926084427918474025309948677")))
	require.Zero(t, ToSqrtRatio(-1_000_000).Cmp(IntFromString("206391740095027370700312310531588921767")))
	require.Zero(t, ToSqrtRatio(-10_000_000).Cmp(IntFromString("2292810285051363400276741638672651165")))
}

func TestMinTick(t *testing.T) {
	require.Zero(t, ToSqrtRatio(MinTick).Cmp(MinSqrtRatio))
}

func TestMaxTick(t *testing.T) {
	require.Zero(t, ToSqrtRatio(MaxTick).Cmp(MaxSqrtRatio))
}
