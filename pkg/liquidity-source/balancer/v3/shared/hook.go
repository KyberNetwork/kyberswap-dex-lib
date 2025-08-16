package shared

type HookType string

const (
	DirectionalFeeHookType   HookType = "DIRECTIONAL_FEE"
	FeeTakingHookType        HookType = "FEE_TAKING"
	StableSurgeHookType      HookType = "STABLE_SURGE"
	VeBALFeeDiscountHookType HookType = "VEBAL_DISCOUNT"
)

// Define a map of supported hooks
var hooksMap = map[HookType]bool{
	DirectionalFeeHookType:   true,
	FeeTakingHookType:        true,
	StableSurgeHookType:      true,
	VeBALFeeDiscountHookType: true,
}

func IsHookSupported(hook HookType) bool {
	if hook == "" {
		return true
	}
	_, ok := hooksMap[hook]
	return ok
}
