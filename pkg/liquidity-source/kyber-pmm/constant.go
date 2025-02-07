package kyberpmm

type SwapDirection uint8

const (
	DexTypeKyberPMM = "kyber-pmm"

	PoolIDPrefix    = "kyber_pmm"
	PoolIDSeparator = "_"
)

var (
	DefaultGas = Gas{Swap: 100000}
)
