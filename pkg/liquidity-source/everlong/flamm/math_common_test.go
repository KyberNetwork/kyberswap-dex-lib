package everlongflamm

import (
	"testing"

	"github.com/holiman/uint256"
)

func TestMathCommon(t *testing.T) {
	t.Parallel()
	z, o := mulDivCeil(uint256.NewInt(7), uint256.NewInt(3), uint256.NewInt(2))
	if o || z.Uint64() != 11 {
		t.Fatal(z, o)
	}
	z, o = mulDivCeil(uint256.NewInt(8), uint256.NewInt(3), uint256.NewInt(2))
	if o || z.Uint64() != 12 {
		t.Fatal(z, o)
	}
	_, o = mulDiv(maxUint256, maxUint256, uint256.NewInt(1))
	if !o {
		t.Fatal("expected overflow")
	}
	// One sentinel per custom error of the deployed c104 contracts (errors.go): 139 declared under src/core,
	// src/factory, src/hooks, src/interfaces and src/libraries at 80abd43, plus the eight OpenZeppelin ERC20 and
	// Initializable errors. Every selector maps to a distinct sentinel, and every sentinel is reachable from one.
	if len(revertSelectors) != 147 {
		t.Fatal(len(revertSelectors))
	}
	seen := make(map[error]bool, len(revertSelectors))
	for sel, err := range revertSelectors {
		if err == nil || seen[err] {
			t.Fatal(sel, err)
		}
		seen[err] = true
	}
}
