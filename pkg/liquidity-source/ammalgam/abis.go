package ammalgam

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var (
	factoryABI = lo.Must(abi.JSON(bytes.NewReader(factoryABIJSON)))
	pairABI    = lo.Must(abi.JSON(bytes.NewReader(pairABIJSON)))
	beaconABI  = lo.Must(abi.JSON(bytes.NewReader(beaconABIJSON)))
)
