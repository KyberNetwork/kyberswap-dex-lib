package frxeth

import _ "embed"

//go:embed pools/ethereum.json
var ethereumPoolData []byte

var BytesByPath = map[string][]byte{
	"pools/ethereum.json": ethereumPoolData,
}
