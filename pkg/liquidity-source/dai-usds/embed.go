package dai_usds

import _ "embed"

//go:embed pools/ethereum.json
var ethereumPoolData []byte

var bytesByPath = map[string][]byte{
	"pools/ethereum.json": ethereumPoolData,
}
