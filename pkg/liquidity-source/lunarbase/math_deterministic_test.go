package lunarbase

import (
	"bufio"
	"fmt"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

type deterministicVector struct {
	Name   string `json:"name"`
	Dir    string `json:"dir"`
	PX96   string `json:"pX96"`
	Fee    string `json:"fee"`
	ResX   string `json:"resX"`
	ResY   string `json:"resY"`
	K      uint32 `json:"k"`
	Dx     string `json:"dx,omitempty"`
	Dy     string `json:"dy,omitempty"`
	PNext  string `json:"pNext"`
	FeeAmt string `json:"feeAmt"`
}

func TestAllDeterministicVectors(t *testing.T) {
	f, err := os.Open("testdata/deterministic_vectors.jsonl")
	require.NoError(t, err)
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	var (
		total     int
		xToYCount int
		yToXCount int
		failures  []string
	)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for lineNum := 1; scanner.Scan(); lineNum++ {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var v deterministicVector
		require.NoError(t, json.Unmarshal([]byte(line), &v), "parse error line %d", lineNum)
		total++

		params := &PoolParams{
			SqrtPriceX96:   u(v.PX96),
			FeeQ48:         u(v.Fee).Uint64(),
			ReserveX:       u(v.ResX),
			ReserveY:       u(v.ResY),
			ConcentrationK: v.K,
		}

		if v.Dir == "xToY" {
			xToYCount++
			result := quoteXToY(params, u(v.Dx))

			if result.AmountOut.Dec() != v.Dy ||
				result.SqrtPriceNext.Dec() != v.PNext ||
				result.Fee.Dec() != v.FeeAmt {
				failures = append(failures, fmt.Sprintf(
					"%s line %d: xToY MISMATCH\n  dy:    got %s expected %s\n  pNext: got %s expected %s\n  fee:   got %s expected %s",
					v.Name, lineNum, result.AmountOut.Dec(), v.Dy,
					result.SqrtPriceNext.Dec(), v.PNext,
					result.Fee.Dec(), v.FeeAmt,
				))
			}
		} else {
			yToXCount++
			result := quoteYToX(params, u(v.Dy))

			if result.AmountOut.Dec() != v.Dx ||
				result.SqrtPriceNext.Dec() != v.PNext ||
				result.Fee.Dec() != v.FeeAmt {
				failures = append(failures, fmt.Sprintf(
					"%s line %d: yToX MISMATCH\n  dx:    got %s expected %s\n  pNext: got %s expected %s\n  fee:   got %s expected %s",
					v.Name, lineNum, result.AmountOut.Dec(), v.Dx,
					result.SqrtPriceNext.Dec(), v.PNext,
					result.Fee.Dec(), v.FeeAmt,
				))
			}
		}
	}
	require.NoError(t, scanner.Err())

	t.Logf("Deterministic vectors: %d total (%d xToY, %d yToX)", total, xToYCount, yToXCount)

	if len(failures) > 0 {
		show := failures
		if len(show) > 20 {
			show = show[:20]
		}
		for i, f := range show {
			t.Logf("[%d] %s", i+1, f)
		}
		if len(failures) > 20 {
			t.Logf("... and %d more", len(failures)-20)
		}
		t.Fatalf("%d out of %d deterministic vectors failed", len(failures), total)
	}

	t.Logf("ALL %d deterministic vectors passed!", total)
}
