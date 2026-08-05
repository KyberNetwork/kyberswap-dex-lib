package tokentax

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	//go:embed abis/FeeOnTransferDetector.json
	detectorABIJSON []byte
	detectorABI     abi.ABI

	//go:embed abis/FeeOnTransferDetectorBasic.json
	detectorBasicABIJSON []byte
	detectorBasicABI     abi.ABI
)

func init() {
	var err error
	detectorABI, err = abi.JSON(bytes.NewReader(detectorABIJSON))
	if err != nil {
		panic(err)
	}
	detectorBasicABI, err = abi.JSON(bytes.NewReader(detectorBasicABIJSON))
	if err != nil {
		panic(err)
	}
}
