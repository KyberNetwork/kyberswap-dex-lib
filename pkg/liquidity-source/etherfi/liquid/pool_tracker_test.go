package liquid

import (
	"math/big"
	"testing"
)

func TestFinalBlockNumber(t *testing.T) {
	tests := []struct {
		name          string
		first, second *big.Int
		want          uint64
	}{
		{name: "final second batch", first: big.NewInt(123), second: big.NewInt(456), want: 456},
		{name: "fallback to first batch", first: big.NewInt(123), want: 123},
		{name: "no response block", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := finalBlockNumber(tt.first, tt.second); got != tt.want {
				t.Fatalf("finalBlockNumber() = %d, want %d", got, tt.want)
			}
		})
	}
}
