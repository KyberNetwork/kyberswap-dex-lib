package customwrappednative

import _ "embed"

//go:embed pools/elysium.json
var elysiumPoolData []byte

var BytesByPath = map[string][]byte{
	"pools/elysium.json": elysiumPoolData,
}
