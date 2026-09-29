package ponsv2

import _ "embed"

//go:embed abis/Curve.json
var curveABIBytes []byte

//go:embed abis/Factory.json
var factoryABIBytes []byte
