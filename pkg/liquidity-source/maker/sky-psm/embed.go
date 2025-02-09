package skypsm

import _ "embed"

//go:embed pools/base.json
var basePoolData []byte

var BytesByPath = map[string][]byte{
	"pools/base.json": basePoolData,
}
