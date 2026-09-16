package lunarbase

import (
	"errors"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestLegacyJSONUnknownZeroModelRequiresOnlyPoolRefresh(t *testing.T) {
	for _, kind := range []string{"unknown", "verified-punishment-zero", "explicit-legacy-zero"} {
		t.Run(kind, func(t *testing.T) {
			p := verifiedEntity()
			e := Extra{SqrtPriceX96: new(uint256.Int).Set(q96), LatestUpdateBlock: 800, BlockDelay: 25}
			if kind == "verified-punishment-zero" {
				e.BlockHash = "test-snapshot"
			}
			if kind == "explicit-legacy-zero" {
				e.ConcentrationModel = true
			}
			b, err := json.Marshal(e)
			if err != nil {
				t.Fatal(err)
			}
			p.Extra = string(b)
			s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p, ChainID: valueobject.ChainIDBSC})
			if err != nil {
				t.Fatal("legacy entity itself must remain readable:", err)
			}
			for _, sim := range []pool.IPoolSimulator{s, s.CloneState()} {
				_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(1)}, TokenOut: p.Tokens[1].Address})
				if kind == "unknown" {
					if !errors.Is(err, ErrStalePool) {
						t.Fatalf("ambiguous model quoted: %v", err)
					}
				} else if err != nil {
					t.Fatalf("explicit zero model rejected: %v", err)
				}
			}
		})
	}
}
