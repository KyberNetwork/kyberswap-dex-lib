package dexv2

import (
	"errors"
	"testing"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestUnsupportedFeesAndControllers(t *testing.T) {
	cases := []struct {
		name      string
		simulator PoolSimulator
		want      error
	}{
		{"fee version", PoolSimulator{unsupportedDynamicFee: true}, ErrUnsupportedDynamicFee},

		{"controller", PoolSimulator{staticExtra: StaticExtra{Controller: "0x0000000000000000000000000000000000000001"}}, ErrUnsupportedController},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.simulator.CalcAmountOut(pool.CalcAmountOutParams{})
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestNetSwapInStep(t *testing.T) {
	var net, protocol, lp uint256.Int
	netSwapInStep(&net, &protocol, &lp, uint256.NewInt(1000000), uint256.NewInt(1000), uint256.NewInt(100))
	if net.Uint64() != 998899 || protocol.Uint64() != 1001 || lp.Uint64() != 100 {
		t.Fatalf("net %s protocol %s lp %s", &net, &protocol, &lp)
	}
}

func TestPackSqrtPrice(t *testing.T) {
	p := new(uint256.Int).Lsh(uint256.NewInt(1), 96)
	p.AddUint64(p, 12345)
	var down, up uint256.Int
	packSqrtPrice(&down, p, false)
	packSqrtPrice(&up, p, true)
	if down.Cmp(p) > 0 || up.Cmp(p) <= 0 {
		t.Fatalf("down %s up %s price %s", &down, &up, p)
	}
	var step uint256.Int
	step.Sub(&up, &down)
	if step.Uint64() != 1<<33 {
		t.Fatalf("step %s", &step)
	}
}
