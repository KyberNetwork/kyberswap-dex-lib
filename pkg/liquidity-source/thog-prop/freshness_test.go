package thogprop

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"math/big"
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestLiveAgingMatchesCurveAtEveryModeledBlock(t *testing.T) {
	s := stateFrom(t, fixture(t, "deployed-model3.json.gz").Snapshots[0])
	posted := s.small(2, 36, 32)
	s.BlockNumber = posted
	s.Words[2] = setField(s.Words[2], 138, 8, u(15))
	ep := entityFrom(t, s)
	sim, err := NewPoolSimulatorFromParams(pool.FactoryParams{EntityPool: ep, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	amount := u(1_000_000)
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Tokens[0].Address, Amount: amount.ToBig()}, TokenOut: s.Tokens[3].Address}
	for elapsed := int64(0); elapsed <= 4500; elapsed += 100 {
		block, age, contextErr := sim.quoteContext(sim.capturedAtMs + elapsed)
		require.Equal(t, elapsed, age)
		want := posted + uint64((elapsed+299)/300) + 1
		require.Equal(t, want, block)
		result, err := sim.calcAmountOut(params, sim.capturedAtMs+elapsed)
		if block-posted > 15 {
			require.ErrorIs(t, contextErr, ErrStaleSnapshot)
			require.ErrorIs(t, err, ErrStaleSnapshot)
			require.Nil(t, result)
			continue
		}
		require.NoError(t, contextErr)
		expected, expectedErr := s.QuoteExactInputAt(0, 3, amount, block)
		require.Equal(t, expectedErr, err)
		if err == nil {
			require.Equal(t, expected.AmountOut.ToBig(), result.TokenAmountOut.Amount)
		}
	}
	// The quote includes the next block: estimated head at the on-chain cutoff withdraws.
	_, _, err = sim.quoteContext(sim.capturedAtMs + 15*300)
	require.ErrorIs(t, err, ErrStaleSnapshot)
}

func TestFactoryRoutingExpiresSameInstanceAndIndexerSkipsAge(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := stateFrom(t, fixture(t, "deployed-model3.json.gz").Snapshots[0])
		s.Words[2] = setField(s.Words[2], 138, 8, u(15))
		ep := entityFrom(t, s)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(ep.Extra), &extra))
		extra.CapturedAtMs = time.Now().UnixMilli()
		data, err := json.Marshal(&extra)
		require.NoError(t, err)
		ep.Extra = string(data)
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Tokens[0].Address, Amount: big.NewInt(1_000_000)}, TokenOut: s.Tokens[3].Address}
		live, err := pool.Factory(DexType)(pool.FactoryParams{EntityPool: ep, Opts: pool.FactoryOpts{StaleCheck: true}})
		require.NoError(t, err)
		indexer, err := pool.Factory(DexType)(pool.FactoryParams{EntityPool: ep, Opts: pool.FactoryOpts{StaleCheck: false}})
		require.NoError(t, err)
		before, err := live.CalcAmountOut(params)
		require.NoError(t, err)
		require.True(t, before.IsValid())
		time.Sleep(5 * time.Second)
		_, err = live.CalcAmountOut(params)
		require.ErrorIs(t, err, ErrStaleSnapshot)
		_, err = indexer.CalcAmountOut(params)
		require.NoError(t, err)
		meta := live.GetMetaInfo("", "").(MetaInfo)
		require.Equal(t, int64(5000), meta.SnapshotAgeMs)
		require.Greater(t, meta.QuoteBlock, meta.BlockNumber)
	})
}

