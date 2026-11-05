package uniswapv3

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

func benchPayload(b *testing.B, n int) []byte {
	var raw struct{ ABI json.RawMessage }
	if err := json.Unmarshal(lensJSON, &raw); err != nil {
		b.Fatal(err)
	}
	parsed, err := abi.JSON(bytes.NewReader(raw.ABI))
	if err != nil {
		b.Fatal(err)
	}
	type pt struct {
		Tick           *big.Int
		LiquidityNet   *big.Int
		LiquidityGross *big.Int
	}
	ticks := make([]pt, n)
	for i := range ticks {
		ticks[i] = pt{big.NewInt(int64(i*60 - n*30)), big.NewInt(int64(1e15 + i)), big.NewInt(int64(2e15 + i))}
	}
	out, err := parsed.Methods["getSnapshot"].Outputs.Pack(big.NewInt(60), new(big.Int).Lsh(big.NewInt(1), 96),
		big.NewInt(100), big.NewInt(1e18), big.NewInt(20_000_000), int16(5), true, ticks)
	if err != nil {
		b.Fatal(err)
	}
	return out
}

func BenchmarkLensUnpack(b *testing.B) {
	payload := benchPayload(b, 5000)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for b.Loop() {
		var out ephemeral.TickSnapshot
		if err := lens.UnpackCall(payload, &out); err != nil {
			b.Fatal(err)
		}
	}
}
