package doppleramm

import _ "embed"

//go:embed abis/DopplerAMM.json
var ammABIData []byte

//go:embed abis/DopplerQuoter.json
var quoterABIData []byte

//go:embed abis/LaunchModule.json
var launchModuleABIData []byte
