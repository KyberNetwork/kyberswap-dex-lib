package pool

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

type limitSnapshot struct {
	IPoolSimulator
	limits map[string]*big.Int
}

func (s limitSnapshot) CalculateLimit() map[string]*big.Int { return s.limits }

type policyLimitSnapshot struct {
	limitSnapshot
	policies map[string]SwapLimitMergePolicy
}

func (s policyLimitSnapshot) GetLimitMergePolicy(key string) SwapLimitMergePolicy {
	return s.policies[key]
}

func TestMergeSwapLimitsDefaultMax(t *testing.T) {
	first := limitSnapshot{limits: map[string]*big.Int{"opaque": big.NewInt(80)}}
	second := limitSnapshot{limits: map[string]*big.Int{"opaque": big.NewInt(100), "new": big.NewInt(0)}}
	for _, snapshots := range [][]IPoolSimulator{{first, second}, {second, first}} {
		limits := make(map[string]*big.Int)
		for _, snapshot := range snapshots {
			MergeSwapLimits(limits, snapshot)
		}
		require.Equal(t, int64(100), limits["opaque"].Int64())
		require.Zero(t, limits["new"].Sign())
	}
	require.Equal(t, int64(80), first.limits["opaque"].Int64(), "snapshots remain read-only")
}

func TestMergeSwapLimitsPerKeyPolicy(t *testing.T) {
	policies := map[string]SwapLimitMergePolicy{"a": SwapLimitMergeMin, "b": SwapLimitMergeMax}
	first := policyLimitSnapshot{limitSnapshot{limits: map[string]*big.Int{"a": big.NewInt(100), "b": big.NewInt(7), "default": big.NewInt(3)}}, policies}
	second := policyLimitSnapshot{limitSnapshot{limits: map[string]*big.Int{"a": big.NewInt(80), "b": big.NewInt(9), "default": big.NewInt(4)}}, policies}
	for _, snapshots := range [][]IPoolSimulator{{first, second}, {second, first}} {
		limits := make(map[string]*big.Int)
		for _, snapshot := range snapshots {
			MergeSwapLimits(limits, snapshot)
		}
		require.Equal(t, int64(80), limits["a"].Int64())
		require.Equal(t, int64(9), limits["b"].Int64())
		require.Equal(t, int64(4), limits["default"].Int64())
		MergeSwapLimits(limits, policyLimitSnapshot{limitSnapshot{limits: map[string]*big.Int{"a": big.NewInt(0)}}, policies})
		require.Zero(t, limits["a"].Sign(), "a closed capacity must remain closed")
	}
	require.Equal(t, int64(100), first.limits["a"].Int64())
	require.Equal(t, int64(80), second.limits["a"].Int64())
}
