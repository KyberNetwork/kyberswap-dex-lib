package wombat

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/dsmath"
	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/holiman/uint256"
)

var (
	ErrCoreUnderflow = errors.New("core underflow")
)

var CoreV2 *coreV2

type coreV2 struct {
	WADI *big.Int
}

func init() {
	CoreV2 = &coreV2{
		WADI: integer.TenPow(18),
	}
}

// _swapQuoteFunc
// https://github.com/wombat-exchange/v1-core/blob/d9bb07de2272e0d9cb4c640671a925896b64fc14/contracts/wombat-core/pool/CoreV2.sol#L31
func (l *coreV2) _swapQuoteFunc(ax, ay, lx, ly, dx, a *big.Int) (*uint256.Int, error) {
	if lx.Cmp(integer.Zero()) == 0 || ly.Cmp(integer.Zero()) == 0 {
		return nil, ErrCoreUnderflow
	}
	d := new(big.Int).Sub(
		new(big.Int).Add(ax, ay),
		dsmath.WMul(
			a,
			new(big.Int).Add(
				new(big.Int).Div(new(big.Int).Mul(lx, lx), ax),
				new(big.Int).Div(new(big.Int).Mul(ly, ly), ay)),
		),
	)
	rx := dsmath.WDiv(new(big.Int).Add(ax, dx), lx)
	b := new(big.Int).Sub(
		new(big.Int).Div(
			new(big.Int).Mul(
				lx,
				new(big.Int).Sub(
					rx,
					dsmath.WDiv(a, rx)),
			),
			ly,
		),
		dsmath.WDiv(d, ly),
	)
	ry := l._solveSquad(b, a)
	dy := new(big.Int).Sub(dsmath.WMul(ly, ry), ay)
	if dy.Cmp(integer.Zero()) < 0 {
		return uint256.MustFromBig(new(big.Int).Neg(dy)), nil
	} else {
		return uint256.MustFromBig(dy), nil
	}
}

// _solveSquad
// https://github.com/wombat-exchange/v1-core/blob/d9bb07de2272e0d9cb4c640671a925896b64fc14/contracts/wombat-core/pool/CoreV2.sol#L62
func (l *coreV2) _solveSquad(b, c *big.Int) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Sub(
			SignedSafeMath.sqrt(
				new(big.Int).Add(
					new(big.Int).Mul(b, b),
					new(big.Int).Mul(new(big.Int).Mul(c, integer.Four()), l.WADI)),
				b,
			),
			b,
		),
		integer.Two(),
	)
}