func TestCaptureTimeFallbackBackstopAndClockErrors(t *testing.T) {
	s := stateFrom(t, fixture(t, "deployed-model3.json.gz").Snapshots[0])
	s.Words[2] = setField(s.Words[2], 138, 8, u(255))
	for _, tc := range []struct {
		name                             string
		captured, timestamp, now, maxAge int64
		stale                            bool
	}{
		{"timestamp fallback fresh", 0, 100, 100000, 0, false},
		{"timestamp fallback old", 0, 100, 120000, 0, true},
		{"missing both", 0, 0, 100000, 0, true},
		{"future capture", 100001, 0, 100000, 0, true},
		{"default backstop", 100000, 0, 115001, 0, true},
		{"custom backstop", 100000, 0, 101001, 1000, true},
		{"backstop equality", 100000, 0, 101000, 1000, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ep := entityFrom(t, s)
			data, err := json.Marshal(&Extra{State: s, CapturedAtMs: tc.captured, MaxSnapshotAgeMs: tc.maxAge})
			require.NoError(t, err)
			ep.Extra, ep.Timestamp = string(data), tc.timestamp
			sim, err := NewPoolSimulator(ep)
			require.NoError(t, err)
			_, _, err = sim.quoteContext(tc.now)
			if tc.stale {
				require.ErrorIs(t, err, ErrStaleSnapshot)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIndexerSkipsExpiredOnChainAgeWithoutChangingState(t *testing.T) {
	s := stateFrom(t, fixture(t, "deployed-model3.json.gz").Snapshots[0])
	s.BlockNumber = s.small(2, 36, 32) + s.small(2, 138, 8) + 1
	ep := entityFrom(t, s)
	sim, err := NewPoolSimulatorFromParams(pool.FactoryParams{EntityPool: ep})
	require.NoError(t, err)
	before := sim.state
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Tokens[0].Address, Amount: big.NewInt(1_000_000)}, TokenOut: s.Tokens[3].Address}
	_, err = sim.calcAmountOut(params, 1000000000)
	require.NoError(t, err)
	require.Equal(t, before, sim.state)
}

func TestLensBlockTimeCannotFreshenAnOldProviderHead(t *testing.T) {
	s := stateFrom(t, fixture(t, "deployed-model3.json.gz").Snapshots[0])
	ep := entityFrom(t, s)
	data, err := json.Marshal(&Extra{State: s, CapturedAtMs: 100000, BlockTimestamp: 50})
	require.NoError(t, err)
	ep.Extra = string(data)
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	_, age, err := sim.quoteContext(100000)
	require.Equal(t, int64(49001), age)
	require.ErrorIs(t, err, ErrStaleSnapshot)
}

func TestAgingMatchesDeployedBytecodeThroughMaxAge15(t *testing.T) {
	compressed, err := os.ReadFile("testdata/lens-age15-monad.json.gz")
	require.NoError(t, err)
	z, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer z.Close()
	data, err := io.ReadAll(z)
	require.NoError(t, err)
	var fixture struct {
		Proxy           string `json:"proxy"`
		LensPayload     string `json:"lensPayload"`
		ControlOverride string `json:"controlOverride"`
		Cases           []struct {
			InputIndex  int    `json:"inputIndex"`
			OutputIndex int    `json:"outputIndex"`
			Amount      string `json:"amount"`
			ElapsedMs   int64  `json:"elapsedMs"`
			QuoteBlock  uint64 `json:"quoteBlock"`
			Result      string `json:"result"`
		} `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(data, &fixture))
	s, _, err := decodeSnapshotLens(raw(t, fixture.LensPayload), common.HexToAddress(fixture.Proxy))
	require.NoError(t, err)
	s.Words[2] = number(t, fixture.ControlOverride)
	sim, err := NewPoolSimulator(entityFrom(t, s))
	require.NoError(t, err)
	successes := 0
	for _, c := range fixture.Cases {
		amount, ok := new(big.Int).SetString(c.Amount, 10)
		require.True(t, ok)
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Tokens[c.InputIndex].Address, Amount: amount}, TokenOut: s.Tokens[c.OutputIndex].Address}
		block, _, _ := sim.quoteContext(sim.capturedAtMs + c.ElapsedMs)
		require.Equal(t, c.QuoteBlock, block)
		result, err := sim.calcAmountOut(params, sim.capturedAtMs+c.ElapsedMs)
		if c.Result == "" {
			require.Error(t, err)
			require.Nil(t, result)
			continue
		}
		require.NoError(t, err)
		want := new(big.Int).SetBytes(raw(t, c.Result)[:32])
		require.Equal(t, want, result.TokenAmountOut.Amount, "pair %d/%d block %d", c.InputIndex, c.OutputIndex, c.QuoteBlock)
		swap := result.SwapInfo.(SwapInfo)
		require.Equal(t, c.QuoteBlock, swap.QuoteBlock)
		require.Equal(t, c.ElapsedMs, swap.SnapshotAgeMs)
		successes++
	}
	require.Len(t, fixture.Cases, 32)
	require.GreaterOrEqual(t, successes, 15)
	t.Logf("Age-15 deployed-bytecode corpus: %d exact successes, %d matching rejections", successes, len(fixture.Cases)-successes)
}

// setField is limited to fixture policy controls; production snapshots are never rewritten.
func setField(word uint256.Int, shift, width uint, value uint256.Int) uint256.Int {
	var mask, bits uint256.Int
	mask.SetAllOne()
	mask.Rsh(&mask, 256-width)
	mask.Lsh(&mask, shift)
	mask.Not(&mask)
	word.And(&word, &mask)
	bits.Lsh(&value, shift)
	word.Or(&word, &bits)
	return word
}
