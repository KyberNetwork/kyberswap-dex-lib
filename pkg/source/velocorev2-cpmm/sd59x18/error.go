package sd59x18

import "errors"

var (
	ErrMathSD59x18ConvertUnderflow = errors.New("Math_SD59x18_Convert_Underflow")
	ErrMathSD59x18ConvertOverflow  = errors.New("Math_SD59x18_Convert_Overflow")

	ErrMathSD59x18LogInputTooSmall = errors.New("Math_SD59x18_Log_InputTooSmall")
	ErrMathSD59x18Exp2InputTooBig  = errors.New("Math_SD59x18_Exp2_InputTooBig")

	ErrMathSD59x18MulInputTooSmall = errors.New("Math_SD59x18_Mul_InputTooSmall")
	ErrMathSD59x18MulOverflow      = errors.New("Math_SD59x18_Mul_Overflow")

	ErrMathMulDiv18Overflow = errors.New("Math_MulDiv18_Overflow")

	ErrMathSD59x18DivInputTooSmall = errors.New("Math_SD59x18_Div_InputTooSmall")
	ErrMathSD59x18DivOverflow   = errors.New("Math_SD59x18_Div_Overflow")

	ErrMathMulDivOverflow = errors.New("Math_MulDiv_Overflow")
)
