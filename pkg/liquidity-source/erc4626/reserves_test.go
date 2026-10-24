package erc4626

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func bigStr(s string) *big.Int { v, _ := new(big.Int).SetString(s, 10); return v }

// Reserves are [shares, assets], each in its own token's units. The price graph skips a hop when the
// output token's reserve is below the output amount, so a wrong unit hides a direction.
func TestUpdateEntityStateReservesUnits(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		state PoolState
		want  entity.PoolReserves
	}{
		{ // nvPRIME-like: 6-dec asset, 18-dec share, unbounded limits
			name:  "unbounded, 1e12 decimals gap",
			state: PoolState{TotalAssets: bigStr("6200000000000"), TotalSupply: bigStr("6200000000000000000000000")},
			want:  entity.PoolReserves{"6200000000000000000000000", "6200000000000"},
		},
		{ // bounded maxDeposit: 1e12 assets -> 1e24 shares; redeem unbounded -> totalAssets
			name: "bounded maxDeposit converts",
			state: PoolState{TotalAssets: bigStr("6200000000000"), TotalSupply: bigStr("6200000000000000000000000"),
				MaxDeposit: bigStr("1000000000000")},
			want: entity.PoolReserves{"1000000000000000000000000", "6200000000000"},
		},
		{ // bounded maxRedeem: 1e24 shares -> 1e12 assets; deposit unbounded -> totalSupply
			name: "bounded maxRedeem converts",
			state: PoolState{TotalAssets: bigStr("6200000000000"), TotalSupply: bigStr("6200000000000000000000000"),
				MaxRedeem: bigStr("1000000000000000000000000")},
			want: entity.PoolReserves{"6200000000000000000000000", "1000000000000"},
		},
		{ // sUSDe-like: redeem blocked (maxRedeem 0) must stay "0", deposit side stays usable
			name: "maxRedeem zero",
			state: PoolState{TotalAssets: bigStr("1200000000000000000000000000"), TotalSupply: bigStr("1000000000000000000000000000"),
				MaxRedeem: big.NewInt(0)},
			want: entity.PoolReserves{"1000000000000000000000000000", "0"},
		},
		{ // deposit-only vault: no totalSupply read, no redeem reserve
			name:  "deposit only",
			state: PoolState{TotalAssets: bigStr("1200000000000000000000000000")},
			want:  entity.PoolReserves{"1200000000000000000000000000", "0"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p entity.Pool
			require.NoError(t, UpdateEntityState(&p, VaultCfg{Gas: GasCfg{Deposit: 1, Redeem: 1}}, &tc.state))
			assert.Equal(t, tc.want, p.Reserves)
		})
	}
}
