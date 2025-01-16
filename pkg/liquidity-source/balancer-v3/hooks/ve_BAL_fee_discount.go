package hooks

const VeBALFeeDiscountHookType = "VeBALFeeDiscountHook"

type VeBALFeeDiscountHook struct {
	BaseHook
}

func NewVeBALFeeDiscountHook() *VeBALFeeDiscountHook {
	return &VeBALFeeDiscountHook{}
}
