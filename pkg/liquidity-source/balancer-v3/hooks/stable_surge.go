package hooks

type StableSurgeHook struct {
	NoOpHook
}

func NewStableSurgeHook() *StableSurgeHook {
	return &StableSurgeHook{}
}
