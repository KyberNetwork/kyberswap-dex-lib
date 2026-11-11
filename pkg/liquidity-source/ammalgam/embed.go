package ammalgam

import _ "embed"

//go:embed abi/AmmalgamFactory.json
var factoryABIJSON []byte

//go:embed abi/AmmalgamPair.json
var pairABIJSON []byte

//go:embed abi/Beacon.json
var beaconABIJSON []byte
