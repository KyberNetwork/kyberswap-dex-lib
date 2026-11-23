package stableng

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type wRow struct {
	Pool  int
	Kind  string
	Input string
	Dy    string
	DyFee string
	Mint  string
	Err   string
	Site  string
}

// TestCaptureWithdrawNative runs the withdraw probe against the working tree. Inputs match the legacy capture.
func TestCaptureWithdrawNative(t *testing.T) {
	outPath := os.Getenv("CAPTURE_WITHDRAW_NATIVE")
	if outPath == "" {
		t.Skip("set CAPTURE_WITHDRAW_NATIVE=<path>")
	}
	var rows []wRow
	for pi, poolJSON := range dCacheTestPools {
		for n := range 4000 {
			rng := rand.New(rand.NewPCG(uint64(pi+1), uint64(n+1)))
			var e entity.Pool
			if err := json.Unmarshal([]byte(poolJSON), &e); err != nil {
				t.Fatal(err)
			}
			s, err := NewPoolSimulator(e)
			if err != nil {
				t.Fatal(err)
			}
			var amt uint256.Int
			amt.SetUint64(rng.Uint64())
			amt.Lsh(&amt, uint(rng.IntN(80)))
			i := rng.IntN(s.NumTokens)
			var dy, dyFee uint256.Int
			row := wRow{Pool: pi, Kind: "W1", Input: amt.Dec()}
			err = s.CalculateWithdrawOneCoinU256(&amt, i, &dy, &dyFee)
			if err != nil {
				row.Err = err.Error()
			} else {
				row.Dy, row.DyFee = dy.Dec(), dyFee.Dec()
			}
			_ = fmt.Sprint
			rows = append(rows, row)
		}
	}
	b, _ := json.MarshalIndent(rows, "", " ")
	if err := os.WriteFile(outPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
