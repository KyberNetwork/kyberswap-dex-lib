package balancerv1

import (
	"errors"
	"fmt"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"
)

var (
	ErrDivZero         = errors.New("ERR_DIV_ZERO")
	ErrDivInternal     = errors.New("ERR_DIV_INTERNAL")
	ErrSubUnderflow    = errors.New("ERR_SUB_UNDERFLOW")
	ErrMulOverflow     = errors.New("ERR_MUL_OVERFLOW")
	ErrAddOverFlow     = errors.New("ERR_ADD_OVERFLOW")
	ErrBPowBaseTooLow  = errors.New("ERR_BPOW_BASE_TOO_LOW")
	ErrBPowBaseTooHigh = errors.New("ERR_BPOW_BASE_TOO_HIGH")
)

var BNum *bNum

type bNum struct {
}

func init() {
	BNum = &bNum{}
	fmt.Println("wtf bedug bnum")
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BNum.sol#L20
func (l *bNum) BToI(a *uint256.Int) *uint256.Int {
	return new(uint256.Int).Div(a, BConst.BONE)
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BNum.sol#L27
func (l *bNum) BFloor(a *uint256.Int) *uint256.Int {
	return new(uint256.Int).Mul(l.BToI(a), BConst.BONE)
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BNum.sol#L34
func (l *bNum) BAdd(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	c := new(uint256.Int).Add(a, b)

	if c.Lt(a) {
		return nil, ErrAddOverFlow
	}

	return c, nil
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BNum.sol#L43
func (l *bNum) BSub(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	c, flag := l.BSubSign(a, b)

	if flag {
		return nil, ErrSubUnderflow
	}

	return c, nil
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BNum.sol#L52
func (l *bNum) BSubSign(a *uint256.Int, b *uint256.Int) (*uint256.Int, bool) {
	if !a.Lt(b) {
		return new(uint256.Int).Sub(a, b), false
	}

	return new(uint256.Int).Sub(b, a), true
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BNum.sol#L63
func (l *bNum) BMul(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	c0 := new(uint256.Int).Mul(a, b)

	if !a.Eq(number.Zero) && !new(uint256.Int).Div(c0, a).Eq(b) {
		return nil, ErrMulOverflow
	}

	c1 := new(uint256.Int).Add(c0, new(uint256.Int).Div(BConst.BONE, number.Number_2))

	if c1.Lt(c0) {
		return nil, ErrMulOverflow
	}

	c2 := new(uint256.Int).Div(c1, BConst.BONE)

	return c2, nil
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BNum.sol#L75
func (l *bNum) BDiv(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	if b.Eq(number.Zero) {
		return nil, ErrDivZero
	}

	c0 := new(uint256.Int).Mul(a, BConst.BONE)

	if !a.Eq(number.Zero) && !new(uint256.Int).Div(c0, a).Eq(BConst.BONE) {
		return nil, ErrDivInternal
	}

	c1 := new(uint256.Int).Add(c0, new(uint256.Int).Div(b, number.Number_2))

	if c1.Lt(c0) {
		return nil, ErrDivInternal
	}

	c2 := new(uint256.Int).Div(c1, b)

	return c2, nil
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BNum.sol#L89
func (l *bNum) BPowI(a *uint256.Int, n *uint256.Int) (*uint256.Int, error) {
	var (
		z   *uint256.Int
		err error
	)

	if !new(uint256.Int).Mod(n, number.Number_2).Eq(number.Zero) {
		z = a
	} else {
		z = BConst.BONE
	}

	for n = new(uint256.Int).Div(n, number.Number_2); !n.Eq(number.Zero); n = new(uint256.Int).Div(n, number.Number_2) {
		a, err = l.BMul(a, a)
		if err != nil {
			return nil, err
		}

		if !new(uint256.Int).Mod(n, number.Number_2).Eq(number.Zero) {
			z, err = l.BMul(z, a)
			if err != nil {
				return nil, err
			}
		}
	}

	return z, nil
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BNum.sol#L128C14-L128C24
func (l *bNum) BPowApprox(base *uint256.Int, exp *uint256.Int, precision *uint256.Int) (*uint256.Int, error) {
	logger.WithFields(logger.Fields{
		"base":      base,
		"exp":       exp,
		"precision": precision,
	}).Infof("BPowApprox Input data")
	fmt.Println("BPowApprox Input data", base, exp, precision)

	a := new(uint256.Int).Set(exp)
	x, xneg := l.BSubSign(base, BConst.BONE)
	term := new(uint256.Int).Set(BConst.BONE)
	sum := new(uint256.Int).Set(term)
	negative := false

	for i := number.Number_1; !term.Lt(precision); i = new(uint256.Int).Add(i, number.Number_1) {
		fmt.Println("counter i ", i)
		bigK := new(uint256.Int).Mul(i, BConst.BONE)

		bsubBigKAndBone, err := l.BSub(bigK, BConst.BONE)
		if err != nil {
			return nil, err
		}

		c, cneg := l.BSubSign(a, bsubBigKAndBone)

		bmulCAndX, err := l.BMul(c, x)
		if err != nil {
			return nil, err
		}

		term, err := l.BMul(term, bmulCAndX)
		if err != nil {
			return nil, err
		}

		term, err = l.BDiv(term, bigK)
		if err != nil {
			return nil, err
		}

		if term.Eq(number.Zero) {
			break
		}

		if xneg {
			negative = !negative
		}

		if cneg {
			negative = !negative
		}

		if negative {
			sum, err = l.BSub(sum, term)
			if err != nil {
				return nil, err
			}
		} else {
			sum, err = l.BAdd(sum, term)
			if err != nil {
				return nil, err
			}
		}
	}

	return sum, nil
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BNum.sol#L108
func (l *bNum) BPow(base *uint256.Int, exp *uint256.Int) (*uint256.Int, error) {
	if base.Lt(BConst.MIN_BPOW_BASE) {
		return nil, ErrBPowBaseTooLow
	}

	if base.Gt(BConst.MAX_BPOW_BASE) {
		return nil, ErrBPowBaseTooHigh
	}

	whole := l.BFloor(exp)
	remain, err := l.BSub(exp, whole)
	if err != nil {
		return nil, err
	}

	wholePow, err := l.BPowI(base, l.BToI(whole))
	if err != nil {
		return nil, err
	}

	if remain.Eq(number.Zero) {
		return wholePow, nil
	}

	partialResult, err := l.BPowApprox(base, remain, BConst.BPOW_PRECISION)
	if err != nil {
		fmt.Println("BPowApprox err", err)
		return nil, err
	}

	return l.BMul(wholePow, partialResult)
}
