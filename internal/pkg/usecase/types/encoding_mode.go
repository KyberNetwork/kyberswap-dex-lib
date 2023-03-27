package types

type EncodingMode string

const (
	EncodingModeSimple = "simple"
	EncodingModeNormal = "normal"
)

func (m EncodingMode) IsSimple() bool {
	return m == EncodingModeSimple
}
