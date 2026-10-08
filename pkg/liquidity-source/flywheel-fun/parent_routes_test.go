package flywheelfun

import (
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestNativeParentsSharedLiquidityAndClones(t *testing.T) {
	b, err := os.ReadFile("testdata/replacement-fork-markets.json")
	require.NoError(t, err)
	var rows []entity.Pool
	require.NoError(t, json.Unmarshal(b, &rows))
	tested := 0
	for _, p := range rows {
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
		if len(extra.Parents) == 0 || extra.Curve.Graduated {
			continue
		}
		a, err := NewPoolSimulator(p)
		require.NoError(t, err)
		bases := map[string]pool.IPoolSimulator{}
		for _, base := range a.GetBasePools() {
			bases[base.GetAddress()] = base
		}
		p.Address = "0x2222222222222222222222222222222222222222"
		p.Tokens = entity.ClonePoolTokens(p.Tokens)
		p.Tokens[1].Address = p.Address
		child, err := NewPoolSimulatorWithBases(p, bases)
		require.NoError(t, err)
		before := quoteBuy(t, child, 10000000000)
		cloned := child.CloneState().(*PoolSimulator)
		first := quoteBuy(t, a, 10000000000)
		a.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
		after := quoteBuy(t, child, 10000000000)
		require.NotEqual(t, [2]string{before.TokenAmountOut.Amount.String(), before.RemainingTokenAmountIn.Amount.String()}, [2]string{after.TokenAmountOut.Amount.String(), after.RemainingTokenAmountIn.Amount.String()}, "shared parent price impact must change received tokens or the capped-graduation refund")
		require.Equal(t, before.TokenAmountOut.Amount.String(), quoteBuy(t, cloned, 10000000000).TokenAmountOut.Amount.String())
		prior := make([]string, len(child.ParentPools))
		for i, base := range child.ParentPools {
			prior[i] = fingerprint(base)
		}
		child.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: before.SwapInfo})
		require.False(t, child.Valid)
		for i, base := range child.ParentPools {
			require.Equal(t, prior[i], fingerprint(base), "stale replay must not partially update any parent")
		}
		// The settlement takes abi.encode(bytes4("FWL1"), parentsNearestFirst, externalRoute) verbatim.
		unpacked, err := parentArguments.Unpack(first.SwapInfo.(SwapInfo).Route)
		require.NoError(t, err)
		require.Equal(t, [4]byte{'F', 'W', 'L', '1'}, unpacked[0])
		require.Len(t, unpacked[1], len(extra.Parents))
		tested++
	}
	require.Equal(t, 4, tested)
}

func TestNativeParentSnapshotValidation(t *testing.T) {
	var p entity.Pool
	for _, row := range snapshots(t) {
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(row.Extra), &extra))
		if len(extra.Parents) == 2 {
			p = row
			break
		}
	}
	require.NotEmpty(t, p.Address)
	for name, alter := range map[string]func(*Extra){
		"wrong order":          func(e *Extra) { e.Parents[0], e.Parents[1] = e.Parents[1], e.Parents[0] },
		"cycle":                func(e *Extra) { e.Parents[1].Quote = p.Address },
		"too deep":             func(e *Extra) { e.Parents = append(e.Parents, e.Parents[1]) },
		"mixed blocks":         func(e *Extra) { e.Parents[0].Pool.BlockNumber-- },
		"invalid protocol fee": func(e *Extra) { e.Parents[0].Protocol[0] = 1001 },
		"duplicate pool":       func(e *Extra) { e.Parents[1].Pool.Address = e.Parents[0].Pool.Address },
	} {
		t.Run(name, func(t *testing.T) {
			var extra Extra
			require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
			alter(&extra)
			encoded, err := json.Marshal(extra)
			require.NoError(t, err)
			bad := p
			bad.Extra = string(encoded)
			_, err = NewPoolSimulator(bad)
			require.Error(t, err)
		})
	}
	s := newSim(t, p)
	wrong := s.ParentPools[0].CloneState()
	c, err := core(wrong)
	require.NoError(t, err)
	c.Info.BlockNumber--
	s.SetBasePool(wrong)
	require.False(t, s.Valid, "do not relink liquidity from a different block")
}
