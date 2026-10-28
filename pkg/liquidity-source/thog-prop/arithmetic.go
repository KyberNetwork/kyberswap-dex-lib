package thogprop

import (
	"github.com/KyberNetwork/int256"
	v3utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

// A calculation records the first EVM arithmetic failure. Callers check err
// before exposing a result; no wrapping value can become a successful quote.
type arithmetic struct{ err error }

func u(n uint64) uint256.Int { return uint256.Int{n, 0, 0, 0} }

var powers = func() [55]uint256.Int {
	var p [55]uint256.Int
	p[0] = u(1)
	ten := u(10)
	for i := 1; i < len(p); i++ {
		p[i].Mul(&p[i-1], &ten)
	}
	return p
}()
var maxNotional = *uint256.MustFromDecimal("664619068552888026590626425199190042")
var maxInventory = uint256.Int{^uint64(0), ^uint64(0), 0, 0}
var maxUint = uint256.Int{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)}
var sqrt2e54 = *uint256.MustFromDecimal("1414213562373095048801688725")

func (m *arithmetic) add(a, b uint256.Int) (z uint256.Int) {
	if _, o := z.AddOverflow(&a, &b); o {
		m.err = ErrOverflow
	}
	return
}
func (m *arithmetic) sub(a, b uint256.Int) (z uint256.Int) {
	if _, o := z.SubOverflow(&a, &b); o {
		m.err = ErrOverflow
	}
	return
}
func (m *arithmetic) mul(a, b uint256.Int) (z uint256.Int) {
	if _, o := z.MulOverflow(&a, &b); o {
		m.err = ErrOverflow
	}
	return
}
func (m *arithmetic) div(a, b uint256.Int) (z uint256.Int) {
	if b.IsZero() {
		m.err = ErrOverflow
		return
	}
	z.Div(&a, &b)
	return
}
func (m *arithmetic) md(a, b, d uint256.Int, up bool) (z uint256.Int) {
	if d.IsZero() {
		m.err = ErrOverflow
		return
	}
	if up {
		if err := v3utils.MulDivRoundingUpV2(&a, &b, &d, &z); err != nil {
			m.err = ErrOverflow
		}
	} else if _, o := z.MulDivOverflow(&a, &b, &d); o {
		m.err = ErrOverflow
	}
	return
}
func si(n int64) (z int256.Int) { z.SetInt64(n); return }
func (m *arithmetic) signed(a uint256.Int) int256.Int {
	if a[3]>>63 != 0 {
		m.err = ErrOverflow
	}
	return int256.Int(a)
}
func (m *arithmetic) ia(a, b int256.Int) (z int256.Int) {
	if _, o := z.AddOverflow(&a, &b); o {
		m.err = ErrOverflow
	}
	return
}
func (m *arithmetic) im(a, b int256.Int) (z int256.Int) {
	if _, o := z.MulOverflow(&a, &b); o {
		m.err = ErrOverflow
	}
	return
}
func (m *arithmetic) id(a, b int256.Int) (z int256.Int) {
	if b.IsZero() || (a.IsMinI256() && b == si(-1)) {
		m.err = ErrOverflow
		return
	}
	z.Quo(&a, &b)
	return
}
func (m *arithmetic) abs(a int256.Int) uint256.Int {
	if a.IsMinI256() {
		m.err = ErrOverflow
	}
	if a.IsNegative() {
		a.Neg(&a)
	}
	return uint256.Int(a)
}
func min(a, b uint256.Int) uint256.Int {
	if a.Lt(&b) {
		return a
	}
	return b
}
func max(a, b uint256.Int) uint256.Int {
	if a.Gt(&b) {
		return a
	}
	return b
}
func bits(word *uint256.Int, shift, width uint) (z uint256.Int) {
	var mask uint256.Int
	mask.SetAllOne()
	mask.Rsh(&mask, 256-width)
	z.Rsh(word, shift)
	z.And(&z, &mask)
	return
}
