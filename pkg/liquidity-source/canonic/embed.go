package canonic

import _ "embed"

//go:embed abi/Pool.json
var poolABIJson []byte

//go:embed abi/Previewer.json
var previewerABIJson []byte
