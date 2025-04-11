package math

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExamples(t *testing.T) {
	require.Zero(t, ToSqrtRatio(1_000_000).Cmp(IntFromString("561030636129153856579134353873645338624")))
	require.Zero(t, ToSqrtRatio(10_000_000).Cmp(IntFromString("50502254805927926084423855178401471004672")))
	require.Zero(t, ToSqrtRatio(-1_000_000).Cmp(IntFromString("206391740095027370700312310528859963392")))
	require.Zero(t, ToSqrtRatio(-10_000_000).Cmp(IntFromString("2292810285051363400276741630355046400")))
}

func TestMinTick(t *testing.T) {
	require.Zero(t, ToSqrtRatio(MinTick).Cmp(MinSqrtRatio))
}

func TestMaxTick(t *testing.T) {
	require.Zero(t, ToSqrtRatio(MaxTick).Cmp(MaxSqrtRatio))
}
