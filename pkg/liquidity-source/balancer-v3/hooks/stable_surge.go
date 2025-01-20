package hooks

const StableSurgeHookType = "StableSurgeHook"

type StableSurgeHook struct {
	BaseHook
}

func NewStableSurgeHook() *StableSurgeHook {
	return &StableSurgeHook{}
}
