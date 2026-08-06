package everlongclamm

import _ "embed"

//go:embed abi/CLPoolManager.json
var poolManagerABIJson []byte

//go:embed abi/ClammALM.json
var almABIJson []byte
