package sd59x18

import "errors"

var (
	Err_PRBMath_SD59x18_Convert_Underflow = errors.New("PRBMath_SD59x18_Convert_Underflow")
	Err_PRBMath_SD59x18_Convert_Overflow  = errors.New("PRBMath_SD59x18_Convert_Overflow")
	Err_PRBMath_SD59x18_Log_InputTooSmall = errors.New("PRBMath_SD59x18_Log_InputTooSmall")
	Err_PRBMath_SD59x18_Exp2_InputTooBig  = errors.New("PRBMath_SD59x18_Exp2_InputTooBig")
	Err_PRBMath_SD59x18_Mul_InputTooSmall = errors.New("PRBMath_SD59x18_Mul_InputTooSmall")
	Err_PRBMath_SD59x18_Mul_Overflow      = errors.New("PRBMath_SD59x18_Mul_Overflow")
	Err_PRBMath_SD59x18_Div_InputTooSmall = errors.New("PRBMath_SD59x18_Div_InputTooSmall")
	Err_PRBMath_SD59x18_Div_Overflow      = errors.New("PRBMath_SD59x18_Div_Overflow")
)
