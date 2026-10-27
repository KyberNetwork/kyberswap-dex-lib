package integral

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestCalcAmountMatchesChain: each fixture holds a pool's state at one block, its plugin's sliding fee
// data as the tracker reads it, and what the pool's own swap gave at that block, by amount in and by
// amount out, both ways. The pools are Camelot's, Cypher's and Sheriff's, whose plugins scale the
// dynamic fee by a fee factor, and a Thena pool whose plugin does not. The simulator must quote each
// amount to the wei.
func TestCalcAmountMatchesChain(t *testing.T) {
	files, err := filepath.Glob("testdata/onchain/*.json")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	oldBlockTimestamp := blockTimestamp
	t.Cleanup(func() { blockTimestamp = oldBlockTimestamp })
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			b, err := os.ReadFile(file)
			require.NoError(t, err)
			var fixture struct {
				BlockTimestamp uint32      `json:"blockTimestamp"`
				Pool           entity.Pool `json:"pool"`
				Quotes         []struct {
					Kind      string `json:"kind"`
					Amount    string `json:"amount"`
					ZeroToOne bool   `json:"zeroToOne"`
					Result    string `json:"result"`
				} `json:"quotes"`
			}
			require.NoError(t, json.Unmarshal(b, &fixture))
			blockTimestamp = func() uint32 { return fixture.BlockTimestamp }

			token0, token1 := fixture.Pool.Tokens[0].Address, fixture.Pool.Tokens[1].Address
			for _, q := range fixture.Quotes {
				in, out := token0, token1
				if !q.ZeroToOne {
					in, out = token1, token0
				}
				amount, _ := new(big.Int).SetString(q.Amount, 10)
				msg := []any{"%s %s, zeroToOne %v", q.Amount, q.Kind, q.ZeroToOne}
				sim, err := NewPoolSimulator(fixture.Pool) // a quote writes the block's timepoint once
				require.NoError(t, err)
				var got *big.Int
				if q.Kind == "in" {
					res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount}, TokenOut: out})
					if !assert.NoError(t, err, msg...) {
						continue
					}
					got = res.TokenAmountOut.Amount
				} else {
					res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
						TokenAmountOut: pool.TokenAmount{Token: out, Amount: amount}, TokenIn: in})
					if !assert.NoError(t, err, msg...) {
						continue
					}
					got = res.TokenAmountIn.Amount
				}
				assert.Equal(t, q.Result, got.String(), msg...)
			}
		})
	}
}
