package hooks

type VeBALFeeDiscountHook struct {
	NoOpHook
}

func NewVeBALFeeDiscountHook() *VeBALFeeDiscountHook {
	return &VeBALFeeDiscountHook{}
}
