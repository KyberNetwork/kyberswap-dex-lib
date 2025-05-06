package ekubo

type ExtensionType int

const (
	ExtensionTypeBase ExtensionType = iota + 1
	ExtensionTypeOracle
	ExtensionTypeTwamm
)
