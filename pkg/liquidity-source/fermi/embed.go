package fermi

import _ "embed"

//go:embed abi/FermiSwapper.json
var fermiSwapperABIJson []byte

//go:embed abi/FermiEngine.json
var fermiEngineABIJson []byte
