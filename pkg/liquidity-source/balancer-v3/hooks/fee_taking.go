package hooks

type FeeTakingHook struct {
	NoOpHook
}

func NewFeeTakingHook() *FeeTakingHook {
	return &FeeTakingHook{}
}
