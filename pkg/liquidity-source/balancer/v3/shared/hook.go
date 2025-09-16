package shared

type HookType string

const (
	DirectionalFeeHookType   HookType = "DIRECTIONAL_FEE"
	FeeTakingHookType        HookType = "FEE_TAKING"
	StableSurgeHookType      HookType = "STABLE_SURGE"
	VeBALFeeDiscountHookType HookType = "VEBAL_DISCOUNT"
	// reCLAMM pools are also their own hook. The hook only affects add/remove function, not swap.
	// https://docs.balancer.fi/concepts/explore-available-balancer-pools/reclamm-pool/reclamm-pool-math.html#on-add-remove-liquidity
	ReClammHookType HookType = "RECLAMM"
)

// Define a map of supported hooks
var hooksMap = map[HookType]bool{
	DirectionalFeeHookType:   true,
	FeeTakingHookType:        true,
	StableSurgeHookType:      true,
	VeBALFeeDiscountHookType: true,
	ReClammHookType:          true,
}

func IsHookSupported(hook HookType) bool {
	if hook == "" {
		return true
	}
	_, ok := hooksMap[hook]
	return ok
}
