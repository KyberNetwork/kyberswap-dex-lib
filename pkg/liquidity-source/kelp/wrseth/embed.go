package wrseth

import _ "embed"

var (
	//go:embed pools/avalanche.json
	avalanchePoolData []byte

	//go:embed pools/base.json
	basePoolData []byte

	//go:embed pools/linea.json
	lineaPoolData []byte

	//go:embed pools/optimism.json
	optimismPoolData []byte

	//go:embed pools/plasma.json
	plasmaPoolData []byte

	//go:embed pools/sonic.json
	sonicPoolData []byte
)

var BytesByPath = map[string][]byte{
	"pools/avalanche.json": avalanchePoolData,
	"pools/base.json":      basePoolData,
	"pools/linea.json":     lineaPoolData,
	"pools/optimism.json":  optimismPoolData,
	"pools/plasma.json":    plasmaPoolData,
	"pools/sonic.json":     sonicPoolData,
}
