package hyperamm

import _ "embed"

//go:embed abi/HyperAMMFactory.json
var hyperAMMFactoryBytes []byte

//go:embed abi/HyperAMM.json
var hyperAMMBytes []byte

//go:embed abi/HyperAMMSwapFeeModule.json
var hyperAMMSwapFeeModuleBytes []byte

//go:embed abi/HyperAMMLens.json
var hyperAMMLensBytes []byte
