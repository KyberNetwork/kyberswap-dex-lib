package ekubov3

const (
	ExtensionTypeBase ExtensionType = iota + 1
	ExtensionTypeOracle
	ExtensionTypeTwamm
	ExtensionTypeMevCapture
)

type ExtensionType int
