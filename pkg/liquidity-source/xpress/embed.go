package xpress

import _ "embed"

//go:embed abis/OnchainClobHelper.json
var onchainClobHelperABIJson []byte

//go:embed abis/OnchainClob.json
var OnchainClobABIJson []byte
