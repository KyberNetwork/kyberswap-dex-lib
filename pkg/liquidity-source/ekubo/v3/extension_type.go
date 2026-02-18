package ekubov3

const (
	ExtensionTypeUnknown ExtensionType = iota
	ExtensionTypeNoSwapCallPoints
	ExtensionTypeOracle
	ExtensionTypeTwamm
	ExtensionTypeMevCapture
	ExtensionTypeBoostedFeesConcentrated
)

type ExtensionType int
