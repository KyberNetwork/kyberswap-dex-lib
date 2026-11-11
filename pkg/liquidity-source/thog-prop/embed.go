package thogprop

import _ "embed"

//go:embed abi/ThogAMM.json
var thogAMMBytes []byte

// ThogSnapshotLens.sol: solc 0.8.25, viaIR, shanghai, optimizer 200, no metadata.
//
//go:embed abi/ThogSnapshotLens.json
var snapshotLensABIBytes []byte

//go:embed abi/ThogSnapshotLens.bin
var snapshotLensBytecodeHex string
