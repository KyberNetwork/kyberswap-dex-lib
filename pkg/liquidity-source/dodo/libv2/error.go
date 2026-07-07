package libv2

import "errors"

var (
	ErrMulError        = errors.New("MUL_ERROR")
	ErrDividingError   = errors.New("DIVIDING_ERROR")
	ErrSubError        = errors.New("SUB_ERROR")
	ErrAddError        = errors.New("ADD_ERROR")
	ErrTargetIsZero    = errors.New("TARGET_IS_ZERO")
	ErrShouldNotBeZero = errors.New("DODOMath: should not be zero")
)
