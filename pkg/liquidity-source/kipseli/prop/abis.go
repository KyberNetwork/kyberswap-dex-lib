package prop

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const lensSnapshotError = "KipseliPropSnapshot"

var (
	lensABI      abi.ABI
	lensBytecode []byte
)

func init() {
	var err error
	if lensABI, err = abi.JSON(bytes.NewReader(lensABIData)); err != nil {
		panic(err)
	}
	if lensBytecode, err = hexutil.Decode(strings.TrimSpace(lensBytecodeHex)); err != nil {
		panic(err)
	}
}
