package umbraedlmm

import _ "embed"

//go:embed abi/LBPair.json
var pairABIJson []byte

//go:embed abi/LBFactory.json
var factoryABIJson []byte

//go:embed abi/PairViewer.json
var viewerABIJson []byte

//go:embed abi/LBRouter.json
var routerABIJson []byte
