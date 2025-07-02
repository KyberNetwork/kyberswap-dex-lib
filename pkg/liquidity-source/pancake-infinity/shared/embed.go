package shared

import _ "embed"

//go:embed abi/Quoter.json
var quoterABIJson []byte

//go:embed abi/BinPoolManager.json
var binPoolManagerABIJson []byte

//go:embed abi/CLPoolManager.json
var clPoolManagerABIJson []byte
