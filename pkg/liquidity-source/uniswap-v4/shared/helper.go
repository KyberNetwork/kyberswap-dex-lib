package shared

var (
	MAX_FEE_PIPS     uint32 = 1000000 // 100%
	DYNAMIC_FEE_FLAG uint32 = 0x800000
)

func IsDynamicFee(fee uint32) bool {
	return fee == DYNAMIC_FEE_FLAG
}
