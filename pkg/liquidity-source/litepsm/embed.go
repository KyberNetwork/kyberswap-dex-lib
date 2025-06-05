package litepsm

import _ "embed"

//go:embed dexconfig/ethereum/DaiLitePSM.json
var ethereumDaiLitePSMBytes []byte

//go:embed dexconfig/ethereum/UsdsLitePSM.json
var ethereumUsdsLitePSMBytes []byte

var bytesByPath = map[string][]byte{
	"dexconfig/ethereum/DaiLitePSM.json":  ethereumDaiLitePSMBytes,
	"dexconfig/ethereum/UsdsLitePSM.json": ethereumUsdsLitePSMBytes,
}

//go:embed abis/LitePSM.json
var litePSMBytes []byte
