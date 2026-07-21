package ekubov3

const (
	ExtensionTypeUnknown ExtensionType = iota
	ExtensionTypeNoSwapCallPoints
	ExtensionTypeOracle
	ExtensionTypeTwamm
	ExtensionTypeMevCapture
	ExtensionTypeBoostedFeesConcentrated
	ExtensionTypeVe33
)

type ExtensionType int
