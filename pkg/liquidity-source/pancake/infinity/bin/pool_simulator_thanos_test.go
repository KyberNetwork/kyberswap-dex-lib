package bin

import (
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// On-chain quotes at BSC block 127938800 for SquadSwap Thanos pool 0x999e8f48... (XRP/WBNB),
// from the BinQuoter code run against manager 0x7f8e5a02... with the hook's swapFees(poolId)
// storage overridden. Columns: swapFees(pips) zeroForOne kind amount quote ("Error:" = revert).
const thanosQuotes = `
0 true in 1000000000000000000 1830268254350661
0 true in 12345678901234567 22611159220099
0 true in 2000000000000000000000 3645006758041892895
0 true out 1000000000000000000 546436694115809266679
0 true out 12345678901234567 6745332337366082051
0 true out 2000000000000000000000 Error:
0 false in 1000000000000000000 543856910704331969216
0 false in 12345678901234567 6715081315636153773
0 false in 2000000000000000000000 Error:
0 false out 1000000000000000000 1838500283308186
0 false out 12345678901234567 22697534157553
0 false out 2000000000000000000000 3692907373778271959
777 true in 1000000000000000000 1828846147918368
777 true in 12345678901234567 22593602350723
777 true in 2000000000000000000000 3642187990270089860
777 true out 1000000000000000000 546862499226629041949
777 true out 12345678901234567 6750573467149417449
777 true out 2000000000000000000000 Error:
777 false in 1000000000000000000 543435547716728941965
777 false in 12345678901234567 6709863697453904245
777 false in 2000000000000000000000 Error:
777 false out 1000000000000000000 1839928798028317
777 false out 12345678901234567 22715170141593
777 false out 2000000000000000000000 3695790193700429909
10000 true in 1000000000000000000 1811965726264523
10000 true in 12345678901234567 22385202085267
10000 true in 2000000000000000000000 3608729180541210339
10000 true out 1000000000000000000 551916811372177030770
10000 true out 12345678901234567 6812785745131017843
10000 true out 2000000000000000000000 Error:
10000 false in 1000000000000000000 538433963630933935100
10000 false in 12345678901234567 6647930502479792459
10000 false in 2000000000000000000000 Error:
10000 false out 1000000000000000000 1856885286141267
10000 false out 12345678901234567 22924509499128
10000 false out 2000000000000000000000 3730009303278374268
`

func TestThanosHookSwapFee(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("testdata/thanos_pool.json")
	require.NoError(t, err)
	var ent entity.Pool
	require.NoError(t, json.Unmarshal(data, &ent))
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(ent.Extra), &extra))
	require.Zero(t, extra.HookSwapFee) // swapFees was 0 on-chain

	for _, line := range strings.Split(strings.TrimSpace(thanosQuotes), "\n") {
		f := strings.Fields(line)
		sf, _ := new(big.Int).SetString(f[0], 10)
		extra.HookSwapFee = uint32(sf.Uint64())
		e := ent
		eb, _ := json.Marshal(extra)
		e.Extra = string(eb)
		sim, err := NewPoolSimulator(e, valueobject.ChainIDBSC)
		require.NoError(t, err)

		in, out := e.Tokens[0].Address, e.Tokens[1].Address
		if f[1] == "false" {
			in, out = out, in
		}
		amt, _ := new(big.Int).SetString(f[3], 10)
		var got *big.Int
		if f[2] == "in" {
			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt}, TokenOut: out})
			if err == nil {
				got = res.TokenAmountOut.Amount
			}
		} else {
			res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: out, Amount: amt}, TokenIn: in})
			if err == nil {
				got = res.TokenAmountIn.Amount
			}
		}
		if f[4] == "Error:" {
			require.Nil(t, got, line)
			continue
		}
		require.NotNil(t, got, line)
		require.Equal(t, f[4], got.String(), line)
	}
}

func TestThanosHookSwapFeeFull(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("testdata/thanos_pool.json")
	require.NoError(t, err)
	var ent entity.Pool
	require.NoError(t, json.Unmarshal(data, &ent))
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(ent.Extra), &extra))
	extra.HookSwapFee = 1_000_000 // owner can set up to 100%: nothing left to swap
	eb, _ := json.Marshal(extra)
	ent.Extra = string(eb)
	sim, err := NewPoolSimulator(ent, valueobject.ChainIDBSC)
	require.NoError(t, err)
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: ent.Tokens[0].Address, Amount: big.NewInt(1e18)},
		TokenOut:      ent.Tokens[1].Address})
	require.ErrorIs(t, err, ErrInsufficientAmountUnSpecified)
}
