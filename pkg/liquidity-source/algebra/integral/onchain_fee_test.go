package integral

import (
	"encoding/json"
	"fmt"
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
//
// The Camelot pool 0x7529fe05 on Arbitrum, at block 518746249, is the case of a price that fell:
// its tick is 265624 and its last timepoint's is 265632, so a swap at that block moves the fee
// factors by a negative price change, which the plugin divides as an int256. Each of its quotes is
// the pool's own swap in an eth_call at that block.
func TestCalcAmountMatchesChain(t *testing.T) {
	for name, fixture := range loadOnchainFeeFixtures(t) {
		t.Run(name, func(t *testing.T) {
			setBlockTimestamp(t, fixture.BlockTimestamp)
			for _, q := range fixture.Quotes {
				sim, err := NewPoolSimulator(fixture.Pool) // a quote writes the block's timepoint once
				require.NoError(t, err)
				got, err := fixture.quote(sim, q)
				if assert.NoError(t, err, q) {
					assert.Equal(t, q.Result, got, q)
				}
			}
		})
	}
}

type onchainFeeQuote struct {
	Kind      string `json:"kind"` // "in": the output of an amount in; "out": the input of an amount out
	Amount    string `json:"amount"`
	ZeroToOne bool   `json:"zeroToOne"`
	Result    string `json:"result"`
}

func (q onchainFeeQuote) String() string {
	return fmt.Sprintf("%s %s, zeroToOne %v", q.Amount, q.Kind, q.ZeroToOne)
}

type onchainFeeFixture struct {
	BlockTimestamp uint32            `json:"blockTimestamp"`
	Pool           entity.Pool       `json:"pool"`
	Quotes         []onchainFeeQuote `json:"quotes"`
}

func loadOnchainFeeFixtures(t *testing.T) map[string]onchainFeeFixture {
	files, err := filepath.Glob("testdata/onchain/*.json")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	fixtures := make(map[string]onchainFeeFixture, len(files))
	for _, file := range files {
		b, err := os.ReadFile(file)
		require.NoError(t, err)
		var fixture onchainFeeFixture
		require.NoError(t, json.Unmarshal(b, &fixture), file)
		fixtures[filepath.Base(file)] = fixture
	}
	return fixtures
}

// setBlockTimestamp makes the simulators take ts as the time of a swap until the test ends.
func setBlockTimestamp(t *testing.T, ts uint32) {
	old := blockTimestamp
	t.Cleanup(func() { blockTimestamp = old })
	blockTimestamp = func() uint32 { return ts }
}

// quote returns what sim quotes for q: the output of an amount in, or the input of an amount out.
func (f onchainFeeFixture) quote(sim *PoolSimulator, q onchainFeeQuote) (string, error) {
	in, out := f.Pool.Tokens[0].Address, f.Pool.Tokens[1].Address
	if !q.ZeroToOne {
		in, out = out, in
	}
	amount, _ := new(big.Int).SetString(q.Amount, 10)
	if q.Kind == "in" {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount}, TokenOut: out})
		if err != nil {
			return "", err
		}
		return res.TokenAmountOut.Amount.String(), nil
	}
	res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: out, Amount: amount}, TokenIn: in})
	if err != nil {
		return "", err
	}
	return res.TokenAmountIn.Amount.String(), nil
}
