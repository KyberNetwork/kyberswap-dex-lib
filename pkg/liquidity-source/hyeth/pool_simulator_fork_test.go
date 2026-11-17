package hyeth

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The hyETH FlashMint pool on Ethereum at block 26375896 (state 0), on a fork of that block after
// issueExactSetFromETH(hyETH, 755204291037922288) with 1 ETH sent (state 1, block 26375897, 0.999983901439842020
// ETH spent), and after redeemExactSetForETH(hyETH, 755204291037922288, 0) (state 2, block 26375899, 0.999983911510952318
// ETH received). Each state is what the tracker read on the fork.
const (
	forkState0 = `{"feeI":"0","feeR":"0","comp":"0x701907283a57ff77e255c3f1aad790466b8ce4ef","compSup":"617527847753778552754","compAss":"651345980688267815311","compHyb":"157373215919122576119","hySup":"125357539445702562068","dpru":"1255394901774434537","epru":[],"isDisabled":false,"maxDeposit":"1000000019348654019311732184688","maxRedeem":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}`
	forkState1 = `{"feeI":"0","feeR":"0","comp":"0x701907283a57ff77e255c3f1aad790466b8ce4ef","compSup":"618485889582740232959","compAss":"652345986468261489129","compHyb":"158321295535889760043","hySup":"126112743736740484356","dpru":"1255394901774434537","epru":[],"isDisabled":false,"maxDeposit":"1000000019347654013531738510870","maxRedeem":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}`
	forkState2 = `{"feeI":"0","feeR":"0","comp":"0x701907283a57ff77e255c3f1aad790466b8ce4ef","compSup":"617537810658077476608","compAss":"651346009856699617222","compHyb":"157373215919122576120","hySup":"125357539445702562068","dpru":"1255394901774434537","epru":[],"isDisabled":false,"maxDeposit":"1000000019348653990143300382777","maxRedeem":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}`
)

func forkPool(t *testing.T, extra string) *PoolSimulator {
	t.Helper()
	p, err := NewPoolSimulator(entity.Pool{Address: "0xcb1eea349f25288627f008c5e2a69b684bdddf49", Exchange: "hyeth",
		Type: DexType, Tokens: []*entity.PoolToken{{Address: testHyeth}, {Address: testWeth}},
		Reserves: entity.PoolReserves{"0", "0"}, Extra: extra})
	require.NoError(t, err)
	return p
}

// TestUpdateBalanceFollowsTheChain issues and redeems on state 0 and 1 and compares the simulator after
// UpdateBalance with the tracker's read of the next state. hyETH's balance of vault shares and its supply match to
// the wei. The vault's shares used to move by the hyETH amount instead of the vault shares (1.255 a hyETH), which
// put the next redeem quote 3.3 bps off. What is left after the issue is the vault's performance fee: its deposit
// mints fee shares that totalSupply() did not count before, 0.16 bps of the next quote.
func TestUpdateBalanceFollowsTheChain(t *testing.T) {
	for _, tc := range []struct {
		name, from, to, in, out, amount, want string
		maxNextBps                            float64
	}{
		{"issue", forkState0, forkState1, testWeth, testHyeth, "1000000000000000000", "755204291037922288", 0.2},
		{"redeem", forkState1, forkState2, testHyeth, testWeth, "755204291037922288", "999983901439841985", 0.01},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := forkPool(t, tc.from)
			s := base.CloneState().(*PoolSimulator)
			amt, _ := new(big.Int).SetString(tc.amount, 10)
			res, err := quote(s, tc.in, tc.out, amt)
			require.NoError(t, err)
			require.Equal(t, tc.want, res.TokenAmountOut.Amount.String())
			s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})

			var want Extra
			require.NoError(t, json.Unmarshal([]byte(tc.to), &want))
			require.Equal(t, want.ComponentHyethBalance, s.componentHyethBalance)
			require.Equal(t, want.HyethTotalSupply, s.hyethTotalSupply)

			next, err := quote(s, testHyeth, testWeth, big.NewInt(1e18))
			require.NoError(t, err)
			ref, err := quote(forkPool(t, tc.to), testHyeth, testWeth, big.NewInt(1e18))
			require.NoError(t, err)
			diff := new(big.Float).SetInt(new(big.Int).Sub(next.TokenAmountOut.Amount, ref.TokenAmountOut.Amount))
			bps, _ := diff.Quo(diff, new(big.Float).SetInt(ref.TokenAmountOut.Amount)).Float64()
			require.InDelta(t, 0, bps*1e4, tc.maxNextBps, "next redeem %s, chain %s", next.TokenAmountOut.Amount,
				ref.TokenAmountOut.Amount)

			// The clone's update leaves the simulator it was cloned from as it was.
			again, err := quote(base, tc.in, tc.out, amt)
			require.NoError(t, err)
			require.Equal(t, tc.want, again.TokenAmountOut.Amount.String())
		})
	}
}
