package pool

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type limitSnapshot struct {
	IPoolSim
	limits map[string]*uint256.Int
}

func (s limitSnapshot) CalcLimit() map[string]*uint256.Int { return s.limits }

type policyLimitSnapshot struct {
	limitSnapshot
	policies map[string]SwapLimitMergePolicy
}

func (s policyLimitSnapshot) GetLimitMergePolicy(key string) SwapLimitMergePolicy {
	return s.policies[key]
}

func TestMergeSwapLimitsDefaultMax(t *testing.T) {
	first := limitSnapshot{limits: map[string]*uint256.Int{"opaque": uint256.NewInt(80)}}
	second := limitSnapshot{limits: map[string]*uint256.Int{"opaque": uint256.NewInt(100), "new": uint256.NewInt(0)}}
	for _, snapshots := range [][]IPoolSim{{first, second}, {second, first}} {
		limits := make(map[string]*uint256.Int)
		for _, snapshot := range snapshots {
			MergeSwapLimits(limits, snapshot)
		}
		require.Equal(t, uint64(100), limits["opaque"].Uint64())
		require.Zero(t, limits["new"].Sign())
	}
	require.Equal(t, uint64(80), first.limits["opaque"].Uint64(), "snapshots remain read-only")
}

func TestMergeSwapLimitsPerKeyPolicy(t *testing.T) {
	policies := map[string]SwapLimitMergePolicy{"a": SwapLimitMergeMin, "b": SwapLimitMergeMax}
	first := policyLimitSnapshot{limitSnapshot{limits: map[string]*uint256.Int{"a": uint256.NewInt(100), "b": uint256.NewInt(7), "default": uint256.NewInt(3)}}, policies}
	second := policyLimitSnapshot{limitSnapshot{limits: map[string]*uint256.Int{"a": uint256.NewInt(80), "b": uint256.NewInt(9), "default": uint256.NewInt(4)}}, policies}
	for _, snapshots := range [][]IPoolSim{{first, second}, {second, first}} {
		limits := make(map[string]*uint256.Int)
		for _, snapshot := range snapshots {
			MergeSwapLimits(limits, snapshot)
		}
		require.Equal(t, uint64(80), limits["a"].Uint64())
		require.Equal(t, uint64(9), limits["b"].Uint64())
		require.Equal(t, uint64(4), limits["default"].Uint64())
		MergeSwapLimits(limits, policyLimitSnapshot{limitSnapshot{limits: map[string]*uint256.Int{"a": uint256.NewInt(0)}}, policies})
		require.Zero(t, limits["a"].Sign(), "a closed capacity must remain closed")
	}
	require.Equal(t, uint64(100), first.limits["a"].Uint64())
	require.Equal(t, uint64(80), second.limits["a"].Uint64())
}
